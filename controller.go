package forum

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/pkg/csrf"
	"github.com/kosatka-cms/kosatka/pkg/htmlsanitize"
	"github.com/kosatka-cms/kosatka/pkg/httputil"
	"github.com/kosatka-cms/kosatka/pkg/i18n"
	"github.com/kosatka-cms/kosatka/pkg/pagination"
)

// ForumController handles forum HTTP requests.
type ForumController struct {
	*modulekit.BaseController
	forumService    ForumService
	taxonomyService modulekit.TaxonomyService
	commentsService modulekit.CommentsService
	nodeRepo        modulekit.NodesRepository
	nodeDeleter     modulekit.NodeDeleter
	// forumSettingsRepo - the module's settings (ForumDeps, admin.go).
	forumSettingsRepo ForumSettingsRepository
}

// ForumDeps - what NewForumController needs besides the BaseDeps.
type ForumDeps struct {
	ForumService    ForumService
	TaxonomyService modulekit.TaxonomyService
	NodeDeleter     modulekit.NodeDeleter
	Settings        ForumSettingsRepository
}

// NewForumController creates a ForumController.
func NewForumController(deps modulekit.BaseDeps, d ForumDeps) *ForumController {
	base := modulekit.NewBaseController(deps)

	return &ForumController{
		BaseController:    base,
		forumService:      d.ForumService,
		taxonomyService:   d.TaxonomyService,
		commentsService:   deps.Comments,
		nodeRepo:          deps.Nodes,
		nodeDeleter:       d.NodeDeleter,
		forumSettingsRepo: d.Settings,
	}
}

// Index displays the main forum page with categories and sections.
func (c *ForumController) Index(w http.ResponseWriter, r *http.Request) {
	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	forumIndex, err := c.forumService.GetForumIndex(dbCtx)
	if err != nil {
		c.LogFailure(err, "error fetching forum index")
		httputil.InternalServerError(w)
		return
	}

	var user *modulekit.User
	userID, loggedIn := c.AuthService().GetUserID(r)
	if loggedIn {
		user, err = c.UserService().ByID(dbCtx, userID)
		if err != nil {
			c.LogFailure(err, "loading the signed-in user", "userID", userID)
			user = nil
		}
	}

	data := c.GetCommonDataWithUser(r, user)
	data["categories"] = forumIndex.Categories
	data["statistics"] = forumIndex.Statistics
	pageTitle := i18n.T(r.Context(), "Forum")
	data["pageSEO"] = modulekit.PageSEOForList(pageTitle, r.URL.Path)
	data["title"] = pageTitle
	c.SetPathTrail(data, "/forum", pageTitle)

	c.RenderOrFail(w, r, "forum/index.html", data)
}

// SectionFallback serves a forum section at the forum vocabulary's own
// term path prefix (/<prefix>/category/section - the URLs of its terms),
// besides the module's /forum/… routes. Returns false for any other path.
func (c *ForumController) SectionFallback(w http.ResponseWriter, r *http.Request) bool {
	prefix := c.forumService.SectionPathPrefix(r.Context())
	if prefix == "" || prefix == "forum" || !strings.HasPrefix(strings.TrimPrefix(r.URL.Path, "/"), prefix+"/") {
		return false
	}
	c.Section(w, r)
	return true
}

