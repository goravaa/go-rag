package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("description").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).
			Ref("projects").
			Unique(),

		edge.To("documents", Document.Type).
			Annotations(
				entsql.OnDelete(entsql.Cascade),
			),

		edge.To("queries", UserPrompt.Type).
			Annotations(
				entsql.OnDelete(entsql.Cascade),
			),
	}
}
