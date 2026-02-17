package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Unique().Immutable().Comment("GitHub user ID"),
		field.String("login").NotEmpty().Comment("GitHub username"),
		field.String("avatar_url").Optional().Comment("Avatar URL"),
		field.String("html_url").Optional().Comment("User profile URL"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
