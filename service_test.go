package forum

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit"
)

// fakeForumSectionRepo - records the slugs GetSectionBySlug is asked for.
type fakeForumSectionRepo struct {
	ForumRepository
	prefix string
	asked  []string
}

func (f *fakeForumSectionRepo) SectionPathPrefix(context.Context) (string, error) {
	return f.prefix, nil
}

func (f *fakeForumSectionRepo) GetSectionBySlug(_ context.Context, slug string) (*ForumSection, error) {
	f.asked = append(f.asked, slug)
	return nil, modulekit.ErrEntityNotFound
}

// TestGetSectionBySlugExact - a section is looked up by the slug as given,
// no variants of it; the URL prefix of the sections is the forum
// vocabulary's term path prefix.
func TestGetSectionBySlugExact(t *testing.T) {
	repo := &fakeForumSectionRepo{prefix: "forums"}
	s := NewForumService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, _, _ = s.GetSectionBySlug(context.Background(), "games/pc")
	if len(repo.asked) != 1 || repo.asked[0] != "games/pc" {
		t.Fatalf("asked %v, want [games/pc]", repo.asked)
	}
	if s.SectionPathPrefix(context.Background()) != "forums" {
		t.Fatal("SectionPathPrefix")
	}
}
