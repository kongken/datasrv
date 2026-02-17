package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Repository holds the schema definition for the Repository entity.
type Repository struct {
	ent.Schema
}

// Fields of the Repository.
func (Repository) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").Unique().Immutable().Comment("GitHub repository ID"),
		field.String("name").NotEmpty().Comment("Repository short name"),
		field.String("full_name").Unique().NotEmpty().Comment("Repository full name (owner/name)"),
		field.String("owner_login").NotEmpty().Comment("Repository owner login"),
		field.String("description").Optional().Comment("Repository description"),
		field.Bool("private").Default(false).Comment("Whether repository is private"),
		field.Bool("archived").Default(false).Comment("Whether repository is archived"),
		field.Bool("disabled").Default(false).Comment("Whether repository is disabled"),
		field.String("html_url").Optional().Comment("Repository HTML URL"),
		field.String("default_branch").Default("main").Comment("Default branch name"),
		field.String("language").Optional().Comment("Primary language"),
		field.Int32("stargazers_count").Default(0).Comment("Stargazers count"),
		field.Int32("forks_count").Default(0).Comment("Forks count"),
		field.Int32("open_issues_count").Default(0).Comment("Open issues count"),
		field.Time("created_at").Default(time.Now).Immutable().Comment("Creation time"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).Comment("Last update time"),
		field.Time("pushed_at").Optional().Nillable().Comment("Last push time"),
	}
}

// Edges of the Repository.
func (Repository) Edges() []ent.Edge {
	return nil
}
