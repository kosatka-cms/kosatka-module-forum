package forum

import (
	"database/sql"
	"time"

	"github.com/kosatka-cms/kosatka-module-forum/ent/generated"
	"github.com/kosatka-cms/kosatka/modulekit"
)

// ForumCategoryRowDTO represents a row from GetForumCategories query.
type ForumCategoryRowDTO struct {
	ContainerID     int
	ContainerName   string
	ContainerDesc   string
	ContainerSlug   string
	ContainerWeight int
	SectionID       sql.NullInt64
	SectionName     sql.NullString
	SectionDesc     sql.NullString
	SectionSlug     sql.NullString
	SectionWeight   sql.NullInt64
	TopicsCount     int
	PostsCount      int
	LastActivity    sql.NullTime
	LastTopicTitle  sql.NullString
	LastTopicSlug   sql.NullString
	LastAuthor      sql.NullString
}

// ForumTopicRowDTO represents a row from GetTopicsBySection query.
type ForumTopicRowDTO struct {
	ID                int
	Title             string
	Slug              string
	CreatedAt         time.Time
	AuthorID          int
	AuthorName        string
	Sticky            bool
	CommentCount      int
	LastCommentAt     time.Time
	ViewsCount        int
	LastCommentAuthor sql.NullString
	SectionName       string
	SectionSlug       string
}

// ForumTopicDetailDTO represents a topic from GetTopicBySlug query.
type ForumTopicDetailDTO struct {
	ID            int
	Title         string
	Slug          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	AuthorID      int
	AuthorName    string
	AuthorCreated time.Time
	Sticky        bool
	CommentCount  int
	LastCommentAt time.Time
	ViewsCount    int
	Content       string
}

// ForumCommentRowDTO represents a comment row from GetTopicBySlug query.
type ForumCommentRowDTO struct {
	ID            int
	ParentID      sql.NullInt64
	CreatedAt     time.Time
	Content       string
	AuthorID      int
	AuthorName    string
	AuthorCreated time.Time
}

// ForumStatisticsDTO represents statistics from GetForumStatistics query.
type ForumStatisticsDTO struct {
	TotalTopics int
	TotalPosts  sql.NullInt64
	TotalUsers  int
}

// ToForumCategoriesEntity converts DTO rows to entity categories.
func ToForumCategoriesEntity(rows []ForumCategoryRowDTO) []ForumCategory {
	categoriesMap := make(map[int]*ForumCategory)
	var categoriesOrder []int

	for _, row := range rows {
		category, exists := categoriesMap[row.ContainerID]
		if !exists {
			category = &ForumCategory{
				ID:          row.ContainerID,
				Name:        row.ContainerName,
				Description: row.ContainerDesc,
				Slug:        row.ContainerSlug,
				Weight:      row.ContainerWeight,
				Sections:    []ForumSection{},
			}
			categoriesMap[row.ContainerID] = category
			categoriesOrder = append(categoriesOrder, row.ContainerID)
		}

		if row.SectionID.Valid {
			section := ToForumSectionEntity(row)
			category.Sections = append(category.Sections, section)
		}
	}

	var categories []ForumCategory
	for _, id := range categoriesOrder {
		categories = append(categories, *categoriesMap[id])
	}

	return categories
}

// ToForumSectionEntity converts a DTO row to a ForumSection entity.
func ToForumSectionEntity(row ForumCategoryRowDTO) ForumSection {
	section := ForumSection{
		ID:          int(row.SectionID.Int64),
		ParentID:    row.ContainerID,
		Name:        row.SectionName.String,
		Description: row.SectionDesc.String,
		Slug:        row.SectionSlug.String,
		Weight:      int(row.SectionWeight.Int64),
		TopicsCount: row.TopicsCount,
		PostsCount:  row.PostsCount,
	}

	if row.LastActivity.Valid && row.LastTopicTitle.Valid {
		section.LastTopic = &ForumLastTopic{
			Title:     row.LastTopicTitle.String,
			Slug:      row.LastTopicSlug.String,
			Author:    row.LastAuthor.String,
			Timestamp: row.LastActivity.Time,
		}
	}

	return section
}

// ToForumTopicsEntity converts DTO rows to entity topics.
func ToForumTopicsEntity(rows []ForumTopicRowDTO) []ForumTopic {
	topics := make([]ForumTopic, len(rows))
	for i, row := range rows {
		topics[i] = ToForumTopicEntity(row)
	}
	return topics
}

// ToForumTopicEntity converts a DTO row to a ForumTopic entity.
func ToForumTopicEntity(row ForumTopicRowDTO) ForumTopic {
	topic := ForumTopic{
		ID:           row.ID,
		Title:        row.Title,
		Slug:         row.Slug,
		CreatedAt:    row.CreatedAt,
		Sticky:       row.Sticky,
		CommentCount: row.CommentCount,
		ViewsCount:   row.ViewsCount,
		IsHot:        row.CommentCount > 20,
	}

	if row.SectionSlug != "" {
		topic.Section = &ForumTopicSection{Name: row.SectionName, Slug: row.SectionSlug}
	}

	topic.Author = &modulekit.User{
		ID:       row.AuthorID,
		Username: row.AuthorName,
	}

	if row.LastCommentAuthor.Valid {
		topic.LastComment = &ForumLastComment{
			Author:    row.LastCommentAuthor.String,
			Timestamp: row.LastCommentAt,
		}
	}

	return topic
}

// ToForumStatisticsEntity converts DTO to ForumStatistics entity.
func ToForumStatisticsEntity(dto ForumStatisticsDTO, latestUsers []string) ForumStatistics {
	stats := ForumStatistics{
		TotalTopics: dto.TotalTopics,
		TotalUsers:  dto.TotalUsers,
		LatestUsers: latestUsers,
	}

	if dto.TotalPosts.Valid {
		stats.TotalPosts = int(dto.TotalPosts.Int64)
	}

	return stats
}

// ToForumTopicDetail - a topic from its node (the core's), its author
// (nil - the account is gone), its stats row (nil - none yet) and body.
func ToForumTopicDetail(n modulekit.Node, author *modulekit.User, stats *generated.ForumStats, body string) ForumTopicDetail {
	topic := ForumTopicDetail{
		ID:         n.ID,
		Title:      n.Title,
		Slug:       n.Slug,
		Content:    body,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
		Comments:   []modulekit.Comment{},
		ViewsCount: n.Counter.TotalCount,
		node:       n,
	}
	// Falls back to a placeholder if the author account was deleted.
	if author != nil {
		topic.Author = author
	} else {
		topic.Author = modulekit.DeletedUser()
	}
	if stats != nil {
		topic.Sticky = stats.Sticky != 0
		topic.CommentCount = stats.CommentCount
		topic.LastCommentAt = stats.LastCommentAt
	}
	return topic
}
