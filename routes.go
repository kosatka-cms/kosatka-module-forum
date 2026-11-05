package forum

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/kosatka-cms/kosatka/modulekit"
)

// RegisterForumRoutes registers the forum's pages: the index, the active
// and unanswered topics, a section, and a topic's new, edit, delete and
// pin actions (a new topic - signed in only).
func RegisterForumRoutes(r modulekit.Mux, store *sessions.CookieStore, fc *ForumController) {
	auth := modulekit.AuthMiddleware(store)
	r.HandleFunc("GET /forum", fc.Index)
	r.HandleFunc("GET /forum/active", fc.ActiveTopics)
	r.HandleFunc("GET /forum/unanswered", fc.UnansweredTopics)
	r.Handle("GET /forum/new-topic/{path...}", auth(http.HandlerFunc(fc.CreateTopicForm)))
	r.Handle("POST /forum/new-topic/{path...}", auth(http.HandlerFunc(fc.CreateTopicSubmit)))
	r.HandleFunc("GET /forum/topic/{id}/edit", fc.EditTopicForm)
	r.HandleFunc("POST /forum/topic/{id}/edit", fc.EditTopicSubmit)
	r.HandleFunc("GET /forum/topic/{id}/delete", fc.DeleteTopicConfirm)
	r.HandleFunc("POST /forum/topic/{id}/delete", fc.DeleteTopic)
	r.HandleFunc("POST /forum/topic/{id}/sticky", fc.StickTopic)
	r.HandleFunc("GET /forum/{path...}", fc.Section)
}

// RegisterAdminRoutes registers the forum settings page behind admin
// (sign-in and the admin permission).
func RegisterAdminRoutes(r modulekit.Mux, fc *ForumController, admin func(http.Handler) http.Handler) {
	r.Handle("GET /admin/config/forum", admin(http.HandlerFunc(fc.ForumSettingsPage)))
	r.Handle("POST /admin/config/forum", admin(http.HandlerFunc(fc.ForumSettingsPage)))
}
