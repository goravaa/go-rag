package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
)

type QueryResult struct {
    ent.Schema
}

func (QueryResult) Fields() []ent.Field {
    return []ent.Field{
        field.Int("rank"),
        field.Float("score"),
        field.Text("content_snippet"),
    }
}

func (QueryResult) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("query", UserPrompt.Type).
            Ref("results").
            Unique(),
        edge.From("document", Document.Type).
            Ref("query_results").
            Unique(),
    }
}
