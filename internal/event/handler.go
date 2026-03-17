package event

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/data"
)

// Handler handles signing events from the paperless service
type Handler struct {
	log              *log.Helper
	leaveRequestRepo *data.LeaveRequestRepo
	allowanceRepo    *data.LeaveAllowanceRepo
	absenceTypeRepo  *data.AbsenceTypeRepo
}

// NewHandler creates a new event handler
func NewHandler(ctx *bootstrap.Context, leaveRequestRepo *data.LeaveRequestRepo, allowanceRepo *data.LeaveAllowanceRepo, absenceTypeRepo *data.AbsenceTypeRepo) *Handler {
	return &Handler{
		log:              ctx.NewLoggerHelper("hr/event/handler"),
		leaveRequestRepo: leaveRequestRepo,
		allowanceRepo:    allowanceRepo,
		absenceTypeRepo:  absenceTypeRepo,
	}
}

// HandleSigningCompleted handles a signing.request.completed event
func (h *Handler) HandleSigningCompleted(ctx context.Context, data *SigningRequestCompletedData) error {
	h.log.Infof("Handling signing completed: request_id=%s, tenant_id=%d", data.RequestID, data.TenantID)

	// Look up the leave request by signing_request_id
	leaveReq, err := h.leaveRequestRepo.GetBySigningRequestID(ctx, data.RequestID)
	if err != nil {
		return err
	}
	if leaveReq == nil {
		h.log.Infof("No leave request found for signing_request_id=%s, ignoring", data.RequestID)
		return nil
	}

	// Verify the leave request is in awaiting_signing status
	if leaveReq.Status.String() != "awaiting_signing" {
		h.log.Infof("Leave request %s is not awaiting_signing (status=%s), ignoring", leaveReq.ID, leaveReq.Status)
		return nil
	}

	// Approve the leave request (reviewer info was already stored when the signing was initiated)
	_, err = h.leaveRequestRepo.UpdateStatus(ctx, leaveReq.ID, "approved", 0, "", "")
	if err != nil {
		h.log.Errorf("Failed to approve leave request %s after signing: %v", leaveReq.ID, err)
		return err
	}

	// Atomically deduct from allowance if the absence type requires it
	if leaveReq.Edges.AbsenceType != nil && leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		var tid uint32
		if leaveReq.TenantID != nil {
			tid = *leaveReq.TenantID
		}

		var allowanceID string
		var deductErr error
		if leaveReq.Edges.AbsenceType.AllowancePoolID != "" {
			allowanceID, deductErr = h.allowanceRepo.DeductPoolWithBalanceCheck(ctx, tid, leaveReq.UserID, leaveReq.Edges.AbsenceType.AllowancePoolID, leaveReq.StartDate.Year(), leaveReq.Days)
		} else {
			allowanceID, deductErr = h.allowanceRepo.DeductWithBalanceCheck(ctx, tid, leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year(), leaveReq.Days)
		}
		if deductErr != nil {
			h.log.Errorf("Failed to deduct allowance for leave %s after signing: %v", leaveReq.ID, deductErr)
			return deductErr
		}

		// Store which allowance was deducted for accurate refunds
		if allowanceID != "" {
			if setErr := h.leaveRequestRepo.SetDeductedAllowanceID(ctx, leaveReq.ID, allowanceID); setErr != nil {
				h.log.Errorf("Failed to store deducted_allowance_id on leave %s: %v", leaveReq.ID, setErr)
			}
		}
	}

	h.log.Infof("Leave request %s auto-approved after signing completed", leaveReq.ID)
	return nil
}
