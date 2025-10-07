package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("session_id", uuid.New()).Unique(),
		field.UUID("sessions_userids", uuid.UUID{}),
		field.Enum("session_type").Values("auth", "sync"),
		field.String("access_token").Unique(),
		field.String("refresh_token").Optional().Nillable(),
		field.String("device_name").Optional().Nillable(),
		field.Time("last_sync_time").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("expires_at"),
		field.Time("revoked_at").Optional().Nillable(),
		field.String("ip_address").Optional().Nillable(),
		field.String("user_agent").Optional().Nillable(),
		field.JSON("metadata", map[string]any{}).Optional(),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("sessions").
			Unique().
			Required().
			Field("sessions_userids"),
	}
}
