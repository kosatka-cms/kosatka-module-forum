//go:build dbintegration

package forum

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

func TestForumRepository(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		seedUser(t, db)
		seedRow(t, db, `INSERT INTO taxonomy_vocabulary (id, name, slug, term_path_prefix) VALUES (1, 'Forum', 'forum', '/forum/')`)
		seedRow(t, db, `UPDATE forum_settings SET vocabulary_id = 1`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight) VALUES (1, 1, 'General', 'general', 0)`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight, parent_id) VALUES (2, 1, 'Talk', 'talk', 1, 1), (3, 1, 'Help', 'help', 0, 1)`)

		forum := newForumRepository(testkit.Deps(db))
		if prefix, err := forum.SectionPathPrefix(ctx); err != nil || prefix != "forum" {
			t.Errorf("prefix: %q %v", prefix, err)
		}
		topic, err := forum.CreateTopic(ctx, 2, "Hello", "first", 1)
		if err != nil || topic.Content != "first" || topic.Section == nil || topic.Section.ParentName != "General" {
			t.Fatalf("create topic: %+v %v", topic, err)
		}
		if _, err := forum.CreateTopic(ctx, 2, "Hello", "second", 1); err != nil {
			t.Fatal(err)
		}
		if ok, err := forum.TopicExistsBySlug(ctx, "forum/hello-1"); err != nil || !ok {
			t.Errorf("second slug: %v %v", ok, err)
		}
		if err := forum.UpdateTopic(ctx, topic.ID, "Hello again", "edited", 3); err != nil {
			t.Fatal(err)
		}
		if tid, err := forum.GetTopicSectionID(ctx, topic.ID); err != nil || tid != 3 {
			t.Errorf("section id: %d %v", tid, err)
		}
		got, err := forum.GetTopicBySlug(ctx, topic.Slug)
		if err != nil || got.Title != "Hello again" || got.Content != "edited" || got.Section.Slug != "help" {
			t.Errorf("topic: %+v %v", got, err)
		}
		// A slug without the type's prefix: the topic's page is still under
		// it (NodeRelPath), and found by that path.
		seedRow(t, db, `UPDATE nodes SET slug = 'hello' WHERE id = `+strconv.Itoa(topic.ID))
		if got, err := forum.GetTopicBySlug(ctx, "forum/hello"); err != nil || got.ID != topic.ID {
			t.Errorf("topic by its page's path: %+v %v", got, err)
		}
		seedRow(t, db, `UPDATE nodes SET slug = '`+topic.Slug+`' WHERE id = `+strconv.Itoa(topic.ID))

		if s, err := forum.GetSectionBySlug(ctx, "talk"); err != nil || s.ParentSlug != "general" || s.ParentID != 1 {
			t.Errorf("section: %+v %v", s, err)
		}
		if _, err := forum.GetSectionBySlug(ctx, "nope"); !modulekit.IsNotFound(err) {
			t.Errorf("missing section: %v", err)
		}
		if n, err := forum.CountTopicsBySection(ctx, 2); err != nil || n != 1 {
			t.Errorf("count: %d %v", n, err)
		}
		topics, err := forum.GetTopicsBySection(ctx, 2, ForumTopicFilter{Limit: 10})
		if err != nil || len(topics) != 1 || topics[0].Slug != "forum/hello-1" {
			t.Errorf("topics: %+v %v", topics, err)
		}
		// Across sections each topic names its section; unanswered ones
		// only those without replies.
		seedRow(t, db, `UPDATE forum_stats SET comment_count = 3 WHERE nid = `+strconv.Itoa(topics[0].ID))
		all, err := forum.ListTopics(ctx, ForumTopicFilter{Limit: 10, OrderBy: "title", OrderDir: "asc"})
		if err != nil || len(all) != 2 || all[0].Section == nil || all[0].Section.Slug != "talk" || all[1].Section == nil || all[1].Section.Name != "Help" {
			t.Errorf("all topics: %+v %v", all, err)
		}
		if n, err := forum.CountTopicsFiltered(ctx, ForumTopicFilter{}); err != nil || n != 2 {
			t.Errorf("count all: %d %v", n, err)
		}
		unanswered := ForumTopicFilter{Limit: 10, Unanswered: true}
		if got, err := forum.ListTopics(ctx, unanswered); err != nil || len(got) != 1 || got[0].ID != topic.ID {
			t.Errorf("unanswered: %+v %v", got, err)
		}
		if n, err := forum.CountTopicsFiltered(ctx, unanswered); err != nil || n != 1 {
			t.Errorf("count unanswered: %d %v", n, err)
		}
		if got, err := forum.ListTopics(ctx, ForumTopicFilter{Limit: 10, SectionID: 2}); err != nil || len(got) != 1 || got[0].Section != nil {
			t.Errorf("a section's list names no section: %+v %v", got, err)
		}
		// The sort direction comes from the URL: anything but asc/desc is
		// ignored, never put into the SQL.
		for _, dir := range []string{"asc", "DESC", "desc, (SELECT 1 FROM users)", "asc; DROP TABLE nodes"} {
			if topics, err := forum.GetTopicsBySection(ctx, 2, ForumTopicFilter{Limit: 10, OrderBy: "title", OrderDir: dir}); err != nil || len(topics) != 1 {
				t.Errorf("order %q: %+v %v", dir, topics, err)
			}
		}
		cats, err := forum.GetForumCategories(ctx)
		if err != nil || len(cats) != 1 || len(cats[0].Sections) != 2 {
			t.Errorf("categories: %+v %v", cats, err)
		}
		if st, err := forum.GetForumStatistics(ctx); err != nil || st.TotalTopics != 2 || st.TotalUsers != 1 || !reflect.DeepEqual(st.LatestUsers, []string{"alice"}) {
			t.Errorf("statistics: %+v %v", st, err)
		}
		terms, err := forum.GetForumTermsForSelect(ctx)
		var names []string
		for _, o := range terms {
			names = append(names, o.DisplayName)
		}
		if err != nil || !reflect.DeepEqual(names, []string{"General", "- Help", "- Talk"}) {
			t.Errorf("terms: %v %v", names, err)
		}

		// A topic on moderation leaves the lists, the counts and the last
		// topic of its section; its page is for the visitor's check
		// (ForumTopicDetail.node).
		seedRow(t, db, `UPDATE nodes SET on_moderation = 1 WHERE slug = 'forum/hello-1'`)
		if n, err := forum.CountTopicsBySection(ctx, 2); err != nil || n != 0 {
			t.Errorf("moderated: section count %d %v", n, err)
		}
		if got, err := forum.ListTopics(ctx, ForumTopicFilter{Limit: 10}); err != nil || len(got) != 1 || got[0].ID != topic.ID {
			t.Errorf("moderated: topics %+v %v", got, err)
		}
		if n, err := forum.CountTopicsFiltered(ctx, ForumTopicFilter{}); err != nil || n != 1 {
			t.Errorf("moderated: count %d %v", n, err)
		}
		if st, err := forum.GetForumStatistics(ctx); err != nil || st.TotalTopics != 1 || st.TotalPosts != 0 {
			t.Errorf("moderated: statistics %+v %v", st, err)
		}
		cats, err = forum.GetForumCategories(ctx)
		if err != nil || len(cats) != 1 {
			t.Fatalf("moderated: categories %+v %v", cats, err)
		}
		for _, s := range cats[0].Sections {
			if s.Slug == "talk" && (s.TopicsCount != 0 || (s.LastTopic != nil && s.LastTopic.Title != "")) {
				t.Errorf("moderated: section %+v %+v", s, s.LastTopic)
			}
		}
		if got, err := forum.GetTopicBySlug(ctx, "forum/hello-1"); err != nil || got.node.OnModeration != 1 {
			t.Errorf("moderated topic: %+v %v", got, err)
		}
	})
}

