package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Issue holds the schema definition for the Issue entity.
type Issue struct {
	ent.Schema
}

// Fields of the Issue.
func (Issue) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Unique().Immutable().Comment("GitHub issue ID"),
		field.Int32("number").Comment("Issue number"),
		field.String("title").NotEmpty().Comment("Issue title"),
		field.Text("body").Optional().Comment("Issue body/description"),
		field.String("state").Default("open").Comment("Issue state (open/closed)"),
		field.Int32("comments").Default(0).Comment("Number of comments"),
		field.String("html_url").Optional().Comment("Issue HTML URL"),
		field.Bool("locked").Default(false).Comment("Whether the issue is locked"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("Last update time"),
		field.Time("closed_at").Optional().Nillable().Comment("Close time"),
		// Foreign keys
		field.Int64("user_id").Optional().Comment("Creator user ID"),
		field.Int64("milestone_id").Optional().Nillable().Comment("Milestone ID"),
	}
}

// Edges of the Issue.
func (Issue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).Unique().Field("user_id"),
		edge.To("labels", Label.Type),
		edge.To("assignees", User.Type),
		edge.To("milestone", Milestone.Type).Unique().Field("milestone_id"),
	}
}
