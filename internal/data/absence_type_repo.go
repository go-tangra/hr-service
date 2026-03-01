package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/absencetype"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type AbsenceTypeRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewAbsenceTypeRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AbsenceTypeRepo {
	return &AbsenceTypeRepo{
		log:       ctx.NewLoggerHelper("hr/absence_type/repo"),
		entClient: entClient,
	}
}

func (r *AbsenceTypeRepo) Create(ctx context.Context, tenantID uint32, name string, opts ...func(*ent.AbsenceTypeCreate)) (*ent.AbsenceType, error) {
	id := uuid.New().String()

	create := r.entClient.Client().AbsenceType.Create().
		SetID(id).
		SetTenantID(tenantID).
		SetName(name).
		SetCreateTime(time.Now())

	for _, opt := range opts {
		opt(create)
	}

	entity, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, hrV1.ErrorAlreadyExists("absence type already exists with this name")
		}
		r.log.Errorf("create absence type failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("create absence type failed")
	}
	return entity, nil
}

func (r *AbsenceTypeRepo) GetByID(ctx context.Context, id string) (*ent.AbsenceType, error) {
	entity, err := r.entClient.Client().AbsenceType.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get absence type failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get absence type failed")
	}
	return entity, nil
}

func (r *AbsenceTypeRepo) List(ctx context.Context, tenantID uint32, page, pageSize int, filters map[string]interface{}) ([]*ent.AbsenceType, int, error) {
	query := r.entClient.Client().AbsenceType.Query().Where(absencetype.TenantID(tenantID))

	if q, ok := filters["query"].(string); ok && q != "" {
		query = query.Where(absencetype.NameContains(q))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count absence types failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list absence types failed")
	}

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	entities, err := query.Order(ent.Asc(absencetype.FieldSortOrder), ent.Asc(absencetype.FieldName)).All(ctx)
	if err != nil {
		r.log.Errorf("list absence types failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list absence types failed")
	}

	return entities, total, nil
}

func (r *AbsenceTypeRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*ent.AbsenceType, error) {
	update := r.entClient.Client().AbsenceType.UpdateOneID(id)

	if name, ok := updates["name"].(string); ok {
		update = update.SetName(name)
	}
	if description, ok := updates["description"].(string); ok {
		update = update.SetDescription(description)
	}
	if color, ok := updates["color"].(string); ok {
		update = update.SetColor(color)
	}
	if icon, ok := updates["icon"].(string); ok {
		update = update.SetIcon(icon)
	}
	if deducts, ok := updates["deducts_from_allowance"].(bool); ok {
		update = update.SetDeductsFromAllowance(deducts)
	}
	if requires, ok := updates["requires_approval"].(bool); ok {
		update = update.SetRequiresApproval(requires)
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		update = update.SetIsActive(isActive)
	}
	if sortOrder, ok := updates["sort_order"].(int); ok {
		update = update.SetSortOrder(sortOrder)
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		update = update.SetMetadata(metadata)
	}
	if requiresSigning, ok := updates["requires_signing"].(bool); ok {
		update = update.SetRequiresSigning(requiresSigning)
	}
	if signingTemplateID, ok := updates["signing_template_id"].(string); ok {
		update = update.SetSigningTemplateID(signingTemplateID)
	}

	update = update.SetUpdateTime(time.Now())

	entity, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, hrV1.ErrorAbsenceTypeNotFound("absence type not found")
		}
		r.log.Errorf("update absence type failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("update absence type failed")
	}
	return entity, nil
}

func (r *AbsenceTypeRepo) Delete(ctx context.Context, id string) error {
	// Check if absence type is in use
	count, err := r.entClient.Client().AbsenceType.Query().
		Where(absencetype.ID(id)).
		QueryLeaveRequests().
		Count(ctx)
	if err != nil {
		r.log.Errorf("check absence type usage failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete absence type failed")
	}
	if count > 0 {
		return hrV1.ErrorAbsenceTypeInUse("absence type has %d leave requests", count)
	}

	err = r.entClient.Client().AbsenceType.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return hrV1.ErrorAbsenceTypeNotFound("absence type not found")
		}
		r.log.Errorf("delete absence type failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete absence type failed")
	}
	return nil
}

func (r *AbsenceTypeRepo) Count(ctx context.Context, tenantID uint32) (int, error) {
	return r.entClient.Client().AbsenceType.Query().Where(absencetype.TenantID(tenantID)).Count(ctx)
}