// Section displays a forum section with its topics: the section,
// category or topic whose slug is the URL's path.
func (c *ForumController) Section(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	orderBy := r.URL.Query().Get("order")
	if orderBy == "" {
		orderBy = "last_updated"
	}

	sortDir := r.URL.Query().Get("sort")
	if sortDir == "" {
		sortDir = "desc"
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
			page = p
		}
	}

	itemsPerPage := 25

	sectionPage, totalCount, err := c.forumService.GetSectionBySlug(dbCtx, path)
	if err != nil {
		topic, topicErr := c.forumService.GetTopic(dbCtx, path)
		if topicErr == nil && topic != nil {
			c.renderTopicPage(w, r, topic)
			return
		}
		if !modulekit.IsNotFound(err) {
			topicErr = err
		}
		// No such section or topic is a 404 for the visitor.
		c.RenderLoadFailure(w, r, topicErr, "loading forum section or topic", "path", path)
		return
	}
	if sectionPage.Section.ParentID == 0 {
		c.renderCategoryPage(w, r, sectionPage.Section)
		return
	}

	// Pagination.
	pager := pagination.New(
		page,
		totalCount,
		itemsPerPage,
		r.URL.Path,
		r.URL.Query(),
	)

	filter := ForumTopicFilter{
		SectionID: sectionPage.Section.ID,
		Offset:    pager.Offset,
		Limit:     pager.ItemsPerPage,
		OrderBy:   orderBy,
		OrderDir:  sortDir,
	}

	topics, err := c.forumService.GetTopics(dbCtx, sectionPage.Section.ID, filter)
	if err != nil {
		c.LogFailure(err, "error fetching topics")
		httputil.InternalServerError(w)
		return
	}

	sectionPage.Topics = topics

	var user *modulekit.User
	userID, loggedIn := c.AuthService().GetUserID(r)
	if loggedIn {
		user, err = c.UserService().ByID(dbCtx, userID)
		if err != nil {
			c.LogFailure(err, "loading the signed-in user", "userID", userID)
			user = nil
		}
	}

	data := c.GetCommonDataWithUser(r, user)
	data["section"] = sectionPage.Section
	data["topics"] = sectionPage.Topics
	data["canCreateTopic"] = c.AuthService().CanCreateNode(dbCtx, manifest.ContentType, user)
	data["pagination"] = pager
	data["orderBy"] = orderBy
	data["sortDir"] = sortDir
	data["pageSEO"] = modulekit.PageSEOForPaginatedList(sectionPage.Section.Name, r.URL.Path, page, r.URL.Query())
	data["title"] = sectionPage.Section.Name
	sectionParents := forumSectionParents(r.Context(), sectionPage.Section)
	c.SetChildTrail(data, sectionParents[:len(sectionParents)-1], sectionPage.Section.Name)

	c.RenderOrFail(w, r, "forum/section.html", data)
}

// renderTopicPage renders a topic page with the given topic data.
func (c *ForumController) renderTopicPage(w http.ResponseWriter, r *http.Request, topic *ForumTopicDetail) {
	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	var user *modulekit.User
	userID, loggedIn := c.AuthService().GetUserID(r)
	if loggedIn {
		var err error
		user, err = c.UserService().ByID(dbCtx, userID)
		if err != nil {
			c.LogFailure(err, "loading the signed-in user", "userID", userID)
			user = nil
		}
	}
	// Unpublished or on moderation: only for those who may see it.
	if !c.AuthService().CanViewNode(dbCtx, &topic.node, user) {
		c.RenderNotFound(w, r)
		return
	}

	pageStr := r.URL.Query().Get("page")
	page := 1

	// "last" redirects to the last page.
	if pageStr == "last" {
		// Count all comments to determine last page
		totalAllComments, err := c.commentsService.CountByNodeID(dbCtx, topic.ID, 1, false)
		if err == nil && totalAllComments > 0 {
			lastPage := (totalAllComments + modulekit.CommentsPerPage() - 1) / modulekit.CommentsPerPage()
			page = lastPage
		} else {
			c.LogFailure(err, "counting the content's comments", "nodeID", topic.ID)
		}
	} else if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p >= 1 {
			page = p
		}
	}

	// Paginated comments in threaded order (50 per page, replies included).
	comments, totalAllComments, err := c.commentsService.ByNodeIDPaginated(dbCtx, topic.ID, 1, page)
	if err != nil {
		c.LogFailure(err, "error fetching comments", "topic_id", topic.ID, "page", page)
		// On failure, render with no comments.
		comments = modulekit.Comments{}
		totalAllComments = 0
	}

	// Pagination over all comments.
	pager := pagination.NewWithAnchor(
		page,
		totalAllComments,
		modulekit.CommentsPerPage(),
		r.URL.Path,
		r.URL.Query(),
		"comments",
	)

	data := c.GetCommonData(r)
	data["topic"] = topic
	data["comments"] = comments
	data["user"] = user
	data["pagination"] = pager
	data["pageSEO"] = modulekit.PageSEOForPaginatedList(topic.Title, r.URL.Path, page, r.URL.Query())
	data["title"] = topic.Title
	c.SetChildTrail(data, forumSectionParents(r.Context(), topic.Section), topic.Title)
	data["canAdministerComments"] = c.AuthService().HasPermission(dbCtx, user, modulekit.PermAdministerComments)

	// The links follow the checks of the edit and delete pages.
	if n, err := c.nodeRepo.ByID(dbCtx, topic.ID); err == nil && n != nil {
		c.SetViewTrackNID(data, *n)
		data["canEditTopic"] = c.AuthService().CanEditNode(dbCtx, n, user)
		data["canDeleteTopic"] = c.AuthService().CanDeleteNode(dbCtx, n, user)
		data["canStickTopic"] = c.canStickTopics(dbCtx, user)
	} else {
		c.LogFailure(err, "loading the content", "nid", topic.ID)
	}

	c.RenderOrFail(w, r, "forum/topic.html", data)
}

