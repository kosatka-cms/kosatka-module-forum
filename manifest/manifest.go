// Package manifest declares the forum module (its registry entry).
package manifest

import (
	"github.com/kosatka-cms/kosatka/modulekit/manifestkit"
	"github.com/kosatka-cms/kosatka/pkg/i18n"
)

// Name - the module's machine name.
const Name = "forum"

// ContentType - the forum topic node type; its bare slug lives under
// "forum/" (PathPrefix).
const ContentType manifestkit.ContentType = "forum"

// FileSourceTopic - a topic's text in the files manager's "used in".
const FileSourceTopic = "forum_topic"

// Module - the module's registry entry.
func Module() manifestkit.Module {
	return manifestkit.Module{
		Name: Name, Label: i18n.N("Forum"), Description: i18n.N("Forum sections and topics: /forum pages, forum settings, the forum topic content type."),
		Category: manifestkit.CategoryCommunity,
		NodeTypes: []manifestkit.ModuleNodeType{{Name: string(ContentType), Label: i18n.N("Forum"), PathPrefix: "forum/", OwnDiscussion: true,
			EditPath: "/forum/topic/{id}/edit", DeletePath: "/forum/topic/{id}/delete"}},
		CSS:            []string{"forum"},
		CoreData:       []string{i18n.N("The vocabulary of forum sections, unless something else uses it")},
		RobotsDisallow: []string{"/forum/new-topic/", "/forum/topic/*/edit", "/forum/topic/*/delete"},
		Paths:          []string{"/forum"},
		Tables:         []string{"forum_topics", "forum_stats", "forum_settings"},
		AdminMenu: []manifestkit.ModuleMenuItem{{Section: manifestkit.AdminSectionConfiguration,
			Item: manifestkit.AdminMenuItem{Label: i18n.N("Forum"), Path: "/admin/config/forum", Group: manifestkit.AdminGroupModules}}},
		FileSources: []manifestkit.ModuleFileSource{{Source: FileSourceTopic, Table: "forum_topics", IDColumn: "nid", Columns: []string{"body"},
			Label: i18n.N("Forum topic \"@title\""), NodeBacked: true}},
	}
}
