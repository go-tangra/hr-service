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

	// Deduct from allowance if the absence type requires it
	if leaveReq.Edges.AbsenceType != nil && leaveReq.Edges.AbsenceType.DeductsFromAllowance {
		var tid uint32
		if leaveReq.TenantID != nil {
			tid = *leaveReq.TenantID
		}
		allowance, _ := h.allowanceRepo.GetByUserAndTypeAndYear(ctx, tid, leaveReq.UserID, leaveReq.AbsenceTypeID, leaveReq.StartDate.Year())
		if allowance != nil {
			_ = h.allowanceRepo.AddUsedDays(ctx, allowance.ID, leaveReq.Days)
		}
	}

	h.log.Infof("Leave request %s auto-approved after signing completed", leaveReq.ID)
	return nil
}
