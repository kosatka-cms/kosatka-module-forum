package manifest

import (
	"context"
	"fmt"

	"github.com/kosatka-cms/kosatka-module-forum/ent"
	"github.com/kosatka-cms/kosatka-module-forum/ent/generated"
	"github.com/kosatka-cms/kosatka/modulekit/demokit"
	"github.com/kosatka-cms/kosatka/modulekit/manifestkit"
)

// RegisterSteps registers the module's install and data steps with the
// migrations and its demo step (manifests.Register).
func RegisterSteps() {
	demokit.Register(Name, createDemo)
	manifestkit.RegisterSchema(Name, ent.Schema)
	// Installed: its settings row (the forum reads and updates it; without
	// it the settings page fails and saving does nothing), and signed-in
	// users may start topics (grantCreateTopics).
	manifestkit.RegisterInstall(Name, func(ctx context.Context, env manifestkit.InstallEnv) error {
		c := generated.NewClient(generated.Driver(manifestkit.Driver(env)))
		if err := c.ForumSettings.Create().SetID(settingsRowID).Exec(ctx); err != nil {
			return fmt.Errorf("the forum's settings row: %w", err)
		}
		return grantCreateTopics(ctx, env)
	})
}

// settingsRowID - the id of forum_settings' single row.
const settingsRowID = 1

// RecountTopicStatsSQL sets every topic's comment count and last comment
// time from its published comments (the topic's own time while it has
// none); append " WHERE nid = ?" for one topic.
const RecountTopicStatsSQL = `UPDATE forum_stats SET
	comment_count = (SELECT COUNT(*) FROM comments c WHERE c.nid = forum_stats.nid AND c.status = 1),
	last_comment_at = COALESCE(
		(SELECT MAX(c.created_at) FROM comments c WHERE c.nid = forum_stats.nid AND c.status = 1),
		(SELECT n.created_at FROM nodes n WHERE n.id = forum_stats.nid),
		last_comment_at)`

// grantCreateTopics gives the role of every signed-in user (manifestkit.RoleUser)
// the right to start forum topics, keeping its other grants.
func grantCreateTopics(ctx context.Context, env manifestkit.InstallEnv) error {
	return manifestkit.GrantPermission(ctx, env, manifestkit.RoleUser, "create "+string(ContentType)+" content")
}
