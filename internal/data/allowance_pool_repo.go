package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/allowancepool"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type AllowancePoolRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewAllowancePoolRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AllowancePoolRepo {
	return &AllowancePoolRepo{
		log:       ctx.NewLoggerHelper("hr/allowance_pool/repo"),
		entClient: entClient,
	}
}

func (r *AllowancePoolRepo) Create(ctx context.Context, tenantID uint32, name string, opts ...func(*ent.AllowancePoolCreate)) (*ent.AllowancePool, error) {
	id := uuid.New().String()

	create := r.entClient.Client().AllowancePool.Create().
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
			return nil, hrV1.ErrorAlreadyExists("allowance pool already exists with this name")
		}
		r.log.Errorf("create allowance pool failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("create allowance pool failed")
	}
	return entity, nil
}

func (r *AllowancePoolRepo) GetByID(ctx context.Context, id string) (*ent.AllowancePool, error) {
	entity, err := r.entClient.Client().AllowancePool.Query().
		Where(allowancepool.ID(id)).
		WithAbsenceTypes().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get allowance pool failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get allowance pool failed")
	}
	return entity, nil
}

func (r *AllowancePoolRepo) List(ctx context.Context, tenantID uint32, page, pageSize int, filters map[string]interface{}) ([]*ent.AllowancePool, int, error) {
	query := r.entClient.Client().AllowancePool.Query().
		Where(allowancepool.TenantID(tenantID)).
		WithAbsenceTypes()

	if q, ok := filters["query"].(string); ok && q != "" {
		query = query.Where(allowancepool.NameContains(q))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count allowance pools failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list allowance pools failed")
	}

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	entities, err := query.Order(ent.Asc(allowancepool.FieldName)).All(ctx)
	if err != nil {
		r.log.Errorf("list allowance pools failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list allowance pools failed")
	}

	return entities, total, nil
}

func (r *AllowancePoolRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*ent.AllowancePool, error) {
	update := r.entClient.Client().AllowancePool.UpdateOneID(id)

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

	update = update.SetUpdateTime(time.Now())

	entity, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
		}
		r.log.Errorf("update allowance pool failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("update allowance pool failed")
	}
	return entity, nil
}

func (r *AllowancePoolRepo) Delete(ctx context.Context, id string) error {
	// Check if pool has allowances
	allowCount, err := r.entClient.Client().AllowancePool.Query().
		Where(allowancepool.ID(id)).
		QueryLeaveAllowances().
		Count(ctx)
	if err != nil {
		r.log.Errorf("check allowance pool usage failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete allowance pool failed")
	}
	if allowCount > 0 {
		return hrV1.ErrorAllowancePoolInUse("allowance pool has %d leave allowances", allowCount)
	}

	// Clear allowance_pool_id on linked absence types so they can be reassigned
	linkedTypes, err := r.entClient.Client().AllowancePool.Query().
		Where(allowancepool.ID(id)).
		QueryAbsenceTypes().
		All(ctx)
	if err != nil {
		r.log.Errorf("query linked absence types failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete allowance pool failed")
	}
	for _, at := range linkedTypes {
		if _, err := r.entClient.Client().AbsenceType.UpdateOneID(at.ID).
			ClearAllowancePoolID().
			Save(ctx); err != nil {
			r.log.Errorf("clear allowance_pool_id on absence type %s failed: %s", at.ID, err.Error())
			return hrV1.ErrorInternalServerError("delete allowance pool failed")
		}
	}

	err = r.entClient.Client().AllowancePool.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
		}
		r.log.Errorf("delete allowance pool failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete allowance pool failed")
	}
	return nil
}
