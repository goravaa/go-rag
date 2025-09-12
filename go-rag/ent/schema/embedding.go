package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
)

type Embedding struct {
    ent.Schema
}

func (Embedding) Fields() []ent.Field {
    return []ent.Field{
        field.JSON("vector", []float32{}),
    }
}

func (Embedding) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("chunk", Chunk.Type).
            Ref("embeddings").
            Unique(),
    }
}
