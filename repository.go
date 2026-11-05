package forum

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kosatka-cms/kosatka-module-forum/ent/generated"
	"github.com/kosatka-cms/kosatka-module-forum/ent/generated/forumstats"
	"github.com/kosatka-cms/kosatka-module-forum/ent/generated/forumtopic"
	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/pkg/slug"
)

// Taxonomy - the core's vocabularies and terms the forum's sections are
// (Deps.Taxonomy).
type Taxonomy interface {
	VocabularyByID(ctx context.Context, id int) (*modulekit.TaxonomyVocabulary, error)
	TermByID(ctx context.Context, id int) (*modulekit.TaxonomyTerm, error)
	TermBySlug(ctx context.Context, slug string) (*modulekit.TaxonomyTerm, error)
	TermsByVocabularyID(ctx context.Context, vocabularyID int) (modulekit.TaxonomyTerms, error)
	NodeTermsByID(ctx context.Context, nodeID int) (modulekit.TaxonomyTerms, error)
	UpdateNodeTerms(ctx context.Context, nodeID int, termIDs []int) error
}

// NewForumRepository creates a new forum repository: its own tables on db
// (the module's own ent schema, ent/), a topic's node, author and section
// through the core; its lists in SQL over both.
// A built-in module, tested with the engine, may read the core's tables
// in its SQL; a module of its own repository goes through the SDK's
// services instead (docs/modules/sdk.md): the core's columns aren't its
// API.
func NewForumRepository(db *sql.DB, nodes modulekit.NodesRepository, users modulekit.UserRepository, taxonomy Taxonomy) *EntForumRepository {
	return &EntForumRepository{db: db, nodes: nodes, users: users, taxonomy: taxonomy}
}

// EntForumRepository handles forum data access.
type EntForumRepository struct {
	db       *sql.DB
	nodes    modulekit.NodesRepository
	users    modulekit.UserRepository
	taxonomy Taxonomy
}

// client - the module's client for ctx: in its transaction, if any.
func (r *EntForumRepository) client(ctx context.Context) *generated.Client {
	return generated.NewClient(generated.Driver(modulekit.Driver(ctx, r.db)))
}

// settingsRowID - the id of forum_settings' single row.
const settingsRowID = 1

// SectionPathPrefix - see ForumRepository.SectionPathPrefix.
func (r *EntForumRepository) SectionPathPrefix(ctx context.Context) (string, error) {
	vid, err := r.vocabularyID(ctx)
	if err != nil || vid == 0 {
		return "", err
	}
	v, err := r.taxonomy.VocabularyByID(ctx, vid)
	if modulekit.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.Trim(v.TermPathPrefix, "/"), nil
}

