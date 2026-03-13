package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-tangra/go-tangra-hr/internal/data"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
)

// entityTenantID extracts the tenant ID value from an entity's *uint32 TenantID field.
func entityTenantID(e *ent.LeaveRequest) uint32 {
	if e.TenantID != nil {
		return *e.TenantID
	}
	return 0
}

// deductAllowance atomically checks balance and deducts days for a leave request's absence type.
// Returns nil if the absence type doesn't deduct from allowance or has no allowance configured.
func deductAllowance(ctx context.Context, allowanceRepo *data.LeaveAllowanceRepo, leaveReq *ent.LeaveRequest) error {
	if leaveReq.Edges.AbsenceType == nil || !leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		return nil
	}
	_, err := allowanceRepo.DeductWithBalanceCheck(ctx, entityTenantID(leaveReq), leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year(), leaveReq.Days)
	return err
}

// refundAllowance returns previously deducted days to the user's allowance.
// Errors are logged but not returned, since refunds are best-effort during cancellation.
func refundAllowance(ctx context.Context, log *log.Helper, allowanceRepo *data.LeaveAllowanceRepo, leaveReq *ent.LeaveRequest) {
	if leaveReq.Edges.AbsenceType == nil || !leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		return
	}
	tid := entityTenantID(leaveReq)
	allowance, err := allowanceRepo.GetByUserAndTypeAndYear(ctx, tid, leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year())
	if err != nil {
		log.Errorf("Failed to look up allowance for refund on leave %s: %v", leaveReq.ID, err)
		return
	}
	if allowance != nil {
		// Only refund if used_days would stay >= 0 to prevent negative balances
		refund := leaveReq.Days
		if allowance.UsedDays-refund < 0 {
			refund = allowance.UsedDays
		}
		if refund > 0 {
			if err = allowanceRepo.AddUsedDays(ctx, allowance.ID, -refund); err != nil {
				log.Errorf("Failed to refund allowance for leave %s: %v", leaveReq.ID, err)
			}
		}
	}
}

// calculateBusinessDays calculates the number of business days (weekdays) between two dates, inclusive
func calculateBusinessDays(start, end time.Time) float64 {
	// Normalize to start of day
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	if end.Before(start) {
		return 0
	}

	days := 0.0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd != time.Saturday && wd != time.Sunday {
			days++
		}
	}

	return days
}
