package forum

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

type fakeForumPrefixService struct {
	ForumService
	prefix string
}

func (f fakeForumPrefixService) SectionPathPrefix(context.Context) string { return f.prefix }

// TestForumSectionFallbackSkipsOtherPaths - the fallback takes only
// paths under the forum vocabulary's prefix; everything else (and every
// path when no prefix is set) goes on to the next fallback untouched.
func TestForumSectionFallbackSkipsOtherPaths(t *testing.T) {
	for _, tc := range []struct {
		prefix, path string
	}{
		{"forums", "/articles/foo"},
		{"forums", "/forumsx/foo"},
		{"forums", "/forums"},
		{"", "/forums/foo"},
		{"forum", "/forum/foo"},
	} {
		c := &ForumController{forumService: fakeForumPrefixService{prefix: tc.prefix}}
		w := httptest.NewRecorder()
		if c.SectionFallback(w, httptest.NewRequest(http.MethodGet, tc.path, nil)) {
			t.Errorf("prefix %q, path %q: taken", tc.prefix, tc.path)
		}
		if w.Body.Len() != 0 {
			t.Errorf("prefix %q, path %q: wrote a response", tc.prefix, tc.path)
		}
	}
}

// fakeTopics - the ForumService calls of the topic handlers.
type fakeTopics struct {
	ForumService
	created, updated []string
	sticky           []string
	err              error
}

func (f *fakeTopics) SetTopicSticky(_ context.Context, topic modulekit.Node, sticky bool) error {
	f.sticky = append(f.sticky, fmt.Sprintf("%d:%v", topic.ID, sticky))
	return f.err
}

// GetForumTermsForSelect - the forum's terms: category 3 with section 4.
func (f *fakeTopics) GetForumTermsForSelect(context.Context) ([]ForumTermOption, error) {
	return []ForumTermOption{{ID: 3, Name: "General"}, {ID: 4, Name: "Talk", ParentID: 3}}, nil
}

func (f *fakeTopics) CreateTopic(_ context.Context, sectionID int, title, body string, authorID int) (*ForumTopicDetail, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.created = append(f.created, title)
	return &ForumTopicDetail{Slug: "new-topic"}, nil
}

func (f *fakeTopics) UpdateTopic(_ context.Context, nodeID int, title, body string, sectionID int) error {
	if f.err != nil {
		return f.err
	}
	f.updated = append(f.updated, title)
	return nil
}

// fakeNodeDelete - the delete of a node (testkit.NodeDeleter's deleteNode).
type fakeNodeDelete struct{ deleted []int }

func (f *fakeNodeDelete) DeleteNode(_ context.Context, id int) error {
	f.deleted = append(f.deleted, id)
	return nil
}

// topicController - the forum controller with the signed-in user (0 -
// none) allowed or not to change the topics, its topic node 7 and an
// article node 8.
func topicController(t *testing.T, userID int, allow bool, topics *fakeTopics, deletes *fakeNodeDelete) (*ForumController, []*http.Cookie, string) {
	t.Helper()
	return topicControllerAuth(t, testkit.Auth{UserID: userID, Allow: allow}, topics, deletes)
}

// topicControllerAuth - topicController with the given permission checks.
func topicControllerAuth(t *testing.T, auth testkit.Auth, topics *fakeTopics, deletes *fakeNodeDelete) (*ForumController, []*http.Cookie, string) {
	t.Helper()
	testkit.UseEngineRoot(t)
	store, cookies, token := testkit.Session(t)
	nodes := testkit.Nodes{Known: map[int]*modulekit.Node{
		7: {ID: 7, Type: manifest.ContentType, Slug: "forum/old-topic"},
		8: {ID: 8, Type: "article", Slug: "articles/a"},
	}}
	deps := modulekit.BaseDeps{
		Auth:      auth,
		Users:     testkit.Users{Known: map[int]*modulekit.User{3: {ID: 3, Username: "ann"}}},
		Templates: testkit.Templates(),
		Store:     store,
		Logger:    testkit.Logger(),
		Nodes:     nodes,
	}
	deleter := testkit.NodeDeleter(nodes, deletes.DeleteNode)
	return NewForumController(deps, ForumDeps{ForumService: topics, NodeDeleter: deleter}), cookies, token
}