// vocabularyID - the forum vocabulary (ForumSettings); 0 when none is set.
func (r *EntForumRepository) vocabularyID(ctx context.Context) (int, error) {
	s, err := r.client(ctx).ForumSettings.Get(ctx, settingsRowID)
	if generated.IsNotFound(err) || (err == nil && s.VocabularyID == nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return *s.VocabularyID, nil
}

// toForumSection - a forum term with its parent category (for
// breadcrumbs), the parent read when it is in the same vocabulary.
func (r *EntForumRepository) toForumSection(ctx context.Context, t modulekit.TaxonomyTerm) (*ForumSection, error) {
	section := &ForumSection{ID: t.ID, Name: t.Name, Description: t.Desc, Slug: t.Slug, Weight: t.Weight}
	if t.ParentID == 0 {
		return section, nil
	}
	section.ParentID = t.ParentID
	p, err := r.taxonomy.TermByID(ctx, t.ParentID)
	if modulekit.IsNotFound(err) || (err == nil && p.Vocabulary.ID != t.Vocabulary.ID) {
		return section, nil
	}
	if err != nil {
		return nil, err
	}
	section.ParentName, section.ParentSlug = p.Name, p.Slug
	return section, nil
}

// GetForumCategories retrieves all forum categories with sections and statistics.
func (r *EntForumRepository) GetForumCategories(ctx context.Context) ([]ForumCategory, error) {
	query := `
	SELECT 
		container.id as container_id,
		container.name as container_name,
		COALESCE(container.description, '') as container_description,
		container.slug as container_slug,
		container.weight as container_weight,
		section.id as section_id,
		section.name as section_name,
		section.description as section_description,
		section.slug as section_slug,
		section.weight as section_weight,
		COUNT(DISTINCT n.id) as topics_count,
		COALESCE(SUM(fs.comment_count), 0) as posts_count,
		MAX(fs.last_comment_at) as last_activity,
		(SELECT n2.title FROM nodes n2 
		 INNER JOIN forum_stats fs2 ON fs2.nid = n2.id
		 WHERE n2.id IN (SELECT ti.nid FROM taxonomy_index ti WHERE ti.tid = section.id)
		   AND n2.type = ? AND n2.status = 1 AND n2.on_moderation = 0
		 ORDER BY fs2.last_comment_at DESC LIMIT 1) as last_topic_title,
		(SELECT n3.slug FROM nodes n3 
		 INNER JOIN forum_stats fs3 ON fs3.nid = n3.id
		 WHERE n3.id IN (SELECT ti.nid FROM taxonomy_index ti WHERE ti.tid = section.id)
		   AND n3.type = ? AND n3.status = 1 AND n3.on_moderation = 0
		 ORDER BY fs3.last_comment_at DESC LIMIT 1) as last_topic_slug,
		(SELECT u.username FROM nodes n4 
		 INNER JOIN forum_stats fs4 ON fs4.nid = n4.id
		 INNER JOIN users u ON u.id = n4.created_by
		 WHERE n4.id IN (SELECT ti.nid FROM taxonomy_index ti WHERE ti.tid = section.id)
		   AND n4.type = ? AND n4.status = 1 AND n4.on_moderation = 0
		 ORDER BY fs4.last_comment_at DESC LIMIT 1) as last_author
	FROM taxonomy_term_data container
	LEFT JOIN taxonomy_term_data section ON section.parent_id = container.id
	LEFT JOIN taxonomy_index ti ON ti.tid = section.id
	LEFT JOIN nodes n ON n.id = ti.nid AND n.type = ? AND n.status = 1 AND n.on_moderation = 0
	LEFT JOIN forum_stats fs ON fs.nid = n.id
	WHERE container.parent_id IS NULL
	  AND container.vid = ` + forumVocabularySQL + `
	GROUP BY container.id, container.name, container.description, container.slug, container.weight,
			 section.id, section.name, section.description, section.slug, section.weight
	ORDER BY container.weight, container.name, section.weight, section.name
	`

	forum := string(manifest.ContentType)
	rows, err := r.db.QueryContext(ctx, modulekit.Rebind(query), forum, forum, forum, forum)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var dtoRows []ForumCategoryRowDTO
	for rows.Next() {
		var row ForumCategoryRowDTO
		err := rows.Scan(
			&row.ContainerID, &row.ContainerName, &row.ContainerDesc, &row.ContainerSlug, &row.ContainerWeight,
			&row.SectionID, &row.SectionName, &row.SectionDesc, &row.SectionSlug, &row.SectionWeight,
			&row.TopicsCount, &row.PostsCount, &row.LastActivity,
			&row.LastTopicTitle, &row.LastTopicSlug, &row.LastAuthor,
		)
		if err != nil {
			return nil, err
		}
		dtoRows = append(dtoRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ToForumCategoriesEntity(dtoRows), nil
}

// GetSectionBySlug retrieves a forum section by slug with its parent category
// (for breadcrumbs).
func (r *EntForumRepository) GetSectionBySlug(ctx context.Context, slug string) (*ForumSection, error) {
	vid, err := r.vocabularyID(ctx)
	if err != nil {
		return nil, err
	}
	t, err := r.taxonomy.TermBySlug(ctx, slug)
	if modulekit.IsNotFound(err) || (err == nil && t.Vocabulary.ID != vid) {
		return nil, fmt.Errorf("%w: forum section %s", modulekit.ErrEntityNotFound, slug)
	}
	if err != nil {
		return nil, err
	}
	return r.toForumSection(ctx, *t)
}

// GetTopicsBySection retrieves topics of a forum section.
func (r *EntForumRepository) GetTopicsBySection(ctx context.Context, sectionID int, filter ForumTopicFilter) ([]ForumTopic, error) {
	filter.SectionID = sectionID
	return r.ListTopics(ctx, filter)
}

// forumTopicsWhere - the FROM/WHERE part of the topic lists and its
// arguments: public forum topics (published, not on moderation), of one
// section or all, optionally only those without replies.
func forumTopicsWhere(filter ForumTopicFilter) (string, []any) {
	from := `
	FROM nodes n
	INNER JOIN forum_stats fs ON fs.nid = n.id
	INNER JOIN users u ON u.id = n.created_by
	LEFT JOIN node_counter nc ON nc.nid = n.id`
	where := `
	WHERE n.type = ?
	  AND n.status = 1
	  AND n.on_moderation = 0`
	args := []any{string(manifest.ContentType)}
	if filter.SectionID > 0 {
		from += `
	INNER JOIN taxonomy_index ti ON ti.nid = n.id`
		where += `
	  AND ti.tid = ?`
		args = append(args, filter.SectionID)
	}
	if filter.Unanswered {
		where += `
	  AND fs.comment_count = 0`
	}
	return from + where, args
}

// ListTopics retrieves topics by the filter.
func (r *EntForumRepository) ListTopics(ctx context.Context, filter ForumTopicFilter) ([]ForumTopic, error) {
	// ORDER BY from fixed pieces only: the column and the direction come
	// from the visitor's URL.
	dir := "DESC"
	if strings.EqualFold(filter.OrderDir, "asc") {
		dir = "ASC"
	}
	orderBy := "fs.last_comment_at DESC"
	switch filter.OrderBy {
	case "title":
		orderBy = "n.title " + dir
	case "comment_count":
		orderBy = "fs.comment_count " + dir
	case "totalcount":
		orderBy = "COALESCE(nc.totalcount, 0) " + dir
	case "last_updated":
		orderBy = "fs.last_comment_at " + dir
	case "created":
		orderBy = "n.created_at " + dir
	}
	// Stickies lead a section's list only; the ID keeps pages stable.
	if filter.SectionID > 0 {
		orderBy = "fs.sticky DESC, " + orderBy
	}
	orderBy += ", n.id DESC"

	// A topic's section (when listing all): its term in the forum
	// vocabulary.
	section := `''
		, ''`
	if filter.SectionID == 0 {
		section = `COALESCE((SELECT td.name FROM taxonomy_index ti2
		 INNER JOIN taxonomy_term_data td ON td.id = ti2.tid
		 WHERE ti2.nid = n.id AND td.vid = ` + forumVocabularySQL + `
		 ORDER BY td.id LIMIT 1), '')
		, COALESCE((SELECT td.slug FROM taxonomy_index ti2
		 INNER JOIN taxonomy_term_data td ON td.id = ti2.tid
		 WHERE ti2.nid = n.id AND td.vid = ` + forumVocabularySQL + `
		 ORDER BY td.id LIMIT 1), '')`
	}

	fromWhere, args := forumTopicsWhere(filter)
	query := `
	SELECT
		n.id,
		n.title,
		n.slug,
		n.created_at,
		u.id as author_id,
		u.username as author_name,
		fs.sticky,
		fs.comment_count,
		fs.last_comment_at,
		COALESCE(nc.totalcount, 0) as views_count,
		(SELECT u2.username FROM comments c2
		 INNER JOIN users u2 ON u2.id = c2.created_by
		 WHERE c2.nid = n.id AND c2.status = 1
		 ORDER BY c2.created_at DESC LIMIT 1) as last_comment_author,
		` + section + `
	` + fromWhere + `
	ORDER BY ` + orderBy + `
	LIMIT ? OFFSET ?
	`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, modulekit.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var dtoRows []ForumTopicRowDTO
	for rows.Next() {
		var row ForumTopicRowDTO
		err := rows.Scan(
			&row.ID, &row.Title, &row.Slug, &row.CreatedAt,
			&row.AuthorID, &row.AuthorName,
			&row.Sticky, &row.CommentCount, &row.LastCommentAt, &row.ViewsCount,
			&row.LastCommentAuthor, &row.SectionName, &row.SectionSlug,
		)
		if err != nil {
			return nil, err
		}
		dtoRows = append(dtoRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ToForumTopicsEntity(dtoRows), nil
}

// CountTopicsFiltered counts the topics ListTopics pages over.
func (r *EntForumRepository) CountTopicsFiltered(ctx context.Context, filter ForumTopicFilter) (int, error) {
	fromWhere, args := forumTopicsWhere(filter)
	var n int
	err := r.db.QueryRowContext(ctx, modulekit.Rebind(`SELECT COUNT(*) `+fromWhere), args...).Scan(&n)
	return n, err
}

// CountTopicsBySection counts the public topics of a section.
func (r *EntForumRepository) CountTopicsBySection(ctx context.Context, sectionID int) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, modulekit.Rebind(`SELECT COUNT(*) FROM nodes n
		INNER JOIN taxonomy_index ti ON ti.nid = n.id
		WHERE n.type = ? AND n.status = 1 AND n.on_moderation = 0 AND ti.tid = ?`), string(manifest.ContentType), sectionID).Scan(&n)
	return n, err
}

// GetTopicBySlug retrieves a topic by slug, or by the path of its page
// (a slug without the type's prefix has it added there, NodeRelPath),
// whatever its status: who may see it is the page's to decide
// (AuthService.CanViewNode on ForumTopicDetail.node).
func (r *EntForumRepository) GetTopicBySlug(ctx context.Context, slug string) (*ForumTopicDetail, error) {
	n, err := r.nodes.BySlug(ctx, slug)
	if bare, ok := strings.CutPrefix(slug, modulekit.ModuleNodeTypePathPrefix(string(manifest.ContentType))); ok && bare != "" && modulekit.IsNotFound(err) {
		n, err = r.nodes.BySlug(ctx, bare)
	}
	if err == nil && n.Type != manifest.ContentType {
		err = modulekit.ErrEntityNotFound
	}
	if modulekit.IsNotFound(err) {
		return nil, fmt.Errorf("%w: forum topic with slug %s", modulekit.ErrEntityNotFound, slug)
	}
	if err != nil {
		return nil, err
	}

	// The author in full (the avatar too); a deleted account - none.
	var author *modulekit.User
	if n.CreatedBy.ID > 0 {
		u, err := r.users.ByID(ctx, n.CreatedBy.ID)
		if err != nil && !modulekit.IsNotFound(err) {
			return nil, err
		}
		author = u
	}
	client := r.client(ctx)
	stats, err := client.ForumStats.Query().Where(forumstats.Nid(n.ID)).Only(ctx)
	if err != nil && !generated.IsNotFound(err) {
		return nil, err
	}
	// The topic body, when there is one.
	var body string
	if ft, err := client.ForumTopic.Query().Where(forumtopic.Nid(n.ID)).First(ctx); err == nil {
		body = ft.Body
	}

	// Comments are loaded separately by renderTopicPage via commentsService.ByNodeIDPaginated
	topic := ToForumTopicDetail(*n, author, stats, body)

	// The section, for breadcrumbs (a forum term of the topic).
	vid, vidErr := r.vocabularyID(ctx)
	terms, termsErr := r.taxonomy.NodeTermsByID(ctx, n.ID)
	if vidErr == nil && termsErr == nil {
		for _, t := range terms {
			if t.Vocabulary.ID != vid {
				continue
			}
			// The term in full (its parent: the section's category).
			if full, err := r.taxonomy.TermByID(ctx, t.ID); err == nil {
				topic.Section, _ = r.toForumSection(ctx, *full)
			}
			break
		}
	}

	return &topic, nil
}

// GetForumStatistics retrieves overall forum statistics.
func (r *EntForumRepository) GetForumStatistics(ctx context.Context) (*ForumStatistics, error) {
	query := `
	SELECT
		(SELECT COUNT(*) FROM nodes WHERE type = ? AND status = 1 AND on_moderation = 0) as total_topics,
		(SELECT SUM(fs.comment_count) FROM forum_stats fs
			INNER JOIN nodes n ON n.id = fs.nid
			WHERE n.status = 1 AND n.on_moderation = 0) as total_posts,
		(SELECT COUNT(*) FROM users) as total_users
	`

	var statsDTO ForumStatisticsDTO
	err := r.db.QueryRowContext(ctx, modulekit.Rebind(query), string(manifest.ContentType)).Scan(
		&statsDTO.TotalTopics, &statsDTO.TotalPosts, &statsDTO.TotalUsers,
	)
	if err != nil {
		return nil, err
	}

	// The latest users; the statistics show without them on an error.
	latestUsers, err := r.latestUsernames(ctx, 5)
	if err != nil {
		latestUsers = nil
	}

	stats := ToForumStatisticsEntity(statsDTO, latestUsers)
	return &stats, nil
}

// latestUsernames - the names of the n newest active users (not a
// blocked account, nor one awaiting approval - a spammer's often).
func (r *EntForumRepository) latestUsernames(ctx context.Context, n int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, modulekit.Rebind(`SELECT username FROM users WHERE status = ? ORDER BY created_at DESC, id DESC LIMIT ?`), modulekit.UserStatusActive, n)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// GetForumTermsForSelect returns all forum terms (categories and sections)
// for the section dropdown.
func (r *EntForumRepository) GetForumTermsForSelect(ctx context.Context) ([]ForumTermOption, error) {
	vid, err := r.vocabularyID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.taxonomy.TermsByVocabularyID(ctx, vid)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]modulekit.TaxonomyTerm, len(rows))
	for _, t := range rows {
		byID[t.ID] = t
	}
	// A section follows its category: ordered by the category's weight and
	// name, the category first, then the sections by weight and name.
	group := func(t modulekit.TaxonomyTerm) modulekit.TaxonomyTerm {
		if t.ParentID != 0 {
			if p, ok := byID[t.ParentID]; ok {
				return p
			}
		}
		return t
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		ga, gb := group(a), group(b)
		if ga.Weight != gb.Weight {
			return ga.Weight < gb.Weight
		}
		if ga.Name != gb.Name {
			return ga.Name < gb.Name
		}
		if (a.ParentID == 0) != (b.ParentID == 0) {
			return a.ParentID == 0
		}
		if a.Weight != b.Weight {
			return a.Weight < b.Weight
		}
		return a.Name < b.Name
	})
	terms := make([]ForumTermOption, 0, len(rows))
	for _, t := range rows {
		opt := ForumTermOption{ID: t.ID, Name: t.Name, DisplayName: t.Name}
		if t.ParentID != 0 {
			opt.ParentID = t.ParentID
			opt.DisplayName = "- " + t.Name
		}
		terms = append(terms, opt)
	}
	return terms, nil
}

