package forum

import (
	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
)

// Wire - the module's services, controller, sitemap and comments hooks,
// its actions in the action log, routes, section pages (the 404-path
// fallback) and uninstall step (modulekit).
func Wire(d modulekit.Deps) modulekit.Parts {
	repo := NewForumRepository(d.DB, d.NodeRepo, d.Users, d.Taxonomy)
	forum := NewForumService(repo, d.Logger)
	registerActions()
	modulekit.AddSitemapPaths(manifest.Name, SitemapPaths(forum))
	d.Comments.OnNodeCommentsChanged(manifest.Name, CommentsChanged(repo))
	settings := NewForumSettingsRepository(d.DB)
	c := NewForumController(d.Base, ForumDeps{
		ForumService:    forum,
		TaxonomyService: d.Taxonomy,
		NodeDeleter:     d.NodeDeleter,
		Settings:        settings,
	})
	return modulekit.Parts{
		Routes: func(r modulekit.Routes) {
			RegisterAdminRoutes(r.Mux, c, r.AdminOnly)
			RegisterForumRoutes(r.Mux, r.Store, c)
		},
		Fallback:  c.SectionFallback,
		Uninstall: ForumUninstallStep(settings, d.Vocabularies, d.Logger),
	}
}
