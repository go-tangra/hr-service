package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/leaverequest"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

// StatisticsRepo provides methods for collecting HR statistics.
type StatisticsRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

// NewStatisticsRepo creates a new StatisticsRepo.
func NewStatisticsRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StatisticsRepo {
	return &StatisticsRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("hr/statistics/repo"),
	}
}

// GetAbsenceTypeCount returns the total number of absence types across all tenants.
func (r *StatisticsRepo) GetAbsenceTypeCount(ctx context.Context) (int, error) {
	count, err := r.entClient.Client().AbsenceType.Query().Count(ctx)
	if err != nil {
		r.log.Errorf("get absence type count failed: %s", err.Error())
		return 0, hrV1.ErrorInternalServerError("get statistics failed")
	}
	return count, nil
}

// GetLeaveRequestCountByStatus returns the count of leave requests grouped by status across all tenants.
func (r *StatisticsRepo) GetLeaveRequestCountByStatus(ctx context.Context) (map[string]int, error) {
	result := make(map[string]int)
	statuses := []leaverequest.Status{
		leaverequest.StatusPending,
		leaverequest.StatusApproved,
		leaverequest.StatusRejected,
		leaverequest.StatusCancelled,
		leaverequest.StatusAwaitingSigning,
	}
	for _, status := range statuses {
		count, err := r.entClient.Client().LeaveRequest.Query().
			Where(leaverequest.StatusEQ(status)).
			Count(ctx)
		if err != nil {
			r.log.Errorf("get leave request count by status failed: %s", err.Error())
			return nil, hrV1.ErrorInternalServerError("get statistics failed")
		}
		if count > 0 {
			result[string(status)] = count
		}
	}
	return result, nil
}

// GetLeaveAllowanceCount returns the total number of leave allowances across all tenants.
func (r *StatisticsRepo) GetLeaveAllowanceCount(ctx context.Context) (int, error) {
	count, err := r.entClient.Client().LeaveAllowance.Query().Count(ctx)
	if err != nil {
		r.log.Errorf("get leave allowance count failed: %s", err.Error())
		return 0, hrV1.ErrorInternalServerError("get statistics failed")
	}
	return count, nil
}
