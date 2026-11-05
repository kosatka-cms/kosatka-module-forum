package forum

import (
	"strings"
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

// TestForumCategoryPageRender - forum/index.html as a category page: its
// title, the one category with its sections, no forum statistics.
func TestForumCategoryPageRender(t *testing.T) {
	testkit.UseEngineRoot(t)
	data := map[string]any{
		"title": "General topics",
		"categories": []ForumCategory{{ID: 439, Name: "General topics", Slug: "forumy/obshchie",
			Sections: []ForumSection{{ID: 703, Name: "Section", Slug: "forumy/obshchie-temy/razdel"}}}},
	}
	out := testkit.RenderPageBlock(t, "forum/index.html", "title", data) + testkit.RenderPageContent(t, "forum/index.html", data)
	for _, want := range []string{`<h1 class="page-title">General topics</h1>`, `href="/forumy/obshchie"`, `href="/forumy/obshchie-temy/razdel"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "forum-statistics") {
		t.Error("a category page must not show the forum statistics")
	}
}

// TestForumSettingsPageRender - the forum vocabulary select marks the
// current vocabulary.
func TestForumSettingsPageRender(t *testing.T) {
	testkit.UseEngineRoot(t)
	data := map[string]any{
		"settings":     &ForumSettings{VocabularyID: 7},
		"vocabularies": modulekit.TaxonomyVocabularies{{ID: 3, Name: "Tags"}, {ID: 7, Name: "Forum"}},
		"error":        "",
	}
	out := testkit.RenderPageContent(t, "admin/forum_settings.html", data)
	if !strings.Contains(out, `<option value="7" selected>Forum</option>`) || strings.Contains(out, `<option value="3" selected>`) {
		t.Errorf("current vocabulary not selected:\n%s", out)
	}
}
