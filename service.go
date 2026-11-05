package forum

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/pkg/i18n"
)

// ForumService defines the forum business logic.
type ForumService interface {
	GetForumIndex(ctx context.Context) (*ForumIndex, error)
	// SectionPathPrefix - the forum vocabulary's term path prefix: the
	// URL prefix of its sections ("" - none).
	SectionPathPrefix(ctx context.Context) string
	// GetSectionBySlug - the section or category whose slug is slug;
	// ErrEntityNotFound when none.
	GetSectionBySlug(ctx context.Context, slug string) (*ForumSectionPage, int, error)
	// GetCategory - a category (root term) with its sections and stats.
	GetCategory(ctx context.Context, id int) (*ForumCategory, error)
	GetTopics(ctx context.Context, sectionID int, filter ForumTopicFilter) ([]ForumTopic, error)
	CountTopics(ctx context.Context, sectionID int) (int, error)
	// ListTopics - a page of topics by the filter and how many there are
	// in all (the active and unanswered lists).
	ListTopics(ctx context.Context, filter ForumTopicFilter) ([]ForumTopic, int, error)
	GetTopic(ctx context.Context, slug string) (*ForumTopicDetail, error)
	GetStatistics(ctx context.Context) (*ForumStatistics, error)
	GetForumTermsForSelect(ctx context.Context) ([]ForumTermOption, error)
	GetTopicSectionID(ctx context.Context, nodeID int) (int, error)
	CreateTopic(ctx context.Context, sectionID int, title, body string, authorID int) (*ForumTopicDetail, error)
	UpdateTopic(ctx context.Context, nodeID int, title, body string, sectionID int) error
	// SetTopicSticky pins the topic at the top of its section (or not).
	SetTopicSticky(ctx context.Context, topic modulekit.Node, sticky bool) error
}

// DefaultForumService implements ForumService.
type DefaultForumService struct {
	repository ForumRepository
	logger     *slog.Logger
}

// NewForumService creates a new forum service.
func NewForumService(r ForumRepository, baseLogger *slog.Logger) ForumService {
	return &DefaultForumService{
		repository: r,
		logger:     baseLogger.With("component", "forum.service"),
	}
}

// GetForumIndex retrieves the main forum page data.
func (s *DefaultForumService) GetForumIndex(ctx context.Context) (*ForumIndex, error) {
	categories, err := s.repository.GetForumCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting forum categories: %w", err)
	}

	statistics, err := s.repository.GetForumStatistics(ctx)
	if err != nil {
		modulekit.LogFallback(s.logger, err, "error getting forum statistics")
		// Don't fail if statistics retrieval fails
		statistics = &ForumStatistics{}
	}

	return &ForumIndex{
		Categories: categories,
		Statistics: statistics,
	}, nil
}

// SectionPathPrefix - see ForumService.SectionPathPrefix.
func (s *DefaultForumService) SectionPathPrefix(ctx context.Context) string {
	prefix, err := s.repository.SectionPathPrefix(ctx)
	if err != nil {
		modulekit.LogFallback(s.logger, err, "error loading forum section path prefix")
		return ""
	}
	return prefix
}

// GetSectionBySlug - see ForumService.GetSectionBySlug.
func (s *DefaultForumService) GetSectionBySlug(ctx context.Context, slug string) (*ForumSectionPage, int, error) {
	section, err := s.repository.GetSectionBySlug(ctx, slug)
	if modulekit.IsNotFound(err) {
		return nil, 0, err
	}
	if err != nil {
		return nil, 0, fmt.Errorf("getting forum section (slug %v): %w", slug, err)
	}

	totalCount, err := s.repository.CountTopicsBySection(ctx, section.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("counting topics (section_id %v): %w", section.ID, err)
	}

	return &ForumSectionPage{
		Section:    section,
		Topics:     []ForumTopic{},
		TotalCount: totalCount,
	}, totalCount, nil
}

// GetCategory - see ForumService.GetCategory.
func (s *DefaultForumService) GetCategory(ctx context.Context, id int) (*ForumCategory, error) {
	categories, err := s.repository.GetForumCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting forum categories: %w", err)
	}
	for i := range categories {
		if categories[i].ID == id {
			return &categories[i], nil
		}
	}
	return nil, modulekit.ErrEntityNotFound
}

// GetTopics retrieves topics for a section with filters.
func (s *DefaultForumService) GetTopics(ctx context.Context, sectionID int, filter ForumTopicFilter) ([]ForumTopic, error) {
	topics, err := s.repository.GetTopicsBySection(ctx, sectionID, filter)
	if err != nil {
		return nil, fmt.Errorf("getting topics (section_id %v): %w", sectionID, err)
	}

	return topics, nil
}

