// Package schema - the forum module's own ent schema: its tables, apart
// from the core's. A topic is a core node and the forum's sections a
// core vocabulary: Node and TaxonomyVocabulary here are the module's
// stand-ins for their tables (migrations.Sets).
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/kosatka-cms/kosatka/modulekit/schemakit"
)

// ForumTopic - a forum topic's text: the body of the topic node (nid).
type ForumTopic struct {
	ent.Schema
}

// Annotations of the ForumTopic.
func (ForumTopic) Annotations() []schema.Annotation {
	return []schema.Annotation{
		schemakit.Table("forum_topics"),
	}
}

// Fields of the ForumTopic.
func (ForumTopic) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").SchemaType(schemakit.MySQLType("int")),
		field.Int("nid").SchemaType(schemakit.MySQLType("int unsigned")),
		field.Text("body").SchemaType(schemakit.MySQLType("mediumtext")).Optional(),
	}
}

// Edges of the ForumTopic.
func (ForumTopic) Edges() []ent.Edge {
	return []ent.Edge{
		// The topic goes with its node.
		edge.To("nodes", Node.Type).Field("nid").Required().Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the ForumTopic.
func (ForumTopic) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("nid").Unique().StorageKey("forum_topics_nid"),
	}
}
