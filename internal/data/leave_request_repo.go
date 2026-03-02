package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/leaverequest"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type LeaveRequestRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewLeaveRequestRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LeaveRequestRepo {
	return &LeaveRequestRepo{
		log:       ctx.NewLoggerHelper("hr/leave_request/repo"),
		entClient: entClient,
	}
}

func (r *LeaveRequestRepo) Create(ctx context.Context, tenantID uint32, userID uint32, absenceTypeID string, startDate, endDate time.Time, days float64, status string, opts ...func(*ent.LeaveRequestCreate)) (*ent.LeaveRequest, error) {
	id := uuid.New().String()

	create := r.entClient.Client().LeaveRequest.Create().
		SetID(id).
		SetTenantID(tenantID).
		SetUserID(userID).
		SetAbsenceTypeID(absenceTypeID).
		SetStartDate(startDate).
		SetEndDate(endDate).
		SetDays(days).
		SetStatus(leaverequest.Status(status)).
		SetCreateTime(time.Now())

	for _, opt := range opts {
		opt(create)
	}

	entity, err := create.Save(ctx)
	if err != nil {
		r.log.Errorf("create leave request failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("create leave request failed")
	}
	return entity, nil
}

func (r *LeaveRequestRepo) GetByID(ctx context.Context, id string) (*ent.LeaveRequest, error) {
	entity, err := r.entClient.Client().LeaveRequest.Query().
		Where(leaverequest.ID(id)).
		WithAbsenceType().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get leave request failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get leave request failed")
	}
	return entity, nil
}

func (r *LeaveRequestRepo) List(ctx context.Context, tenantID uint32, page, pageSize int, filters map[string]interface{}) ([]*ent.LeaveRequest, int, error) {
	query := r.entClient.Client().LeaveRequest.Query().
		Where(leaverequest.TenantID(tenantID)).
		WithAbsenceType()

	if userID, ok := filters["user_id"].(uint32); ok && userID > 0 {
		query = query.Where(leaverequest.UserID(userID))
	}
	if absenceTypeID, ok := filters["absence_type_id"].(string); ok && absenceTypeID != "" {
		query = query.Where(leaverequest.AbsenceTypeID(absenceTypeID))
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where(leaverequest.StatusEQ(leaverequest.Status(status)))
	}
	if startDate, ok := filters["start_date"].(time.Time); ok {
		query = query.Where(leaverequest.StartDateGTE(startDate))
	}
	if endDate, ok := filters["end_date"].(time.Time); ok {
		query = query.Where(leaverequest.EndDateLTE(endDate))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count leave requests failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list leave requests failed")
	}

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	entities, err := query.Order(ent.Desc(leaverequest.FieldCreateTime)).All(ctx)
	if err != nil {
		r.log.Errorf("list leave requests failed: %s", err.Error())
		return nil, 0, hrV1.ErrorInternalServerError("list leave requests failed")
	}

	return entities, total, nil
}

func (r *LeaveRequestRepo) CheckOverlap(ctx context.Context, tenantID uint32, userID uint32, startDate, endDate time.Time, excludeID string) (bool, error) {
	query := r.entClient.Client().LeaveRequest.Query().
		Where(
			leaverequest.TenantID(tenantID),
			leaverequest.UserID(userID),
			leaverequest.StatusIn(leaverequest.StatusPending, leaverequest.StatusApproved, leaverequest.StatusAwaitingSigning),
			leaverequest.StartDateLT(endDate),
			leaverequest.EndDateGT(startDate),
		)

	if excludeID != "" {
		query = query.Where(leaverequest.IDNEQ(excludeID))
	}

	count, err := query.Count(ctx)
	if err != nil {
		r.log.Errorf("check overlap failed: %s", err.Error())
		return false, hrV1.ErrorInternalServerError("check overlap failed")
	}

	return count > 0, nil
}

