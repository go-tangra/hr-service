package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/leaveallowance"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type LeaveAllowanceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewLeaveAllowanceRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LeaveAllowanceRepo {
	return &LeaveAllowanceRepo{
		log:       ctx.NewLoggerHelper("hr/leave_allowance/repo"),
		entClient: entClient,
	}
}

func (r *LeaveAllowanceRepo) Create(ctx context.Context, tenantID uint32, userID uint32, absenceTypeID string, year int, totalDays float64, opts ...func(*ent.LeaveAllowanceCreate)) (*ent.LeaveAllowance, error) {
	id := uuid.New().String()

	create := r.entClient.Client().LeaveAllowance.Create().
		SetID(id).
		SetTenantID(tenantID).
		SetUserID(userID).
		SetAbsenceTypeID(absenceTypeID).
		SetYear(year).
		SetTotalDays(totalDays).
		SetCreateTime(time.Now())

	for _, opt := range opts {
		opt(create)
	}

	entity, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, hrV1.ErrorAlreadyExists("allowance already exists for this user, type, and year")
		}
		r.log.Errorf("create leave allowance failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("create leave allowance failed")
	}
	return entity, nil
}

func (r *LeaveAllowanceRepo) GetByID(ctx context.Context, id string) (*ent.LeaveAllowance, error) {
	entity, err := r.entClient.Client().LeaveAllowance.Query().
		Where(leaveallowance.ID(id)).
		WithAbsenceType().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get leave allowance failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get leave allowance failed")
	}
	return entity, nil
}

func (r *LeaveAllowanceRepo) List(ctx context.Context, tenantID uint32, page, pageSize int, filters map[string]interface{}) ([]*ent.LeaveAllowance, int, error) {
	query := r.entClient.Client().LeaveAllowance.Query().
		Where(leaveallowance.TenantID(tenantID)).
		WithAbsenceType()

	if userID, ok := filters["user_id"].(uint32); ok && userID > 0 {
		query = query.Where(leaveallowance.UserID(userID))
	}
	if year, ok := filters["year"].(int); ok && year > 0 {
		query = query.Where(leaveallowance.Year(year))
	}
	if absenceTypeID, ok := filters["absence_type_id"].(string); ok && absenceTypeID != "" {
		query = query.Where(leaveallowance.AbsenceTypeID(absenceTypeID))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count leave allowances failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list leave allowances failed")
	}

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	entities, err := query.Order(ent.Asc(leaveallowance.FieldYear)).All(ctx)
	if err != nil {
		r.log.Errorf("list leave allowances failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list leave allowances failed")
	}

	return entities, total, nil
}

func (r *LeaveAllowanceRepo) GetByUserAndTypeAndYear(ctx context.Context, tenantID uint32, userID uint32, absenceTypeID string, year int) (*ent.LeaveAllowance, error) {
	entity, err := r.entClient.Client().LeaveAllowance.Query().
		Where(
			leaveallowance.TenantID(tenantID),
			leaveallowance.UserID(userID),
			leaveallowance.AbsenceTypeID(absenceTypeID),
			leaveallowance.Year(year),
		).
		WithAbsenceType().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get leave allowance by user/type/year failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get leave allowance failed")
	}
	return entity, nil
}

func (r *LeaveAllowanceRepo) GetByUserAndYear(ctx context.Context, tenantID uint32, userID uint32, year int) ([]*ent.LeaveAllowance, error) {
	entities, err := r.entClient.Client().LeaveAllowance.Query().
		Where(
			leaveallowance.TenantID(tenantID),
			leaveallowance.UserID(userID),
			leaveallowance.Year(year),
		).
		WithAbsenceType().
		All(ctx)
	if err != nil {
		r.log.Errorf("get user allowances failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get user allowances failed")
	}
	return entities, nil
}

func (r *LeaveAllowanceRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*ent.LeaveAllowance, error) {
	update := r.entClient.Client().LeaveAllowance.UpdateOneID(id)

	if totalDays, ok := updates["total_days"].(float64); ok {
		update = update.SetTotalDays(totalDays)
	}
	if usedDays, ok := updates["used_days"].(float64); ok {
		update = update.SetUsedDays(usedDays)
	}
	if carriedOver, ok := updates["carried_over"].(float64); ok {
		update = update.SetCarriedOver(carriedOver)
	}
	if notes, ok := updates["notes"].(string); ok {
		update = update.SetNotes(notes)
	}

	update = update.SetUpdateTime(time.Now())

	entity, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, hrV1.ErrorAllowanceNotFound("leave allowance not found")
		}
		r.log.Errorf("update leave allowance failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("update leave allowance failed")
	}
	return entity, nil
}

func (r *LeaveAllowanceRepo) AddUsedDays(ctx context.Context, id string, days float64) error {
	_, err := r.entClient.Client().LeaveAllowance.UpdateOneID(id).
		AddUsedDays(days).
		SetUpdateTime(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("add used days failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("update allowance failed")
	}
	return nil
}

func (r *LeaveAllowanceRepo) Delete(ctx context.Context, id string) error {
	err := r.entClient.Client().LeaveAllowance.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return hrV1.ErrorAllowanceNotFound("leave allowance not found")
		}
		r.log.Errorf("delete leave allowance failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete leave allowance failed")
	}
	return nil
}
