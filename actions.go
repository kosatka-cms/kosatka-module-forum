package forum

import (
	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/pkg/i18n"
)

// The forum's actions in the action log: their group and types. A topic
// created or deleted is the core's "Content" group
// (modulekit.RecordNodeCreated, the core's deletion).
const (
	ActionGroup      = "forum"
	ActionTopicMove  = "forum.topic_move"
	ActionTopicPin   = "forum.topic_pin"
	ActionTopicUnpin = "forum.topic_unpin"
)

// registerActions adds the forum's group and actions to the action log.
func registerActions() {
	modulekit.AddActionGroup(modulekit.ActionGroup{Name: ActionGroup, Module: manifest.Name, Label: i18n.N("Forum"),
		Description: i18n.N("Topics moved to another section, pinned and unpinned. Topics created or deleted are in the Content group.")})
	for _, t := range []modulekit.ActionType{
		{Name: ActionTopicMove, Group: ActionGroup, Label: i18n.N("Topic moved"), Tone: modulekit.ActionToneChange},
		{Name: ActionTopicPin, Group: ActionGroup, Label: i18n.N("Topic pinned"), Tone: modulekit.ActionToneChange},
		{Name: ActionTopicUnpin, Group: ActionGroup, Label: i18n.N("Topic unpinned"), Tone: modulekit.ActionToneChange},
	} {
		modulekit.AddActionType(t)
	}
}

// topicNode - the topic as content, for its object in the action log.
func topicNode(id int, title string) modulekit.Node {
	return modulekit.Node{ID: id, Title: title, Type: manifest.ContentType}
}
