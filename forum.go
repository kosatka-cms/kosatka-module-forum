package forum

import (
	"context"
	"time"

	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
)

// ForumCategory represents a top-level forum category (container)
type ForumCategory struct {
	ID          int
	Name        string
	Description string
	Slug        string
	Weight      int
	Sections    []ForumSection
}

// ForumSection represents a forum section within a category
type ForumSection struct {
	ID          int
	ParentID    int
	Name        string
	Description string
	Slug        string
	Weight      int
	TopicsCount int
	PostsCount  int
	LastTopic   *ForumLastTopic
	// Parent category for breadcrumbs
	ParentName string
	ParentSlug string
}

// ForumLastTopic represents the last topic info in a section
type ForumLastTopic struct {
	ID        int
	Title     string
	Slug      string
	Author    string
	Timestamp time.Time
}

// ForumTopic represents a forum topic in a list
type ForumTopic struct {
	ID           int
	Title        string
	Slug         string
	Author       *modulekit.User
	CreatedAt    time.Time
	Sticky       bool
	CommentCount int
	ViewsCount   int
	LastComment  *ForumLastComment
	// Section - the topic's section, in lists across sections (nil in a
	// section's own list).
	Section *ForumTopicSection
	IsHot   bool
}

// ForumTopicSection - a section as a topic list names it.
type ForumTopicSection struct {
	Name string
	Slug string
}

// ForumTopicDetail represents detailed forum topic info
type ForumTopicDetail struct {
	ID            int
	Title         string
	Slug          string
	Content       string
	Author        *modulekit.User
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Sticky        bool
	ViewsCount    int
	CommentCount  int
	LastCommentAt time.Time
	Comments      []modulekit.Comment
	// Section for breadcrumbs (includes parent category)
	Section *ForumSection

	// node - the topic's node, for the view check (AuthService.CanViewNode).
	node modulekit.Node
}

// ForumTermOption represents a forum term for the section select dropdown
type ForumTermOption struct {
	ID          int
	Name        string
	ParentID    int
	DisplayName string // With "- " prefix for sections
}

// ForumLastComment represents the last comment info
type ForumLastComment struct {
	ID        int
	Author    string
	Timestamp time.Time
}

// ForumStatistics represents overall forum statistics
type ForumStatistics struct {
	TotalTopics      int
	TotalPosts       int
	TotalUsers       int
	LatestUsers      []string
	ActiveUsersCount int
}

// ForumTopicFilter represents filter options for topics
type ForumTopicFilter struct {
	// SectionID - the section's topics (0 - every section's).
	SectionID int
	// Unanswered - only topics without replies.
	Unanswered bool
	Offset     int
	Limit      int
	OrderBy    string
	OrderDir   string
	Sticky     *bool
}

// ForumIndex represents the main forum page data
type ForumIndex struct {
	Categories []ForumCategory
	Statistics *ForumStatistics
}

// ForumSectionPage represents a forum section page data
type ForumSectionPage struct {
	Section    *ForumSection
	Topics     []ForumTopic
	TotalCount int
}

// ForumRepository defines methods for forum data access
type ForumRepository interface {
	GetForumCategories(ctx context.Context) ([]ForumCategory, error)
	GetSectionBySlug(ctx context.Context, slug string) (*ForumSection, error)
	GetTopicsBySection(ctx context.Context, sectionID int, filter ForumTopicFilter) ([]ForumTopic, error)
	CountTopicsBySection(ctx context.Context, sectionID int) (int, error)
	// ListTopics - topics by the filter (one section or all, optionally
	// only unanswered), each with its section when listing all.
	ListTopics(ctx context.Context, filter ForumTopicFilter) ([]ForumTopic, error)
	// CountTopicsFiltered - how many topics ListTopics pages over.
	CountTopicsFiltered(ctx context.Context, filter ForumTopicFilter) (int, error)
	GetTopicBySlug(ctx context.Context, slug string) (*ForumTopicDetail, error)
	GetForumStatistics(ctx context.Context) (*ForumStatistics, error)
	GetForumTermsForSelect(ctx context.Context) ([]ForumTermOption, error)
	GetTopicSectionID(ctx context.Context, nodeID int) (int, error)
	CreateTopic(ctx context.Context, sectionID int, title, body string, authorID int) (*ForumTopicDetail, error)
	UpdateTopic(ctx context.Context, nodeID int, title, body string, sectionID int) error
	// SetSticky pins the topic at the top of its section's list (or not).
	SetSticky(ctx context.Context, nodeID int, sticky bool) error
	TopicExistsBySlug(ctx context.Context, slug string) (bool, error)
	// SectionPathPrefix - the forum vocabulary's term path prefix (its
	// sections' URLs start with it, e.g. "forums"), "" - none set.
	SectionPathPrefix(ctx context.Context) (string, error)
}

// topicPath - a topic's page: its slug under the forum's path prefix
// (modulekit.NodeRelPath), so a bare slug and a "forum/…" one both resolve.
func topicPath(slug string) string {
	if rel := modulekit.NodeRelPath(manifest.ContentType, slug); rel != "" {
		return "/" + rel
	}
	return ""
}

// Path - the topic's page.
func (t ForumTopic) Path() string { return topicPath(t.Slug) }

// Path - the topic's page.
func (t ForumTopicDetail) Path() string { return topicPath(t.Slug) }

// Path - the topic's page.
func (t ForumLastTopic) Path() string { return topicPath(t.Slug) }
