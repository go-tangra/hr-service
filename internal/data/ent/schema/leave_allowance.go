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

// LeaveAllowance represents a user's leave allowance for a specific type and year
type LeaveAllowance struct {
	ent.Schema
}

func (LeaveAllowance) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hr_leave_allowances"},
		entsql.WithComments(true),
	}
}

func (LeaveAllowance) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			NotEmpty().
			Unique().
			Comment("Unique identifier"),

		field.Uint32("user_id").
			Comment("FK to Portal User"),

		field.String("user_name").
			Optional().
			Default("").
			Comment("Denormalized user display name"),

		field.String("absence_type_id").
			NotEmpty().
			Comment("FK to AbsenceType"),

		field.Int("year").
			Positive().
			Comment("Calendar year"),

		field.Float("total_days").
			Comment("Total allowed days (supports half-days)"),

		field.Float("used_days").
			Default(0).
			Comment("Consumed days"),

		field.Float("carried_over").
			Default(0).
			Comment("Carried from previous year"),

		field.Text("notes").
			Optional().
			Comment("Notes"),
	}
}

func (LeaveAllowance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("absence_type", AbsenceType.Type).
			Ref("leave_allowances").
			Field("absence_type_id").
			Unique().
			Required(),
	}
}

func (LeaveAllowance) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.CreateBy{},
		mixin.UpdateBy{},
		mixin.Time{},
		mixin.TenantID[uint32]{},
	}
}

func (LeaveAllowance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "user_id", "absence_type_id", "year").Unique().StorageKey("idx_hr_allowance_tenant_user_type_year"),
		index.Fields("tenant_id").StorageKey("idx_hr_allowance_tenant"),
	}
}