// ActiveTopics handles GET /forum/active: every section's topics by their
// last post.
func (c *ForumController) ActiveTopics(w http.ResponseWriter, r *http.Request) {
	c.renderTopicList(w, r, i18n.T(r.Context(), "Active topics"), ForumTopicFilter{OrderBy: "last_updated"})
}

// UnansweredTopics handles GET /forum/unanswered: topics without replies,
// newest first.
func (c *ForumController) UnansweredTopics(w http.ResponseWriter, r *http.Request) {
	c.renderTopicList(w, r, i18n.T(r.Context(), "Unanswered topics"), ForumTopicFilter{OrderBy: "created", Unanswered: true})
}

// renderTopicList renders a list of topics across sections (filter gives
// the default order) with the section page's table, sorting and pages; a
// page past the last is a 404.
func (c *ForumController) renderTopicList(w http.ResponseWriter, r *http.Request, title string, filter ForumTopicFilter) {
	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()
	q := r.URL.Query()
	if o := q.Get("order"); o != "" {
		filter.OrderBy = o
	}
	filter.OrderDir = "desc"
	if strings.EqualFold(q.Get("sort"), "asc") {
		filter.OrderDir = "asc"
	}
	page := c.GetPageNumber(r)
	const perPage = 25
	filter.Limit, filter.Offset = perPage, (page-1)*perPage
	topics, total, err := c.forumService.ListTopics(dbCtx, filter)
	if err != nil {
		c.LogFailure(err, "listing the topics")
		httputil.InternalServerError(w)
		return
	}
	if page > 1 && len(topics) == 0 {
		c.RenderNotFound(w, r)
		return
	}
	user, _ := c.GetCurrentUser(r)
	data := c.GetCommonDataWithUser(r, user)
	data["title"] = title
	data["topics"] = topics
	data["pagination"] = pagination.New(page, total, perPage, r.URL.Path, q)
	data["orderBy"] = filter.OrderBy
	data["sortDir"] = filter.OrderDir
	// Ever-changing lists of topics found elsewhere: kept out of the index.
	seo := modulekit.PageSEOForPaginatedList(title, r.URL.Path, page, q)
	seo.NoIndex = true
	data["pageSEO"] = seo
	c.SetPathTrail(data, r.URL.Path, title)
	c.RenderOrFail(w, r, "forum/section.html", data)
}