func (r *LeaveRequestRepo) GetCalendarEvents(ctx context.Context, tenantID uint32, startDate, endDate time.Time, orgUnitName string, userID uint32) ([]*ent.LeaveRequest, error) {
	query := r.entClient.Client().LeaveRequest.Query().
		Where(
			leaverequest.TenantID(tenantID),
			leaverequest.StatusIn(leaverequest.StatusPending, leaverequest.StatusApproved, leaverequest.StatusAwaitingSigning),
			leaverequest.StartDateLT(endDate),
			leaverequest.EndDateGT(startDate),
		).
		WithAbsenceType()

	if userID > 0 {
		query = query.Where(leaverequest.UserID(userID))
	}
	if orgUnitName != "" {
		query = query.Where(leaverequest.OrgUnitName(orgUnitName))
	}

	entities, err := query.Order(ent.Asc(leaverequest.FieldStartDate)).All(ctx)
	if err != nil {
		r.log.Errorf("get calendar events failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get calendar events failed")
	}

	return entities, nil
}

func (r *LeaveRequestRepo) UpdateStatus(ctx context.Context, id string, status string, reviewedBy uint32, reviewerName string, reviewNotes string) (*ent.LeaveRequest, error) {
	update := r.entClient.Client().LeaveRequest.UpdateOneID(id).
		SetStatus(leaverequest.Status(status)).
		SetUpdateTime(time.Now())

	if reviewedBy > 0 {
		update = update.SetReviewedBy(reviewedBy)
		update = update.SetReviewerName(reviewerName)
		now := time.Now()
		update = update.SetReviewedAt(now)
	}
	if reviewNotes != "" {
		update = update.SetReviewNotes(reviewNotes)
	}

	entity, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
		}
		r.log.Errorf("update leave request status failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("update leave request status failed")
	}
	return entity, nil
}

func (r *LeaveRequestRepo) Update(ctx context.Context, id string, updates map[string]interface{}) (*ent.LeaveRequest, error) {
	update := r.entClient.Client().LeaveRequest.UpdateOneID(id)

	if reason, ok := updates["reason"].(string); ok {
		update = update.SetReason(reason)
	}
	if notes, ok := updates["notes"].(string); ok {
		update = update.SetNotes(notes)
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		update = update.SetMetadata(metadata)
	}

	update = update.SetUpdateTime(time.Now())

	entity, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
		}
		r.log.Errorf("update leave request failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("update leave request failed")
	}
	return entity, nil
}

func (r *LeaveRequestRepo) Delete(ctx context.Context, id string) error {
	err := r.entClient.Client().LeaveRequest.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return hrV1.ErrorLeaveRequestNotFound("leave request not found")
		}
		r.log.Errorf("delete leave request failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("delete leave request failed")
	}
	return nil
}

func (r *LeaveRequestRepo) GetBySigningRequestID(ctx context.Context, signingRequestID string) (*ent.LeaveRequest, error) {
	entity, err := r.entClient.Client().LeaveRequest.Query().
		Where(leaverequest.SigningRequestID(signingRequestID)).
		WithAbsenceType().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("get leave request by signing_request_id failed: %s", err.Error())
		return nil, hrV1.ErrorInternalServerError("get leave request by signing_request_id failed")
	}
	return entity, nil
}

func (r *LeaveRequestRepo) SetSigningRequestID(ctx context.Context, id string, signingRequestID string) error {
	err := r.entClient.Client().LeaveRequest.UpdateOneID(id).
		SetSigningRequestID(signingRequestID).
		SetUpdateTime(time.Now()).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return hrV1.ErrorLeaveRequestNotFound("leave request not found")
		}
		r.log.Errorf("set signing_request_id failed: %s", err.Error())
		return hrV1.ErrorInternalServerError("set signing_request_id failed")
	}
	return nil
}

func (r *LeaveRequestRepo) CountByStatus(ctx context.Context, tenantID uint32, status string) (int, error) {
	return r.entClient.Client().LeaveRequest.Query().
		Where(leaverequest.TenantID(tenantID), leaverequest.StatusEQ(leaverequest.Status(status))).
		Count(ctx)
}

func (r *LeaveRequestRepo) Count(ctx context.Context, tenantID uint32) (int, error) {
	return r.entClient.Client().LeaveRequest.Query().Where(leaverequest.TenantID(tenantID)).Count(ctx)
}
