package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/kosatka-cms/kosatka/modulekit/schemakit"
)

// The core's tables the module's reference, as its stand-ins: their id
// only (of the core's column type), for the module's edges - the foreign
// keys. A table named as a core one isn't the module's: its set leaves it
// out (migrations.Sets) and the module reads them through the core,
// never through these.

// Node - the core's nodes: a topic's (ON DELETE CASCADE).
type Node struct {
	ent.Schema
}

// Annotations of the Node.
func (Node) Annotations() []schema.Annotation {
	return []schema.Annotation{schemakit.Table("nodes")}
}

// Fields of the Node.
func (Node) Fields() []ent.Field {
	return []ent.Field{field.Int("id").SchemaType(schemakit.MySQLType("int unsigned"))}
}

// Edges of the Node.
func (Node) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("forum_stats", ForumStats.Type).Unique().Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// TaxonomyVocabulary - the core's vocabularies: the forum's sections'
// (ON DELETE SET NULL).
type TaxonomyVocabulary struct {
	ent.Schema
}

// Annotations of the TaxonomyVocabulary.
func (TaxonomyVocabulary) Annotations() []schema.Annotation {
	return []schema.Annotation{schemakit.Table("taxonomy_vocabulary")}
}

// Fields of the TaxonomyVocabulary.
func (TaxonomyVocabulary) Fields() []ent.Field {
	return []ent.Field{field.Int("id").SchemaType(schemakit.MySQLType("int unsigned"))}
}
