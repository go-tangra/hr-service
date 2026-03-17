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

// isPoolBased returns true if the absence type uses a shared allowance pool.
func isPoolBased(absType *ent.AbsenceType) bool {
	return absType != nil && absType.AllowancePoolID != ""
}

// deductAllowance atomically checks balance and deducts days for a leave request's absence type.
// Returns the allowance ID that was deducted (for storing on the request), or "" if no deduction.
func deductAllowance(ctx context.Context, allowanceRepo *data.LeaveAllowanceRepo, leaveReq *ent.LeaveRequest) (string, error) {
	if leaveReq.Edges.AbsenceType == nil || !leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		return "", nil
	}

	tid := entityTenantID(leaveReq)

	if isPoolBased(leaveReq.Edges.AbsenceType) {
		return allowanceRepo.DeductPoolWithBalanceCheck(ctx, tid, leaveReq.UserID, leaveReq.Edges.AbsenceType.AllowancePoolID, leaveReq.StartDate.Year(), leaveReq.Days)
	}

	return allowanceRepo.DeductWithBalanceCheck(ctx, tid, leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year(), leaveReq.Days)
}

// refundAllowance returns previously deducted days to the correct allowance.
// Uses the stored deducted_allowance_id when available for accuracy.
// Falls back to lookup by type/pool if the ID is not stored (legacy requests).
func refundAllowance(ctx context.Context, log *log.Helper, allowanceRepo *data.LeaveAllowanceRepo, leaveReq *ent.LeaveRequest) {
	if leaveReq.Edges.AbsenceType == nil || !leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		return
	}

	if leaveReq.Days <= 0 {
		return
	}

	// Preferred path: use the stored allowance ID for exact refund
	if leaveReq.DeductedAllowanceID != "" {
		if err := allowanceRepo.RefundWithFloorCheck(ctx, leaveReq.DeductedAllowanceID, leaveReq.Days); err != nil {
			log.Errorf("Failed to refund allowance %s for leave %s: %v", leaveReq.DeductedAllowanceID, leaveReq.ID, err)
		}
		return
	}

	// Fallback: look up by type/pool (legacy requests without deducted_allowance_id)
	tid := entityTenantID(leaveReq)
	var allowanceID string

	if isPoolBased(leaveReq.Edges.AbsenceType) {
		a, err := allowanceRepo.GetByUserAndPoolAndYear(ctx, tid, leaveReq.UserID, leaveReq.Edges.AbsenceType.AllowancePoolID, leaveReq.StartDate.Year())
		if err != nil {
			log.Errorf("Failed to look up pool allowance for refund on leave %s: %v", leaveReq.ID, err)
			return
		}
		if a != nil {
			allowanceID = a.ID
		}
	} else {
		a, err := allowanceRepo.GetByUserAndTypeAndYear(ctx, tid, leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year())
		if err != nil {
			log.Errorf("Failed to look up allowance for refund on leave %s: %v", leaveReq.ID, err)
			return
		}
		if a != nil {
			allowanceID = a.ID
		}
	}

	if allowanceID == "" {
		log.Warnf("No allowance found for refund on leave %s", leaveReq.ID)
		return
	}

	if err := allowanceRepo.RefundWithFloorCheck(ctx, allowanceID, leaveReq.Days); err != nil {
		log.Errorf("Failed to refund allowance for leave %s: %v", leaveReq.ID, err)
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
