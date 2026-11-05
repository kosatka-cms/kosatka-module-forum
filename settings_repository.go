package forum

import (
	"context"
	"database/sql"

	"github.com/kosatka-cms/kosatka-module-forum/ent/generated"
	"github.com/kosatka-cms/kosatka/modulekit"
)

// forumVocabularySQL - the forum vocabulary's id (ForumSettings) as a
// subquery for the forum's own queries (forum.go).
const forumVocabularySQL = "(SELECT vocabulary_id FROM forum_settings WHERE id = 1)"

// EntForumSettingsRepository - forum_settings, on the module's own ent
// schema (ent/).
type EntForumSettingsRepository struct {
	db *sql.DB
}

// NewForumSettingsRepository - the forum settings' repository on db.
func NewForumSettingsRepository(db *sql.DB) *EntForumSettingsRepository {
	return &EntForumSettingsRepository{db: db}
}

// client - the module's client for ctx: in its transaction, if any.
func (r *EntForumSettingsRepository) client(ctx context.Context) *generated.Client {
	return generated.NewClient(generated.Driver(modulekit.Driver(ctx, r.db)))
}

// Get - the forum's settings, from its single row.
func (r *EntForumSettingsRepository) Get(ctx context.Context) (*ForumSettings, error) {
	s, err := r.client(ctx).ForumSettings.Get(ctx, settingsRowID)
	if err != nil {
		return nil, err
	}
	out := &ForumSettings{}
	if s.VocabularyID != nil {
		out.VocabularyID = *s.VocabularyID
	}
	return out, nil
}

// Update saves s to the single settings row (VocabularyID 0 - none).
func (r *EntForumSettingsRepository) Update(ctx context.Context, s *ForumSettings) error {
	u := r.client(ctx).ForumSettings.UpdateOneID(settingsRowID)
	if s.VocabularyID > 0 {
		u.SetVocabularyID(s.VocabularyID)
	} else {
		u.ClearVocabularyID()
	}
	return u.Exec(ctx)
}
