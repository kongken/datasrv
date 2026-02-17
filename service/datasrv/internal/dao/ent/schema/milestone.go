package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Milestone holds the schema definition for the Milestone entity.
type Milestone struct {
	ent.Schema
}

// Fields of the Milestone.
func (Milestone) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Unique().Immutable().Comment("GitHub milestone ID"),
		field.Int32("number").Comment("Milestone number"),
		field.String("title").NotEmpty().Comment("Milestone title"),
		field.String("description").Optional().Comment("Milestone description"),
		field.String("state").Default("open").Comment("Milestone state (open/closed)"),
		field.Time("due_on").Optional().Nillable().Comment("Due date"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("Last update time"),
	}
}

// Edges of the Milestone.
func (Milestone) Edges() []ent.Edge {
	return nil
}