// GetTopicSectionID returns the taxonomy term ID (section) of a forum topic.
func (r *EntForumRepository) GetTopicSectionID(ctx context.Context, nodeID int) (int, error) {
	terms, err := r.taxonomy.NodeTermsByID(ctx, nodeID)
	if err != nil {
		return 0, err
	}
	if len(terms) == 0 {
		return 0, fmt.Errorf("%w: the section of topic %d", modulekit.ErrEntityNotFound, nodeID)
	}
	return terms[0].ID, nil
}

// TopicExistsBySlug reports whether a forum topic with the given slug exists.
func (r *EntForumRepository) TopicExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return r.nodes.SlugExists(ctx, slug)
}

// CreateTopic creates a new forum topic.
func (r *EntForumRepository) CreateTopic(ctx context.Context, sectionID int, title, body string, authorID int) (*ForumTopicDetail, error) {
	// Under the forum's path prefix, as its topics' pages are (NodeRelPath).
	baseSlug := slug.Generate(title)
	if baseSlug == "" {
		baseSlug = fmt.Sprintf("topic-%d", time.Now().Unix())
	}
	baseSlug = modulekit.ModuleNodeTypePathPrefix(string(manifest.ContentType)) + baseSlug
	slug := baseSlug
	for i := 1; ; i++ {
		exists, err := r.TopicExistsBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		if !exists {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	// The node, its body and stats and its section in one transaction.
	err := modulekit.InTx(ctx, r.db, func(ctx context.Context) error {
		now := time.Now()
		nodeID, err := r.nodes.CreateRow(ctx, modulekit.Node{
			Type: manifest.ContentType, Title: title, Slug: slug, Status: 1,
			CreatedBy: modulekit.User{ID: authorID}, CreatedAt: now,
		})
		if err != nil {
			return err
		}
		client := r.client(ctx)
		if err := client.ForumTopic.Create().SetNid(nodeID).SetBody(body).Exec(ctx); err != nil {
			return fmt.Errorf("create forum topic: %w", err)
		}
		if err := client.ForumStats.Create().SetNid(nodeID).SetSticky(0).SetCommentCount(0).SetLastCommentAt(now).Exec(ctx); err != nil {
			return fmt.Errorf("create forum stats: %w", err)
		}
		return r.taxonomy.UpdateNodeTerms(ctx, nodeID, []int{sectionID})
	})
	if err != nil {
		return nil, err
	}

	topic, err := r.GetTopicBySlug(ctx, slug)
	if err != nil {
		return topic, err
	}
	return topic, nil
}

// UpdateTopic updates a forum topic's title, body and section (in one
// transaction).
func (r *EntForumRepository) UpdateTopic(ctx context.Context, nodeID int, title, body string, sectionID int) error {
	return modulekit.InTx(ctx, r.db, func(ctx context.Context) error {
		n, err := r.nodes.ByID(ctx, nodeID)
		if err != nil {
			return fmt.Errorf("the topic's node: %w", err)
		}
		n.Title = title
		if err := r.nodes.UpdateRow(ctx, *n); err != nil {
			return err
		}
		client := r.client(ctx)
		ft, err := client.ForumTopic.Query().Where(forumtopic.NidEQ(nodeID)).Only(ctx)
		switch {
		case err == nil:
			if err := client.ForumTopic.UpdateOne(ft).SetBody(body).Exec(ctx); err != nil {
				return fmt.Errorf("update forum topic: %w", err)
			}
		case generated.IsNotFound(err):
			if err := client.ForumTopic.Create().SetNid(nodeID).SetBody(body).Exec(ctx); err != nil {
				return fmt.Errorf("create forum topic: %w", err)
			}
		default:
			return err
		}
		return r.taxonomy.UpdateNodeTerms(ctx, nodeID, []int{sectionID})
	})
}

// RecountTopicStats sets the topic's comment count and last comment time
// from its comments (nothing for a node that isn't a topic).
func (r *EntForumRepository) RecountTopicStats(ctx context.Context, nodeID int) error {
	_, err := r.db.ExecContext(ctx, modulekit.Rebind(manifest.RecountTopicStatsSQL+" WHERE nid = ?"), nodeID)
	return err
}

// SetSticky - see ForumRepository.SetSticky.
func (r *EntForumRepository) SetSticky(ctx context.Context, nodeID int, sticky bool) error {
	v := 0
	if sticky {
		v = 1
	}
	return r.client(ctx).ForumStats.Update().Where(forumstats.Nid(nodeID)).SetSticky(v).Exec(ctx)
}
