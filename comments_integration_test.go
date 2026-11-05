//go:build dbintegration

package forum

import (
	"context"
	"testing"

	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

// TestCommentsChangedRecountsTopic - a topic's reply count and last reply
// time follow its comments through CommentsService (the module's hook):
// a published reply counts, an unpublished one doesn't, a deleted one
// goes; the forum_stats_recount step fixes counts gone stale.
func TestCommentsChangedRecountsTopic(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		d := testkit.Deps(db)
		seedUser(t, db)
		seedRow(t, db, `INSERT INTO taxonomy_vocabulary (id, name, slug, term_path_prefix) VALUES (1, 'Forum', 'forum', '/forum/')`)
		seedRow(t, db, `UPDATE forum_settings SET vocabulary_id = 1`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight) VALUES (1, 1, 'General', 'general', 0)`)
		seedRow(t, db, `INSERT INTO taxonomy_term_data (id, vid, name, slug, weight, parent_id) VALUES (2, 1, 'Talk', 'talk', 0, 1)`)
		testkit.WithModules(t, map[string]bool{manifest.Name: true})

		repo := newForumRepository(d)
		comments := testkit.Comments(d)
		comments.OnNodeCommentsChanged(manifest.Name, CommentsChanged(repo))
		topic, err := repo.CreateTopic(ctx, 2, "Hello", "first", 1)
		if err != nil {
			t.Fatal(err)
		}
		stats := func() (int, string) {
			t.Helper()
			var n int
			var at string
			if err := db.DB.QueryRowContext(ctx, db.Dialect.Rebind("SELECT comment_count, CAST(last_comment_at AS CHAR(19)) FROM forum_stats WHERE nid = ?"), topic.ID).Scan(&n, &at); err != nil {
				t.Fatal(err)
			}
			return n, at
		}
		_, created := stats()

		reply, err := comments.Create(ctx, &modulekit.Comment{Nid: topic.ID, CreatedBy: 1, Content: "a reply", Status: 1})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := comments.Create(ctx, &modulekit.Comment{Nid: topic.ID, CreatedBy: 1, Content: "waiting", Status: 0}); err != nil {
			t.Fatal(err)
		}
		if n, _ := stats(); n != 1 {
			t.Errorf("after a published and an unpublished reply: %d, want 1", n)
		}
		if err := comments.Delete(ctx, reply.ID); err != nil {
			t.Fatal(err)
		}
		if n, at := stats(); n != 0 || at != created {
			t.Errorf("after deleting the reply: %d at %s, want 0 at %s (the topic's own time)", n, at, created)
		}

		// Stale counts (replies made before the hook) - the step recounts.
		if _, err := comments.Create(ctx, &modulekit.Comment{Nid: topic.ID, CreatedBy: 1, Content: "again", Status: 1}); err != nil {
			t.Fatal(err)
		}
		seedRow(t, db, `UPDATE forum_stats SET comment_count = 42`)
		if _, err := db.DB.ExecContext(ctx, manifest.RecountTopicStatsSQL); err != nil { // the forum_stats_recount step
			t.Fatal(err)
		}
		if n, _ := stats(); n != 1 {
			t.Errorf("after the recount step: %d, want 1", n)
		}
	})
}
