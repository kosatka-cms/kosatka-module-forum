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

// ForumSettings - Forum settings (one row): the vocabulary of forum sections.
type ForumSettings struct {
	ent.Schema
}

// Annotations of the ForumSettings.
func (ForumSettings) Annotations() []schema.Annotation {
	return []schema.Annotation{schemakit.Table("forum_settings")}
}

// Fields of the ForumSettings.
func (ForumSettings) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").SchemaType(schemakit.MySQLType("int unsigned")).Annotations(schemakit.NotIncremental()), // the single settings row
		field.Int("vocabulary_id").SchemaType(schemakit.MySQLType("int unsigned")).Optional().Nillable(),
	}
}

// Indexes of the ForumSettings.
func (ForumSettings) Indexes() []ent.Index {
	return []ent.Index{
		// Foreign key columns: PostgreSQL doesn't index them itself (MySQL
		// does), and deleting the row they refer to looks them up.
		index.Fields("vocabulary_id"),
	}
}

// Edges of the ForumSettings.
func (ForumSettings) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("vocabulary", TaxonomyVocabulary.Type).Field("vocabulary_id").Unique().Annotations(entsql.OnDelete(entsql.SetNull)),
	}
}
