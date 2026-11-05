package manifest

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/kosatka-cms/kosatka-module-forum/ent/generated"
	"github.com/kosatka-cms/kosatka/modulekit/demokit"
	"github.com/kosatka-cms/kosatka/pkg/slug"
)

// demoFS - the module's demo content by language: demo/<lang>.json.
//
//go:embed demo/*.json
var demoFS embed.FS

// demoContent - the forum vocabulary's name, its categories with their
// sections, the topics (in a section, by a demo user, with replies; the
// first ones newest) and the title of the main menu's link to /forum.
type demoContent struct {
	Vocabulary string         `json:"vocabulary"`
	Categories []demoCategory `json:"categories"`
	Topics     []demoTopic    `json:"topics"`
	Menu       string         `json:"menu"`
}

type demoCategory struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Sections    []demoSection `json:"sections"`
}

type demoSection struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type demoTopic struct {
	Section string            `json:"section"`
	Title   string            `json:"title"`
	Author  string            `json:"author"`
	Body    string            `json:"body"` // HTML
	Sticky  bool              `json:"sticky"`
	Views   int               `json:"views"`
	Replies []demokit.Comment `json:"replies"`
}

// demoTopicStep - the time between two demo topics.
const demoTopicStep = 26 * time.Hour

// createDemo adds the forum vocabulary (its terms' URLs under /forum, as
// the forum serves them) and makes it the forum's, its categories and
// sections, the topics with their replies and counts, and a link to the
// forum in the main menu.
func createDemo(ctx context.Context, env demokit.Env) error {
	var content demoContent
	if err := demokit.Content(env, demoFS, &content); err != nil {
		return err
	}
	// The module's own client, in the install transaction.
	c := generated.NewClient(generated.Driver(demokit.Driver(env)))
	vid, err := demokit.Vocabulary(ctx, env, content.Vocabulary, Name, Name)
	if err != nil {
		return err
	}
	if err := c.ForumSettings.UpdateOneID(settingsRowID).SetVocabularyID(vid).Exec(ctx); err != nil {
		return err
	}
	term := func(name, description, path string, parent *int, weight int) (int, error) {
		return demokit.Term(ctx, env, vid, name, description, path, parent, weight)
	}
	sections := map[string]int{}
	for i, cat := range content.Categories {
		catPath := Name + "/" + slug.Generate(cat.Name)
		catID, err := term(cat.Name, cat.Description, catPath, nil, i)
		if err != nil {
			return err
		}
		for j, sec := range cat.Sections {
			if sections[sec.Name], err = term(sec.Name, sec.Description, catPath+"/"+slug.Generate(sec.Name), &catID, j); err != nil {
				return err
			}
		}
	}
	for i, t := range content.Topics {
		section, ok := sections[t.Section]
		if !ok {
			return fmt.Errorf("topic %q: unknown section %q", t.Title, t.Section)
		}
		author, err := env.User(t.Author)
		if err != nil {
			return err
		}
		at := env.Now.Add(-time.Duration(i+1) * demoTopicStep)
		nodeID, err := demokit.CreateNode(ctx, env, demokit.Node{Type: string(ContentType), Title: t.Title, Author: author, At: at, Views: t.Views})
		if err != nil {
			return err
		}
		if err := c.ForumTopic.Create().SetNid(nodeID).SetBody(t.Body).Exec(ctx); err != nil {
			return err
		}
		if err := demokit.IndexTerm(ctx, env, nodeID, section, at); err != nil {
			return err
		}
		count, last, err := demokit.CreateComments(ctx, env, nodeID, t.Replies, at)
		if err != nil {
			return fmt.Errorf("replies to %q: %w", t.Title, err)
		}
		sticky := 0
		if t.Sticky {
			sticky = 1
		}
		// The counts the forum keeps in step with the comments (comments.go).
		if err := c.ForumStats.Create().SetNid(nodeID).SetSticky(sticky).SetCommentCount(count).SetLastCommentAt(last).Exec(ctx); err != nil {
			return err
		}
	}
	return demokit.MenuLink(ctx, env, env.MainMenu, content.Menu, "/"+Name, 0, 5)
}
