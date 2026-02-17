package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Label holds the schema definition for the Label entity.
type Label struct {
	ent.Schema
}

// Fields of the Label.
func (Label) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Unique().Immutable().Comment("GitHub label ID"),
		field.String("name").NotEmpty().Comment("Label name"),
		field.String("color").Optional().Comment("Label color hex code"),
		field.String("description").Optional().Comment("Label description"),
	}
}

// Edges of the Label.
func (Label) Edges() []ent.Edge {
	return nil
}
