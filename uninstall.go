package forum

import (
	"context"
	"log/slog"
)

// ForumUninstallStep - the forum module's own uninstall step: its
// vocabulary of sections (forum settings) is deleted with its terms
// unless something else uses it (a taxonomy field, a search badge, a
// field value). Runs after the forum's content is gone, before its
// settings table is dropped.
func ForumUninstallStep(settings ForumSettingsRepository, vocabularies interface {
	DeleteVocabularyIfUnused(ctx context.Context, vid int) (bool, error)
}, logger *slog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		s, err := settings.Get(ctx)
		if err != nil || s == nil || s.VocabularyID == 0 {
			return err
		}
		deleted, err := vocabularies.DeleteVocabularyIfUnused(ctx, s.VocabularyID)
		if err != nil {
			return err
		}
		logger.Info("forum uninstall: sections vocabulary", "vocabularyID", s.VocabularyID, "deleted", deleted)
		return nil
	}
}
