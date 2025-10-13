package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Chunk struct {
	ent.Schema
}

func (Chunk) Fields() []ent.Field {
	return []ent.Field{
		field.Int("index"),
		field.Text("content"),
		field.String("content_hash").Optional(),
	}
}

func (Chunk) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("content_hash"),
	}
}

func (Chunk) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("document", Document.Type).
			Ref("chunks").
			Unique(),
		edge.To("query_results", QueryResult.Type),
	}
}
