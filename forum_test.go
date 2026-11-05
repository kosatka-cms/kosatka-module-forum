package forum

import "testing"

// TestTopicPath - a topic's page is under the forum's path prefix whether
// its slug carries it (most topics) or not (old imports, "forum-topic-N").
func TestTopicPath(t *testing.T) {
	for slug, want := range map[string]string{
		"forum/hello":       "/forum/hello",
		"forum-topic-11479": "/forum/forum-topic-11479",
		"":                  "",
	} {
		if got := (ForumTopic{Slug: slug}).Path(); got != want {
			t.Errorf("ForumTopic{%q}.Path() = %q, want %q", slug, got, want)
		}
		if got := (ForumTopicDetail{Slug: slug}).Path(); got != want {
			t.Errorf("ForumTopicDetail{%q}.Path() = %q, want %q", slug, got, want)
		}
		if got := (ForumLastTopic{Slug: slug}).Path(); got != want {
			t.Errorf("ForumLastTopic{%q}.Path() = %q, want %q", slug, got, want)
		}
	}
}
