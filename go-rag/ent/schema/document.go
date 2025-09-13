package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "time"
)

type Document struct {
    ent.Schema
}

func (Document) Fields() []ent.Field {
    return []ent.Field{
        field.String("name"),
        field.String("storage_path"), // MinIO key or URL
        field.String("status").Default("uploaded"),
        field.Time("created_at").Default(time.Now),
    }
}

func (Document) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("project", Project.Type).
            Ref("documents").
            Unique(),
        edge.To("chunks", Chunk.Type),
        edge.To("query_results", QueryResult.Type),
    }
}