func serve(h http.HandlerFunc, req *http.Request, id string) *httptest.ResponseRecorder {
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// TestCreateTopicSubmit - a signed-in user with a valid form creates the
// topic and goes to it; a visitor is sent to sign in; a user without the
// right, a form without a title, with no section, one outside the forum
// or a category creates nothing (an expired token - the site's CSRF
// middleware).
func TestCreateTopicSubmit(t *testing.T) {
	topics := &fakeTopics{}
	c, cookies, token := topicController(t, 3, true, topics, &fakeNodeDelete{})
	denied, _, _ := topicController(t, 3, false, topics, &fakeNodeDelete{})
	form := func(v url.Values) url.Values {
		if v.Get("csrf_token") == "" {
			v.Set("csrf_token", token)
		}
		return v
	}
	for _, tc := range []struct {
		name     string
		ctl      *ForumController
		form     url.Values
		code     int
		location string
	}{
		{"created", c, form(url.Values{"title": {"Who wins"}, "section_id": {"4"}, "content": {"<p>text</p>"}}), http.StatusFound, "/forum/new-topic"},
		{"no title", c, form(url.Values{"title": {" "}, "section_id": {"4"}}), http.StatusBadRequest, ""},
		{"no section", c, form(url.Values{"title": {"T"}, "section_id": {"x"}}), http.StatusBadRequest, ""},
		{"not a forum section", c, form(url.Values{"title": {"T"}, "section_id": {"9"}}), http.StatusBadRequest, ""},
		{"a category, not a section", c, form(url.Values{"title": {"T"}, "section_id": {"3"}}), http.StatusBadRequest, ""},
		{"no right", denied, form(url.Values{"title": {"T"}, "section_id": {"4"}}), http.StatusForbidden, ""},
	} {
		rec := serve(tc.ctl.CreateTopicSubmit, testkit.PostForm("/forum/topic/create", tc.form, cookies), "")
		if rec.Code != tc.code || rec.Header().Get("Location") != tc.location {
			t.Errorf("%s: %d %q, want %d %q", tc.name, rec.Code, rec.Header().Get("Location"), tc.code, tc.location)
		}
	}
	if len(topics.created) != 1 || topics.created[0] != "Who wins" {
		t.Errorf("created: %v", topics.created)
	}

	anon, _, _ := topicController(t, 0, true, topics, &fakeNodeDelete{})
	rec := serve(anon.CreateTopicSubmit, testkit.PostForm("/forum/topic/create", url.Values{"title": {"T"}}, cookies), "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Errorf("visitor: %d %q", rec.Code, rec.Header().Get("Location"))
	}

	topics.err = errors.New("db down")
	rec = serve(c.CreateTopicSubmit, testkit.PostForm("/forum/topic/create", form(url.Values{"title": {"T"}, "section_id": {"4"}}), cookies), "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("service error: %d", rec.Code)
	}
}

// TestEditTopicSubmit - only a forum topic is edited, by whom may edit it,
// with a valid form; the topic keeps its address.
func TestEditTopicSubmit(t *testing.T) {
	topics := &fakeTopics{}
	c, cookies, token := topicController(t, 3, true, topics, &fakeNodeDelete{})
	valid := url.Values{"csrf_token": {token}, "title": {"New title"}, "section_id": {"4"}}
	if rec := serve(c.EditTopicSubmit, testkit.PostForm("/forum/topic/7/edit", valid, cookies), "7"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/forum/old-topic" { // the slug carries the prefix: not doubled
		t.Errorf("edited: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	for _, tc := range []struct {
		name, id string
		ctl      *ForumController
		code     int
	}{
		{"not a topic", "8", c, http.StatusNotFound},
		{"no node", "99", c, http.StatusNotFound},
	} {
		if rec := serve(tc.ctl.EditTopicSubmit, testkit.PostForm("/x", valid, cookies), tc.id); rec.Code != tc.code {
			t.Errorf("%s: %d, want %d", tc.name, rec.Code, tc.code)
		}
	}
	denied, _, _ := topicController(t, 3, false, topics, &fakeNodeDelete{})
	if rec := serve(denied.EditTopicSubmit, testkit.PostForm("/x", valid, cookies), "7"); rec.Code != http.StatusForbidden {
		t.Errorf("not allowed: %d", rec.Code)
	}
	if len(topics.updated) != 1 || topics.updated[0] != "New title" {
		t.Errorf("updated: %v", topics.updated)
	}
}

// TestDeleteTopic - whom may delete a forum topic deletes it and goes to
// the forum; another node or a user who may not delete it stays.
func TestDeleteTopic(t *testing.T) {
	deletes := &fakeNodeDelete{}
	c, cookies, token := topicController(t, 3, true, &fakeTopics{}, deletes)
	form := url.Values{"csrf_token": {token}}
	if rec := serve(c.DeleteTopic, testkit.PostForm("/forum/topic/7/delete", form, cookies), "7"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/forum" {
		t.Errorf("deleted: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := serve(c.DeleteTopic, testkit.PostForm("/x", form, cookies), "8"); rec.Code != http.StatusNotFound {
		t.Errorf("an article: %d", rec.Code)
	}
	denied, _, _ := topicController(t, 3, false, &fakeTopics{}, deletes)
	if rec := serve(denied.DeleteTopic, testkit.PostForm("/x", form, cookies), "7"); rec.Code != http.StatusForbidden {
		t.Errorf("not allowed: %d", rec.Code)
	}
	if len(deletes.deleted) != 1 || deletes.deleted[0] != 7 {
		t.Errorf("deleted: %v", deletes.deleted)
	}
}

// TestStickTopic - a moderator of forum topics pins a topic and unpins it,
// back to the topic; a user without the right gets 403, a node that isn't
// a topic 404, a visitor goes to sign in.
func TestStickTopic(t *testing.T) {
	topics := &fakeTopics{}
	type client struct {
		ctl     *ForumController
		cookies []*http.Cookie
		token   string
	}
	newClient := func(auth testkit.Auth) client {
		ctl, cookies, token := topicControllerAuth(t, auth, topics, &fakeNodeDelete{})
		return client{ctl, cookies, token}
	}
	moderator := newClient(testkit.Auth{UserID: 3, Allow: true, Permissions: map[string]bool{modulekit.PermEditAny("forum"): true}})
	member := newClient(testkit.Auth{UserID: 3, Allow: true})
	anon := newClient(testkit.Auth{})
	post := func(cl client, id, sticky string) *httptest.ResponseRecorder {
		return serve(cl.ctl.StickTopic, testkit.PostForm("/forum/topic/"+id+"/sticky", url.Values{"csrf_token": {cl.token}, "sticky": {sticky}}, cl.cookies), id)
	}
	if rec := post(moderator, "7", "1"); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/forum/old-topic" {
		t.Errorf("pin: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := post(moderator, "7", "0"); rec.Code != http.StatusSeeOther {
		t.Errorf("unpin: %d", rec.Code)
	}
	if !slices.Equal(topics.sticky, []string{"7:true", "7:false"}) {
		t.Errorf("sticky calls: %v", topics.sticky)
	}
	if rec := post(member, "7", "1"); rec.Code != http.StatusForbidden {
		t.Errorf("without the right: %d", rec.Code)
	}
	if rec := post(moderator, "8", "1"); rec.Code != http.StatusNotFound {
		t.Errorf("an article: %d", rec.Code)
	}
	if rec := post(anon, "7", "1"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Errorf("visitor: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if len(topics.sticky) != 2 {
		t.Errorf("refused requests changed something: %v", topics.sticky)
	}
}

// TestTopicPageHidden - an unpublished topic or one on moderation is not
// found for a visitor who may not see it (AuthService.CanViewNode).
func TestTopicPageHidden(t *testing.T) {
	c, _, _ := topicController(t, 0, false, &fakeTopics{}, &fakeNodeDelete{})
	for _, n := range []modulekit.Node{
		{ID: 7, Type: manifest.ContentType, Status: 1, OnModeration: 1},
		{ID: 7, Type: manifest.ContentType},
	} {
		rec := httptest.NewRecorder()
		c.renderTopicPage(rec, httptest.NewRequest(http.MethodGet, "/forum/old-topic", nil), &ForumTopicDetail{ID: 7, node: n})
		if rec.Code != http.StatusNotFound {
			t.Errorf("status %d, on moderation %d: %d", n.Status, n.OnModeration, rec.Code)
		}
	}
}