// CreateTopicForm displays the form for creating a forum topic.
func (c *ForumController) CreateTopicForm(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		c.RenderNotFound(w, r)
		return
	}
	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	if user, _ := c.GetCurrentUser(r); !c.AuthService().CanCreateNode(dbCtx, manifest.ContentType, user) {
		c.RenderForbidden(w, r)
		return
	}

	sectionPage, _, err := c.forumService.GetSectionBySlug(dbCtx, path)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the forum section", "path", path)
		return
	}
	if sectionPage == nil {
		c.RenderNotFound(w, r)
		return
	}

	forumTerms, err := c.forumService.GetForumTermsForSelect(dbCtx)
	if err != nil {
		c.LogFailure(err, "error fetching forum terms")
		httputil.InternalServerError(w)
		return
	}

	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login?destination=/forum/new-topic/"+path, http.StatusFound)
		return
	}

	data := c.GetCommonData(r)
	data["section"] = sectionPage.Section
	data["forumTerms"] = forumTerms
	data["selectedSectionID"] = sectionPage.Section.ID
	data["formAction"] = "/forum/new-topic/" + path
	data["title"] = i18n.T(r.Context(), "Create new topic")
	c.SetChildTrail(data, forumSectionParents(r.Context(), sectionPage.Section), i18n.T(r.Context(), "Create new topic"))
	data["pageSEO"] = modulekit.PageSEOForList(i18n.T(r.Context(), "Create topic"), r.URL.Path)
	data["CSRFToken"] = csrf.EnsureToken(c.Store(), w, r)

	c.RenderOrFail(w, r, "forum/create_topic.html", data)
}

// checkSection - whether sectionID is a section of the forum (a term of
// its vocabulary under a category: a topic in a category would be listed
// in no section), the topic forms' choice; false - answered (a bad
// request, or a failure to load the terms).
func (c *ForumController) checkSection(ctx context.Context, w http.ResponseWriter, r *http.Request, sectionID int) bool {
	terms, err := c.forumService.GetForumTermsForSelect(ctx)
	if err != nil {
		c.LogFailure(err, "listing the forum sections")
		httputil.InternalServerError(w)
		return false
	}
	if !slices.ContainsFunc(terms, func(t ForumTermOption) bool { return t.ID == sectionID && t.ParentID != 0 }) {
		c.RenderBadRequest(w, r, i18n.T(r.Context(), "Select a forum section"))
		return false
	}
	return true
}

// CreateTopicSubmit handles the create forum topic form submission.
func (c *ForumController) CreateTopicSubmit(w http.ResponseWriter, r *http.Request) {
	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		c.RenderBadForm(w, r)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	body := htmlsanitize.Sanitize(r.FormValue("content"))
	sectionIDStr := r.FormValue("section_id")

	if title == "" {
		c.RenderBadRequest(w, r, i18n.T(r.Context(), "Title is required"))
		return
	}

	sectionID, err := strconv.Atoi(sectionIDStr)
	if err != nil || sectionID <= 0 {
		c.RenderBadRequest(w, r, i18n.T(r.Context(), "Select a forum section"))
		return
	}

	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	if !c.AuthService().CanCreateNode(dbCtx, manifest.ContentType, user) {
		c.RenderForbidden(w, r)
		return
	}
	if !c.checkSection(dbCtx, w, r, sectionID) {
		return
	}

	topic, err := c.forumService.CreateTopic(dbCtx, sectionID, title, body, user.ID)
	if err != nil {
		c.LogFailure(err, "error creating forum topic")
		httputil.InternalServerError(w)
		return
	}

	http.Redirect(w, r, topic.Path(), http.StatusFound)
}

