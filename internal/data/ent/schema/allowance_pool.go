package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// AllowancePool groups multiple absence types to share a single leave allowance budget.
type AllowancePool struct {
	ent.Schema
}

func (AllowancePool) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hr_allowance_pools"},
		entsql.WithComments(true),
	}
}

func (AllowancePool) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			NotEmpty().
			Unique().
			Comment("Unique identifier"),

		field.String("name").
			NotEmpty().
			MaxLen(255).
			Comment("Pool name (e.g. 'Sick Leave Pool')"),

		field.Text("description").
			Optional().
			Comment("Description"),

		field.String("color").
			Optional().
			MaxLen(7).
			Comment("Hex color for display"),

		field.String("icon").
			Optional().
			MaxLen(100).
			Comment("Lucide icon name"),
	}
}

func (AllowancePool) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("absence_types", AbsenceType.Type),
		edge.To("leave_allowances", LeaveAllowance.Type),
	}
}

func (AllowancePool) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.CreateBy{},
		mixin.UpdateBy{},
		mixin.Time{},
		mixin.TenantID[uint32]{},
	}
}

func (AllowancePool) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "name").Unique().StorageKey("idx_hr_pool_tenant_name"),
		index.Fields("tenant_id").StorageKey("idx_hr_pool_tenant"),
	}
}
