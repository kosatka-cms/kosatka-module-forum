package forum

import (
	"context"
	"fmt"
)

// CommentsChanged - the forum's CommentsService.OnNodeCommentsChanged
// hook: a topic's reply count and last reply time follow its comments
// (the topic lists sort and filter by them). Nothing while the module is
// off (its tables may not exist).
func CommentsChanged(repo *EntForumRepository) func(ctx context.Context, nodeID int) error {
	return func(ctx context.Context, nodeID int) error {
		if err := repo.RecountTopicStats(ctx, nodeID); err != nil {
			return fmt.Errorf("recounting forum topic stats (nid %v): %w", nodeID, err)
		}
		return nil
	}
}