// TestForumSettingsRepository - the module's settings row keeps the forum
// vocabulary and clears it.
func TestForumSettingsRepository(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		// The forum's install step created its row.
		const vocab = 1
		seedRow(t, db, `INSERT INTO taxonomy_vocabulary (id, name, slug, created_at, updated_at) VALUES (1, 'Forum', 'forum', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		forum := NewForumSettingsRepository(db.DB)
		if err := forum.Update(ctx, &ForumSettings{VocabularyID: vocab}); err != nil {
			t.Fatal(err)
		}
		if got, err := forum.Get(ctx); err != nil || got.VocabularyID != vocab {
			t.Errorf("forum settings: %+v %v", got, err)
		}
		_ = forum.Update(ctx, &ForumSettings{})
		if got, _ := forum.Get(ctx); got.VocabularyID != 0 {
			t.Errorf("forum vocabulary not cleared: %+v", got)
		}
	})
}

// TestSetSticky - a pinned topic leads its section's list; unpinned, it
// takes its place by date again.
func TestSetSticky(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		seedUser(t, db)
		seedRow(t, db, `INSERT INTO taxonomy_vocabulary (id, name, slug, term_path_prefix) VALUES (1, 'Forum', 'forum', '/forum/')`)
		seedRow(t, db, `UPDATE forum_settings SET vocabulary_id = 1`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight) VALUES (1, 1, 'General', 'general', 0)`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight, parent_id) VALUES (2, 1, 'Talk', 'talk', 0, 1)`)
		forum := newForumRepository(testkit.Deps(db))
		older, err := forum.CreateTopic(ctx, 2, "Older", "a", 1)
		if err != nil {
			t.Fatal(err)
		}
		seedRow(t, db, `UPDATE forum_stats SET last_comment_at = '2020-01-01 00:00:00'`)
		if _, err := forum.CreateTopic(ctx, 2, "Newer", "b", 1); err != nil {
			t.Fatal(err)
		}
		first := func() string {
			t.Helper()
			topics, err := forum.GetTopicsBySection(ctx, 2, ForumTopicFilter{Limit: 10})
			if err != nil || len(topics) != 2 {
				t.Fatalf("topics: %v %v", topics, err)
			}
			return topics[0].Title
		}
		if got := first(); got != "Newer" {
			t.Fatalf("by date first: %q", got)
		}
		if err := forum.SetSticky(ctx, older.ID, true); err != nil {
			t.Fatal(err)
		}
		if got := first(); got != "Older" {
			t.Errorf("pinned first: %q", got)
		}
		if err := forum.SetSticky(ctx, older.ID, false); err != nil {
			t.Fatal(err)
		}
		if got := first(); got != "Newer" {
			t.Errorf("unpinned: %q", got)
		}
	})
}