// ListTopics - a page of topics by the filter and their total.
func (s *DefaultForumService) ListTopics(ctx context.Context, filter ForumTopicFilter) ([]ForumTopic, int, error) {
	total, err := s.repository.CountTopicsFiltered(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("counting topics (unanswered %v): %w", filter.Unanswered, err)
	}
	topics, err := s.repository.ListTopics(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("listing topics (unanswered %v): %w", filter.Unanswered, err)
	}
	return topics, total, nil
}

// CountTopics counts topics in a section.
func (s *DefaultForumService) CountTopics(ctx context.Context, sectionID int) (int, error) {
	count, err := s.repository.CountTopicsBySection(ctx, sectionID)
	if err != nil {
		return 0, fmt.Errorf("counting topics (section_id %v): %w", sectionID, err)
	}

	return count, nil
}

// GetTopic retrieves a forum topic by slug.
func (s *DefaultForumService) GetTopic(ctx context.Context, slug string) (*ForumTopicDetail, error) {
	topic, err := s.repository.GetTopicBySlug(ctx, slug)
	if err != nil {
		if modulekit.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("getting topic by slug (slug %v): %w", slug, err)
	}

	return topic, nil
}

// GetStatistics retrieves overall forum statistics.
func (s *DefaultForumService) GetStatistics(ctx context.Context) (*ForumStatistics, error) {
	stats, err := s.repository.GetForumStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting forum statistics: %w", err)
	}

	return stats, nil
}

// GetForumTermsForSelect returns forum terms for the section dropdown.
func (s *DefaultForumService) GetForumTermsForSelect(ctx context.Context) ([]ForumTermOption, error) {
	return s.repository.GetForumTermsForSelect(ctx)
}

// GetTopicSectionID returns the section (tid) for a forum topic.
func (s *DefaultForumService) GetTopicSectionID(ctx context.Context, nodeID int) (int, error) {
	return s.repository.GetTopicSectionID(ctx, nodeID)
}

// CreateTopic creates a new forum topic; the action log gets "Content
// created".
func (s *DefaultForumService) CreateTopic(ctx context.Context, sectionID int, title, body string, authorID int) (*ForumTopicDetail, error) {
	topic, err := s.repository.CreateTopic(ctx, sectionID, title, body, authorID)
	if err != nil {
		return nil, err
	}
	modulekit.RecordNodeCreated(ctx, topicNode(topic.ID, topic.Title))
	return topic, nil
}

// UpdateTopic updates a forum topic; moved to another section, it is
// "Topic moved" in the action log, from which section to which.
func (s *DefaultForumService) UpdateTopic(ctx context.Context, nodeID int, title, body string, sectionID int) error {
	was, err := s.repository.GetTopicSectionID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("the topic's section (node %v): %w", nodeID, err)
	}
	if err := s.repository.UpdateTopic(ctx, nodeID, title, body, sectionID); err != nil {
		return err
	}
	if was != sectionID {
		modulekit.RecordAction(ctx, modulekit.Action{Type: ActionTopicMove, Object: modulekit.NodeActionObject(topicNode(nodeID, title)),
			Changes: []modulekit.ActionChange{{Field: i18n.T(ctx, "Section"), From: s.sectionName(ctx, was), To: s.sectionName(ctx, sectionID)}}})
	}
	return nil
}

// sectionName - the forum section's name for the action log (its ID when
// it can't be found).
func (s *DefaultForumService) sectionName(ctx context.Context, id int) string {
	terms, err := s.repository.GetForumTermsForSelect(ctx)
	if err != nil {
		modulekit.LogFallback(s.logger, err, "forum sections for the action log")
	}
	for _, t := range terms {
		if t.ID == id {
			return t.Name
		}
	}
	return strconv.Itoa(id)
}

// SetTopicSticky pins the topic at the top of its section (or unpins it);
// the action log gets "Topic pinned" or "Topic unpinned".
func (s *DefaultForumService) SetTopicSticky(ctx context.Context, topic modulekit.Node, sticky bool) error {
	if err := s.repository.SetSticky(ctx, topic.ID, sticky); err != nil {
		return fmt.Errorf("setting topic %v sticky=%v: %w", topic.ID, sticky, err)
	}
	action := ActionTopicUnpin
	if sticky {
		action = ActionTopicPin
	}
	modulekit.RecordAction(ctx, modulekit.Action{Type: action, Object: modulekit.NodeActionObject(topic)})
	return nil
}
