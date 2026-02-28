package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LeaveRequest represents a user's leave/absence request
type LeaveRequest struct {
	ent.Schema
}

func (LeaveRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "hr_leave_requests"},
		entsql.WithComments(true),
	}
}

func (LeaveRequest) Fields() []ent.Field {
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

		field.String("org_unit_name").
			Optional().
			Default("").
			Comment("Denormalized org unit name for grouping"),

		field.String("absence_type_id").
			NotEmpty().
			Comment("FK to AbsenceType"),

		field.Time("start_date").
			Comment("Start date of absence"),

		field.Time("end_date").
			Comment("End date of absence"),

		field.Float("days").
			Comment("Calculated business days"),

		field.Enum("status").
			Values("pending", "approved", "rejected", "cancelled").
			Default("pending").
			Comment("Request status"),

		field.Text("reason").
			Optional().
			Comment("User's reason for request"),

		field.Text("review_notes").
			Optional().
			Comment("HR admin's review notes"),

		field.Uint32("reviewed_by").
			Optional().
			Default(0).
			Comment("User ID of reviewer"),

		field.String("reviewer_name").
			Optional().
			Default("").
			Comment("Denormalized reviewer display name"),

		field.Time("reviewed_at").
			Optional().
			Nillable().
			Comment("When the request was reviewed"),

		field.Text("notes").
			Optional().
			Comment("Additional notes"),

		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Custom metadata (JSON)"),
	}
}

func (LeaveRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("absence_type", AbsenceType.Type).
			Ref("leave_requests").
			Field("absence_type_id").
			Unique().
			Required(),
	}
}

func (LeaveRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.CreateBy{},
		mixin.UpdateBy{},
		mixin.Time{},
		mixin.TenantID[uint32]{},
	}
}

func (LeaveRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "user_id", "start_date").StorageKey("idx_hr_leavereq_tenant_user_start"),
		index.Fields("tenant_id", "status").StorageKey("idx_hr_leavereq_tenant_status"),
		index.Fields("tenant_id").StorageKey("idx_hr_leavereq_tenant"),
	}
}

// Ensure time import is used
var _ = time.Now
