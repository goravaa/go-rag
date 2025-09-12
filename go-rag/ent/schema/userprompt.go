package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "time"
)

type UserPrompt struct {
    ent.Schema
}

func (UserPrompt) Fields() []ent.Field {
    return []ent.Field{
        field.Text("query_text"),
        field.Time("created_at").Default(time.Now),
    }
}

func (UserPrompt) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("user", User.Type).
            Ref("queries").
            Unique(),
        edge.From("project", Project.Type).
            Ref("queries").
            Unique(),
        edge.To("results", QueryResult.Type),
    }
}