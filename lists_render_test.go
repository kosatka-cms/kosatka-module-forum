package forum

import (
	"strings"
	"testing"
	"time"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
	"github.com/kosatka-cms/kosatka/pkg/pagination"
)

// TestForumTopicListsTemplate - a list across sections names each topic's
// section and offers no "new topic" link; a section's list does the
// opposite (to a user who may start one); the forum index links both
// lists.
func TestForumTopicListsTemplate(t *testing.T) {
	testkit.UseEngineRoot(t)
	topic := ForumTopic{ID: 1, Title: "Hello", Slug: "forum/hello", Author: &modulekit.User{Username: "ann"}, CreatedAt: time.Now()}
	across := topic
	across.Section = &ForumTopicSection{Name: "Help", Slug: "forum/help"}
	out := testkit.RenderPageContent(t, "forum/section.html", map[string]any{"title": "Active topics", "topics": []ForumTopic{across}, "orderBy": "last_updated", "sortDir": "desc", "user": &modulekit.User{ID: 1}, "pagination": pagination.New(1, 1, 25, "/forum/active", nil)})
	if !strings.Contains(out, `<a href="/forum/help" class="forum-topic-section">Help</a>`) || strings.Contains(out, "new-topic") {
		t.Errorf("across sections: %s", out)
	}
	section := func(canCreate bool) string {
		return testkit.RenderPageContent(t, "forum/section.html", map[string]any{"title": "Talk", "section": &ForumSection{Name: "Talk", Slug: "forum/talk"}, "topics": []ForumTopic{topic}, "orderBy": "last_updated", "sortDir": "desc", "user": &modulekit.User{ID: 1}, "canCreateTopic": canCreate, "pagination": pagination.New(1, 1, 25, "/forum/active", nil)})
	}
	if out := section(true); !strings.Contains(out, `href="/forum/new-topic/forum/talk"`) || strings.Contains(out, "forum-topic-section") {
		t.Errorf("a section: %s", out)
	}
	// Signed in without the right: neither the link nor the sign-in one.
	if out := section(false); strings.Contains(out, "new-topic") {
		t.Errorf("a section, no right to start a topic: %s", out)
	}
	if out := testkit.RenderPageContent(t, "forum/index.html", map[string]any{"title": "Forum"}); !strings.Contains(out, `href="/forum/active"`) || !strings.Contains(out, `href="/forum/unanswered"`) {
		t.Errorf("index: %s", out)
	}
}
