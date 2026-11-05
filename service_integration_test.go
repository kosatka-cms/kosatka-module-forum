//go:build dbintegration

package forum

import (
	"context"
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

// TestForumService - the forum's pages through the service on a real
// database: the index (categories with their sections and statistics), a
// section found by its slug or by a path with or without the vocabulary's
// prefix, a category, topics (by section, all, unanswered), a topic by its
// slug (a missing one is not found), the section picker; a topic created,
// then edited (saved twice unchanged too) and moved, pinned - each in the
// action log once.
func TestForumService(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		seedUser(t, db)
		seedRow(t, db, `INSERT INTO taxonomy_vocabulary (id, name, slug, term_path_prefix) VALUES (1, 'Forum', 'forum', '/forum/')`)
		seedRow(t, db, `UPDATE forum_settings SET vocabulary_id = 1`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight) VALUES (1, 1, 'General', 'general', 0)`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight, parent_id) VALUES (2, 1, 'Talk', 'talk', 1, 1), (3, 1, 'Help', 'help', 0, 1)`)
		svc := NewForumService(newForumRepository(testkit.Deps(db)), testkit.Logger())
		actions := testkit.CaptureActions(t)

		topic, err := svc.CreateTopic(ctx, 2, "Hello", "first", 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateTopic(ctx, 3, "Question", "how?", 1); err != nil {
			t.Fatal(err)
		}

		index, err := svc.GetForumIndex(ctx)
		if err != nil || len(index.Categories) != 1 || len(index.Categories[0].Sections) != 2 || index.Statistics == nil || index.Statistics.TotalTopics != 2 {
			t.Fatalf("index: %+v %v", index, err)
		}
		if stats, err := svc.GetStatistics(ctx); err != nil || stats.TotalTopics != 2 {
			t.Errorf("statistics: %+v %v", stats, err)
		}
		if c, err := svc.GetCategory(ctx, 1); err != nil || c.Name != "General" {
			t.Errorf("category: %+v %v", c, err)
		}
		if _, err := svc.GetCategory(ctx, 99); !modulekit.IsNotFound(err) {
			t.Errorf("missing category: %v", err)
		}

		if page, total, err := svc.GetSectionBySlug(ctx, "talk"); err != nil || page.Section.ID != 2 || total != 1 || page.TotalCount != 1 {
			t.Errorf("section: %+v %d %v", page, total, err)
		}
		if page, _, err := svc.GetSectionBySlug(ctx, "help"); err != nil || page.Section.ID != 3 {
			t.Errorf("section by slug: %+v %v", page, err)
		}
		if _, _, err := svc.GetSectionBySlug(ctx, "nope"); !modulekit.IsNotFound(err) {
			t.Errorf("missing section: %v", err)
		}

		if topics, err := svc.GetTopics(ctx, 2, ForumTopicFilter{Limit: 10}); err != nil || len(topics) != 1 || topics[0].ID != topic.ID {
			t.Errorf("section topics: %+v %v", topics, err)
		}
		if n, err := svc.CountTopics(ctx, 3); err != nil || n != 1 {
			t.Errorf("count: %d %v", n, err)
		}
		if topics, total, err := svc.ListTopics(ctx, ForumTopicFilter{Limit: 10}); err != nil || total != 2 || len(topics) != 2 {
			t.Errorf("all topics: %d %d %v", len(topics), total, err)
		}
		if topics, total, err := svc.ListTopics(ctx, ForumTopicFilter{Unanswered: true, Limit: 1}); err != nil || total != 2 || len(topics) != 1 {
			t.Errorf("unanswered, a page of one: %d %d %v", len(topics), total, err)
		}

		if got, err := svc.GetTopic(ctx, topic.Slug); err != nil || got.Title != "Hello" || got.Content != "first" {
			t.Errorf("topic: %+v %v", got, err)
		}
		if _, err := svc.GetTopic(ctx, "no-such-topic"); !modulekit.IsNotFound(err) {
			t.Errorf("missing topic: %v", err)
		}
		if opts, err := svc.GetForumTermsForSelect(ctx); err != nil || len(opts) != 3 {
			t.Errorf("section picker: %+v %v", opts, err)
		}

		for range 2 {
			if err := svc.UpdateTopic(ctx, topic.ID, "Hello again", "edited", 3); err != nil {
				t.Fatal(err)
			}
		}
		if sid, err := svc.GetTopicSectionID(ctx, topic.ID); err != nil || sid != 3 {
			t.Errorf("moved to: %d %v", sid, err)
		}
		if err := svc.SetTopicSticky(ctx, modulekit.Node{ID: topic.ID, Title: "Hello again"}, true); err != nil {
			t.Fatal(err)
		}
		var types []string
		for _, a := range *actions {
			types = append(types, a.Type)
		}
		if len(types) != 4 || types[0] != "node.create" || types[1] != "node.create" || types[2] != ActionTopicMove || types[3] != ActionTopicPin {
			t.Errorf("the action log: %v", types)
		} else if c := (*actions)[2].Changes; c[0].From != "Talk" || c[0].To != "Help" || (*actions)[2].Object.Label != "Hello again" {
			t.Errorf("the move: %+v", (*actions)[2])
		}
		if got, err := svc.GetTopic(ctx, topic.Slug); err != nil || got.Title != "Hello again" || got.Content != "edited" {
			t.Errorf("edited topic: %+v %v", got, err)
		}
		if n, err := svc.CountTopics(ctx, 2); err != nil || n != 0 {
			t.Errorf("the old section's topics: %d %v", n, err)
		}
	})
}
