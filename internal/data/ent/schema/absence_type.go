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

// AbsenceType represents a configurable type of absence (vacation, sick, etc.)
type AbsenceType struct {
	ent.Schema
}

func (AbsenceType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hr_absence_types"},
		entsql.WithComments(true),
	}
}

func (AbsenceType) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			NotEmpty().
			Unique().
			Comment("Unique identifier"),

		field.String("name").
			NotEmpty().
			MaxLen(255).
			Comment("Absence type name"),

		field.Text("description").
			Optional().
			Comment("Description"),

		field.String("color").
			Optional().
			MaxLen(7).
			Comment("Hex color for calendar display"),

		field.String("icon").
			Optional().
			MaxLen(100).
			Comment("Lucide icon name"),

		field.Bool("deducts_from_allowance").
			Default(true).
			Comment("Whether this type consumes leave days"),

		field.Bool("requires_approval").
			Default(true).
			Comment("Whether requests need HR admin approval"),

		field.Bool("is_active").
			Default(true).
			Comment("Whether this type is available for new requests"),

		field.Int("sort_order").
			Default(0).
			Comment("Display sort order"),

		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Custom metadata (JSON)"),

		field.Bool("requires_signing").
			Default(false).
			Comment("Whether this type requires document signing"),

		field.String("signing_template_id").
			Optional().
			Comment("Paperless signing template ID"),
	}
}

func (AbsenceType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("leave_allowances", LeaveAllowance.Type),
		edge.To("leave_requests", LeaveRequest.Type),
	}
}

func (AbsenceType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.CreateBy{},
		mixin.UpdateBy{},
		mixin.Time{},
		mixin.TenantID[uint32]{},
	}
}

func (AbsenceType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "name").Unique().StorageKey("idx_hr_abstype_tenant_name"),
		index.Fields("tenant_id").StorageKey("idx_hr_abstype_tenant"),
	}
}
