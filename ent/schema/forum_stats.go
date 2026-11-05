package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/kosatka-cms/kosatka/modulekit/schemakit"
)

// ForumStats holds the schema definition for the ForumStats entity.
type ForumStats struct {
	ent.Schema
}

// Annotations of the ForumStats.
func (ForumStats) Annotations() []schema.Annotation {
	return []schema.Annotation{
		schemakit.Table("forum_stats"),
	}
}

// Fields of the ForumStats.
func (ForumStats) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("nid"),
		field.Int("sticky").SchemaType(schemakit.MySQLType("int")).Default(0).Comment("Whether the forum topic is pinned"),
		field.Int("comment_count").SchemaType(schemakit.MySQLType("int")).Default(0).Comment("The topic's comment count, cached"),
		field.Time("last_comment_at").SchemaType(schemakit.MySQLType("datetime")).Comment("When the last comment was posted"),
	}
}

// Edges of the ForumStats.
func (ForumStats) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("node", Node.Type).
			Ref("forum_stats").
			Field("nid").
			Required().
			Unique(),
	}
}

// Indexes of the ForumStats.
func (ForumStats) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sticky"),
		index.Fields("last_comment_at"),
		// Composite index for sorting: pinned first, then by activity.
	}
}
