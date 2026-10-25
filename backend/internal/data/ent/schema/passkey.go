// SPDX-License-Identifier: MIT; Copyright (c) 2026 Phoobobo
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

type Passkey struct{ ent.Schema }

func (Passkey) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.BaseMixin{}, UserMixin{ref: "passkeys", field: "user_id"}}
}
func (Passkey) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().MaxLen(80),
		field.String("credential_id").Unique().Sensitive(),
		field.Bytes("credential").Sensitive(),
		field.Int("version").Default(0),
		field.Time("last_used_at").Optional().Nillable(),
	}
}
func (Passkey) Indexes() []ent.Index { return []ent.Index{index.Fields("user_id")} }
