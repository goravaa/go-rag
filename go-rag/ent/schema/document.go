package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Document struct {
	ent.Schema
}

func (Document) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.Text("content"),
		field.String("content_hash").Optional(), // .Index() is removed
		field.String("status").Default("uploaded"),
		field.Time("created_at").Default(time.Now),
	}
}

// This is the correct way to define an index
func (Document) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("content_hash"),
	}
}

func (Document) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("documents").
			Unique(),

		edge.To("chunks", Chunk.Type).
			Annotations(
				entsql.OnDelete(entsql.Cascade),
			),
	}
}
