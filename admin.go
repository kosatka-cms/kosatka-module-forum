package forum

import (
	"net/http"
	"strconv"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/pkg/httputil"
	"github.com/kosatka-cms/kosatka/pkg/i18n"
)

// ForumSettingsPage handles GET/POST /admin/config/forum - the forum's
// vocabulary (ForumSettings). Same pattern as UploadSettingsPage.
func (c *ForumController) ForumSettingsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if c.forumSettingsRepo == nil {
		httputil.InternalServerError(w)
		return
	}

	if r.Method == http.MethodPost {
		vocabularyID, _ := strconv.Atoi(r.FormValue("vocabulary_id"))
		if vocabularyID > 0 {
			if _, err := c.taxonomyService.VocabularyByID(ctx, vocabularyID); err != nil {
				c.LogFailure(err, "loading the vocabulary", "vocabularyID", vocabularyID)
				http.Redirect(w, r, "/admin/config/forum?error=invalid", http.StatusSeeOther)
				return
			}
		}
		if err := c.forumSettingsRepo.Update(ctx, &ForumSettings{VocabularyID: vocabularyID}); err != nil {
			c.LogFailure(err, "forum settings update")
			http.Redirect(w, r, "/admin/config/forum?error=1", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin/config/forum?saved=1", http.StatusSeeOther)
		return
	}

	settings, err := c.forumSettingsRepo.Get(ctx)
	if err != nil {
		c.LogFailure(err, "forum settings get")
		httputil.InternalServerError(w)
		return
	}
	vocabularies, err := c.taxonomyService.Vocabularies(ctx)
	if err != nil {
		c.LogFailure(err, "vocabularies for forum settings")
		httputil.InternalServerError(w)
		return
	}

	data := c.AdminPageData(r)
	data["title"] = i18n.T(r.Context(), "Forum")
	data["pageSEO"] = modulekit.PageSEO{Title: i18n.T(r.Context(), "@title - Admin panel", "@title", i18n.T(r.Context(), "Forum"))}
	data["settings"] = settings
	data["vocabularies"] = vocabularies
	data["saved"] = r.URL.Query().Get("saved") == "1"
	data["error"] = r.URL.Query().Get("error")

	c.RenderOrFail(w, r, "admin/forum_settings.html", data)
}
