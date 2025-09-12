package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
)

type Chunk struct {
    ent.Schema
}

func (Chunk) Fields() []ent.Field {
    return []ent.Field{
        field.Int("index"),
        field.Text("content"),
    }
}

func (Chunk) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("document", Document.Type).
            Ref("chunks").
            Unique(),
        edge.To("embeddings", Embedding.Type),
    }
}
