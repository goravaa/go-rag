package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type SecurityQuestion struct {
	ent.Schema
}

func (SecurityQuestion) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),

		field.String("question").
			NotEmpty(),

		field.String("answer").
			NotEmpty(),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (SecurityQuestion) Edges() []ent.Edge {
	return []ent.Edge{

		edge.From("user", User.Type).
			Ref("security_questions").
			Unique().
			Required(),
	}
}