// EditTopicForm displays the form for editing a forum topic.
func (c *ForumController) EditTopicForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeIDStr := r.PathValue("id")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.RenderNotFound(w, r)
		return
	}

	node, err := c.nodeRepo.ByID(ctx, nodeID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the content", "nodeID", nodeID)
		return
	}
	if node == nil || node.Type != manifest.ContentType {
		c.RenderNotFound(w, r)
		return
	}

	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login?destination="+r.URL.Path, http.StatusFound)
		return
	}

	if !c.AuthService().CanEditNode(ctx, node, user) {
		c.RenderForbidden(w, r)
		return
	}

	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	topic, err := c.forumService.GetTopic(dbCtx, node.Slug)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the topic", "slug", node.Slug)
		return
	}

	forumTerms, err := c.forumService.GetForumTermsForSelect(dbCtx)
	if err != nil {
		c.LogFailure(err, "error fetching forum terms")
		httputil.InternalServerError(w)
		return
	}

	selectedSectionID, _ := c.forumService.GetTopicSectionID(dbCtx, nodeID)

	data := c.GetCommonData(r)
	data["topic"] = topic
	data["forumTerms"] = forumTerms
	data["selectedSectionID"] = selectedSectionID
	data["title"] = i18n.T(r.Context(), "Editing: @title", "@title", topic.Title)
	c.SetChildTrail(data, append(forumSectionParents(r.Context(), topic.Section), modulekit.BreadcrumbItem{URL: topic.Path(), Label: topic.Title}), i18n.T(r.Context(), "Editing"))
	data["pageSEO"] = modulekit.PageSEOForList(i18n.T(r.Context(), "Edit topic"), r.URL.Path)
	data["CSRFToken"] = csrf.EnsureToken(c.Store(), w, r)

	c.RenderOrFail(w, r, "forum/edit_topic.html", data)
}

// EditTopicSubmit handles the edit forum topic form submission.
func (c *ForumController) EditTopicSubmit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeIDStr := r.PathValue("id")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.RenderNotFound(w, r)
		return
	}

	node, err := c.nodeRepo.ByID(ctx, nodeID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the content", "nodeID", nodeID)
		return
	}
	if node == nil || node.Type != manifest.ContentType {
		c.RenderNotFound(w, r)
		return
	}

	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if !c.AuthService().CanEditNode(ctx, node, user) {
		c.RenderForbidden(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		c.RenderBadForm(w, r)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	body := htmlsanitize.Sanitize(r.FormValue("content"))
	sectionIDStr := r.FormValue("section_id")

	if title == "" {
		c.RenderBadRequest(w, r, i18n.T(r.Context(), "Title is required"))
		return
	}

	sectionID, err := strconv.Atoi(sectionIDStr)
	if err != nil || sectionID <= 0 {
		c.RenderBadRequest(w, r, i18n.T(r.Context(), "Select a forum section"))
		return
	}

	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	if !c.checkSection(dbCtx, w, r, sectionID) {
		return
	}
	if err := c.forumService.UpdateTopic(dbCtx, nodeID, title, body, sectionID); err != nil {
		c.LogFailure(err, "error updating forum topic")
		httputil.InternalServerError(w)
		return
	}

	http.Redirect(w, r, "/"+modulekit.NodeRelPath(node.Type, node.Slug), http.StatusFound)
}

// DeleteTopicConfirm displays the delete confirmation page for a forum topic.
func (c *ForumController) DeleteTopicConfirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeIDStr := r.PathValue("id")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.RenderNotFound(w, r)
		return
	}

	node, err := c.nodeRepo.ByID(ctx, nodeID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the content", "nodeID", nodeID)
		return
	}
	if node == nil || node.Type != manifest.ContentType {
		c.RenderNotFound(w, r)
		return
	}

	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login?destination="+r.URL.Path, http.StatusFound)
		return
	}

	if !c.AuthService().CanDeleteNode(ctx, node, user) {
		c.RenderForbidden(w, r)
		return
	}

	data := c.GetCommonData(r)
	data["topic"] = node
	data["topicPath"] = "/" + modulekit.NodeRelPath(node.Type, node.Slug)
	data["title"] = i18n.T(ctx, "Delete topic")
	c.SetNodeSubpageTrail(ctx, data, *node, i18n.T(ctx, "Delete topic"))
	data["pageSEO"] = modulekit.PageSEOForList(i18n.T(ctx, "Delete topic"), r.URL.Path)
	data["CSRFToken"] = csrf.EnsureToken(c.Store(), w, r)

	c.RenderOrFail(w, r, "forum/delete_topic.html", data)
}

