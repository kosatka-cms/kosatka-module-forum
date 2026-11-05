package forum

import (
	"context"
	"strings"

	"github.com/kosatka-cms/kosatka/modulekit"
)

// SitemapPaths - the forum sections for the sitemap (AddSitemapPaths), at
// their terms' addresses, as the forum links to them.
func SitemapPaths(svc ForumService) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		idx, err := svc.GetForumIndex(ctx)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, cat := range idx.Categories {
			for _, sec := range cat.Sections {
				if slug := strings.TrimSpace(sec.Slug); slug != "" {
					out = append(out, "/"+modulekit.EscapePathSegments(slug))
				}
			}
		}
		return out, nil
	}
}
