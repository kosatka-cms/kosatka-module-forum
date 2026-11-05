package forum

import "context"

// ForumSettings - the forum module's settings (single DB row). The
// vocabulary's root terms are the forum's categories, their children its
// sections; 0 - none chosen, the forum is empty.
type ForumSettings struct {
	VocabularyID int
}

// ForumSettingsRepository persists the single forum settings row, same
// shape as UploadSettingsRepository.
type ForumSettingsRepository interface {
	Get(ctx context.Context) (*ForumSettings, error)
	Update(ctx context.Context, s *ForumSettings) error
}