// DeleteTopic handles the delete forum topic form submission: the topic is
// deleted for good, with its replies.
func (c *ForumController) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeIDStr := r.PathValue("id")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.RenderNotFound(w, r)
		return
	}

	node, err := c.nodeRepo.ByID(ctx, nodeID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the content", "nodeID", nodeID)
		return
	}
	if node == nil || node.Type != manifest.ContentType {
		c.RenderNotFound(w, r)
		return
	}

	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if !c.AuthService().CanDeleteNode(ctx, node, user) {
		c.RenderForbidden(w, r)
		return
	}

	if err := c.nodeDeleter.Delete(ctx, nodeID); err != nil {
		c.LogFailure(err, "error deleting forum topic", "nodeID", nodeID)
		httputil.InternalServerError(w)
		return
	}

	http.Redirect(w, r, "/forum", http.StatusFound)
}

// renderCategoryPage - a category (a root forum term, which holds
// sections, not topics): the forum index layout with this category only.
func (c *ForumController) renderCategoryPage(w http.ResponseWriter, r *http.Request, section *ForumSection) {
	dbCtx, dbCancel := c.DBContext(r)
	defer dbCancel()

	category, err := c.forumService.GetCategory(dbCtx, section.ID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the forum category", "categoryID", section.ID)
		return
	}

	var user *modulekit.User
	if userID, loggedIn := c.AuthService().GetUserID(r); loggedIn {
		user, _ = c.UserService().ByID(dbCtx, userID)
	}

	data := c.GetCommonDataWithUser(r, user)
	data["categories"] = []ForumCategory{*category}
	data["pageSEO"] = modulekit.PageSEOForList(category.Name, r.URL.Path)
	data["title"] = category.Name
	c.SetChildTrail(data, forumSectionParents(r.Context(), nil), category.Name)

	c.RenderOrFail(w, r, "forum/index.html", data)
}

// forumSectionParents - the forum's own page, then the section's parent
// and the section (the trail parents of a section's pages).
func forumSectionParents(ctx context.Context, section *ForumSection) []modulekit.BreadcrumbItem {
	parents := []modulekit.BreadcrumbItem{{URL: "/forum", Label: i18n.T(ctx, "Forum")}}
	if section == nil {
		return parents
	}
	if section.ParentName != "" {
		parents = append(parents, modulekit.BreadcrumbItem{URL: "/" + section.ParentSlug, Label: section.ParentName})
	}
	return append(parents, modulekit.BreadcrumbItem{URL: "/" + section.Slug, Label: section.Name})
}

// canStickTopics - the user may pin topics: a moderator of the forum's
// topics ("edit any forum content").
func (c *ForumController) canStickTopics(ctx context.Context, user *modulekit.User) bool {
	return user != nil && c.AuthService().HasPermission(ctx, user, modulekit.PermEditAny(manifest.ContentType.ToString()))
}

// StickTopic handles POST /forum/topic/{id}/sticky: sticky=1 pins the
// topic at the top of its section, anything else unpins it; back to the
// topic.
func (c *ForumController) StickTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || nodeID <= 0 {
		c.RenderNotFound(w, r)
		return
	}
	node, err := c.nodeRepo.ByID(ctx, nodeID)
	if err != nil {
		c.RenderLoadFailure(w, r, err, "loading the content", "nodeID", nodeID)
		return
	}
	if node == nil || node.Type != manifest.ContentType {
		c.RenderNotFound(w, r)
		return
	}
	user, err := c.GetCurrentUser(r)
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	if !c.canStickTopics(ctx, user) {
		c.RenderForbidden(w, r)
		return
	}
	if err := c.forumService.SetTopicSticky(ctx, *node, r.FormValue("sticky") == "1"); err != nil {
		c.LogFailure(err, "pinning the topic", "nid", nodeID)
		httputil.InternalServerError(w)
		return
	}
	http.Redirect(w, r, "/"+modulekit.NodeRelPath(node.Type, node.Slug), http.StatusSeeOther)
}
