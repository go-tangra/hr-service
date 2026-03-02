package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/client"
	"github.com/go-tangra/go-tangra-hr/internal/data"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
	paperlesspb "github.com/go-tangra/go-tangra-paperless/gen/go/paperless/service/v1"
)

type LeaveService struct {
	hrV1.UnimplementedHrLeaveServiceServer

	log              *log.Helper
	leaveRequestRepo *data.LeaveRequestRepo
	allowanceRepo    *data.LeaveAllowanceRepo
	absenceTypeRepo  *data.AbsenceTypeRepo
	paperlessClient  *client.PaperlessClient
}

func NewLeaveService(ctx *bootstrap.Context, leaveRequestRepo *data.LeaveRequestRepo, allowanceRepo *data.LeaveAllowanceRepo, absenceTypeRepo *data.AbsenceTypeRepo, paperlessClient *client.PaperlessClient) *LeaveService {
	return &LeaveService{
		log:              ctx.NewLoggerHelper("hr/service/leave"),
		leaveRequestRepo: leaveRequestRepo,
		allowanceRepo:    allowanceRepo,
		absenceTypeRepo:  absenceTypeRepo,
		paperlessClient:  paperlessClient,
	}
}

func (s *LeaveService) CreateLeaveRequest(ctx context.Context, req *hrV1.CreateLeaveRequestRequest) (*hrV1.CreateLeaveRequestResponse, error) {
	tenantID := req.GetTenantId()
	userID := req.GetUserId()

	// Validate absence type exists
	absType, err := s.absenceTypeRepo.GetByID(ctx, req.GetAbsenceTypeId())
	if err != nil {
		return nil, err
	}
	if absType == nil {
		return nil, hrV1.ErrorAbsenceTypeNotFound("absence type not found")
	}

	startDate := req.GetStartDate().AsTime()
	endDate := req.GetEndDate().AsTime()

	if endDate.Before(startDate) {
		return nil, hrV1.ErrorInvalidDateRange("end date must be after start date")
	}

	// Calculate business days
	days := req.GetDays()
	if days <= 0 {
		days = calculateBusinessDays(startDate, endDate)
	}

	// Check for overlapping requests
	overlap, err := s.leaveRequestRepo.CheckOverlap(ctx, tenantID, userID, startDate, endDate, "")
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, hrV1.ErrorOverlapExists("overlapping leave request exists for this period")
	}

	// Check allowance balance if type deducts from allowance
	if absType.DeductsFromAllowance {
		allowance, err := s.allowanceRepo.GetByUserAndTypeAndYear(ctx, tenantID, userID, req.GetAbsenceTypeId(), startDate.Year())
		if err != nil {
			return nil, err
		}
		if allowance == nil {
			return nil, hrV1.ErrorInsufficientAllowance("no leave allowance configured for this type and year")
		}
		remaining := allowance.TotalDays + allowance.CarriedOver - allowance.UsedDays
		if days > remaining {
			return nil, hrV1.ErrorInsufficientAllowance(fmt.Sprintf("insufficient allowance: %.1f days requested, %.1f days remaining", days, remaining))
		}
	}

	// Determine initial status
	status := "pending"
	if !absType.RequiresApproval {
		status = "approved"
	}

	opts := []func(*ent.LeaveRequestCreate){}
	if req.Reason != nil {
		opts = append(opts, func(c *ent.LeaveRequestCreate) { c.SetReason(*req.Reason) })
	}
	if req.Notes != nil {
		opts = append(opts, func(c *ent.LeaveRequestCreate) { c.SetNotes(*req.Notes) })
	}
	if req.Metadata != nil {
		opts = append(opts, func(c *ent.LeaveRequestCreate) { c.SetMetadata(req.Metadata.AsMap()) })
	}
	if req.UserName != nil {
		opts = append(opts, func(c *ent.LeaveRequestCreate) { c.SetUserName(*req.UserName) })
	}
	if req.OrgUnitName != nil {
		opts = append(opts, func(c *ent.LeaveRequestCreate) { c.SetOrgUnitName(*req.OrgUnitName) })
	}

	entity, err := s.leaveRequestRepo.Create(ctx, tenantID, userID, req.GetAbsenceTypeId(), startDate, endDate, days, status, opts...)
	if err != nil {
		return nil, err
	}

	// If auto-approved and deducts from allowance, deduct immediately
	if status == "approved" && absType.DeductsFromAllowance {
		allowance, _ := s.allowanceRepo.GetByUserAndTypeAndYear(ctx, tenantID, userID, req.GetAbsenceTypeId(), startDate.Year())
		if allowance != nil {
			_ = s.allowanceRepo.AddUsedDays(ctx, allowance.ID, days)
		}
	}

	// Re-fetch with edges
	entity, _ = s.leaveRequestRepo.GetByID(ctx, entity.ID)

	return &hrV1.CreateLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) GetLeaveRequest(ctx context.Context, req *hrV1.GetLeaveRequestRequest) (*hrV1.GetLeaveRequestResponse, error) {
	entity, err := s.leaveRequestRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
	}

	return &hrV1.GetLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) ListLeaveRequests(ctx context.Context, req *hrV1.ListLeaveRequestsRequest) (*hrV1.ListLeaveRequestsResponse, error) {
	filters := make(map[string]interface{})
	if req.UserId != nil {
		filters["user_id"] = *req.UserId
	}
	if req.AbsenceTypeId != nil {
		filters["absence_type_id"] = *req.AbsenceTypeId
	}
	if req.Status != nil && *req.Status != hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_UNSPECIFIED {
		filters["status"] = leaveStatusToString(*req.Status)
	}
	if req.StartDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.StartDate); err == nil {
			filters["start_date"] = t
		}
	}
	if req.EndDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.EndDate); err == nil {
			filters["end_date"] = t
		}
	}

	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if req.GetNoPaging() {
		page = 0
		pageSize = 0
	}

	entities, total, err := s.leaveRequestRepo.List(ctx, req.GetTenantId(), page, pageSize, filters)
	if err != nil {
		return nil, err
	}

	items := make([]*hrV1.LeaveRequest, len(entities))
	for i, e := range entities {
		items[i] = leaveRequestToProto(e)
	}

	return &hrV1.ListLeaveRequestsResponse{
		Items: items,
		Total: ptrInt32(int32(total)),
	}, nil
}

func (s *LeaveService) UpdateLeaveRequest(ctx context.Context, req *hrV1.UpdateLeaveRequestRequest) (*hrV1.UpdateLeaveRequestResponse, error) {
	updates := make(map[string]interface{})

	if req.Data != nil {
		if req.Data.Reason != nil {
			updates["reason"] = *req.Data.Reason
		}
		if req.Data.Notes != nil {
			updates["notes"] = *req.Data.Notes
		}
		if req.Data.Metadata != nil {
			updates["metadata"] = req.Data.Metadata.AsMap()
		}
	}

	entity, err := s.leaveRequestRepo.Update(ctx, req.GetId(), updates)
	if err != nil {
		return nil, err
	}

	// Re-fetch with edges
	entity, _ = s.leaveRequestRepo.GetByID(ctx, entity.ID)

	return &hrV1.UpdateLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) DeleteLeaveRequest(ctx context.Context, req *hrV1.DeleteLeaveRequestRequest) (*emptypb.Empty, error) {
	err := s.leaveRequestRepo.Delete(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *LeaveService) ApproveLeaveRequest(ctx context.Context, req *hrV1.ApproveLeaveRequestRequest) (*hrV1.ApproveLeaveRequestResponse, error) {
	existing, err := s.leaveRequestRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
	}
	if existing.Status.String() != "pending" {
		return nil, hrV1.ErrorBadRequest("only pending requests can be approved")
	}

	reviewNotes := ""
	if req.ReviewNotes != nil {
		reviewNotes = *req.ReviewNotes
	}

	// Check if absence type requires signing
	absType := existing.Edges.AbsenceType
	if absType != nil && absType.RequiresSigning && absType.SigningTemplateID != "" {
		return s.approveWithSigning(ctx, existing, absType, req, reviewNotes)
	}

	// Standard approval flow (no signing required)
	return s.approveImmediate(ctx, existing, req.GetId(), reviewNotes)
}

// approveWithSigning creates a signing request in paperless and sets status to awaiting_signing
func (s *LeaveService) approveWithSigning(ctx context.Context, existing *ent.LeaveRequest, absType *ent.AbsenceType, req *hrV1.ApproveLeaveRequestRequest, reviewNotes string) (*hrV1.ApproveLeaveRequestResponse, error) {
	// Store reviewer info with awaiting_signing status
	entity, err := s.leaveRequestRepo.UpdateStatus(ctx, existing.ID, "awaiting_signing", getUserID(ctx), getUsername(ctx), reviewNotes)
	if err != nil {
		return nil, err
	}

	// Build signing recipients: approver signs first, then employee
	approverName := getUsername(ctx)
	if req.ApproverName != nil && *req.ApproverName != "" {
		approverName = *req.ApproverName
	}

	recipients := []*paperlesspb.SigningRecipientInput{
		{
			Email:        req.GetApproverEmail(),
			Name:         approverName,
			SigningOrder: 1,
		},
		{
			Email:        req.GetRequesterEmail(),
			Name:         existing.UserName,
			SigningOrder: 2,
		},
	}

	// Prefill template fields with leave request data
	fieldValues := []*paperlesspb.SigningFieldValueInput{
		{FieldId: "StartDate", Value: existing.StartDate.Format("2006-01-02")},
		{FieldId: "EndDate", Value: existing.EndDate.Format("2006-01-02")},
		{FieldId: "TotalDays", Value: fmt.Sprintf("%.1f", existing.Days)},
	}

	requestName := fmt.Sprintf("Leave Request - %s (%s to %s)",
		existing.UserName,
		existing.StartDate.Format("2006-01-02"),
		existing.EndDate.Format("2006-01-02"),
	)

	message := fmt.Sprintf("Please sign the absence approval for %s. Period: %s to %s (%.1f days).",
		existing.UserName,
		existing.StartDate.Format("2006-01-02"),
		existing.EndDate.Format("2006-01-02"),
		existing.Days,
	)

	signingRequestID, err := s.paperlessClient.CreateSigningRequest(
		ctx,
		absType.SigningTemplateID,
		requestName,
		recipients,
		fieldValues,
		message,
	)
	if err != nil {
		// Roll back to pending on failure
		s.log.Errorf("Failed to create signing request, rolling back to pending: %v", err)
		_, _ = s.leaveRequestRepo.UpdateStatus(ctx, existing.ID, "pending", 0, "", "")
		return nil, hrV1.ErrorInternalServerError("failed to create signing request")
	}

	// Store the signing request ID on the leave request
	if err := s.leaveRequestRepo.SetSigningRequestID(ctx, existing.ID, signingRequestID); err != nil {
		s.log.Errorf("Failed to store signing_request_id: %v", err)
	}

	// Re-fetch with edges
	entity2, _ := s.leaveRequestRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.ApproveLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

// approveImmediate performs standard approval without signing
func (s *LeaveService) approveImmediate(ctx context.Context, existing *ent.LeaveRequest, id string, reviewNotes string) (*hrV1.ApproveLeaveRequestResponse, error) {
	entity, err := s.leaveRequestRepo.UpdateStatus(ctx, id, "approved", getUserID(ctx), getUsername(ctx), reviewNotes)
	if err != nil {
		return nil, err
	}

	// Deduct from allowance if the absence type requires it
	if existing.Edges.AbsenceType != nil && existing.Edges.AbsenceType.DeductsFromAllowance {
		var tid uint32
		if existing.TenantID != nil {
			tid = *existing.TenantID
		}
		allowance, _ := s.allowanceRepo.GetByUserAndTypeAndYear(ctx, tid, existing.UserID, existing.AbsenceTypeID, existing.StartDate.Year())
		if allowance != nil {
			_ = s.allowanceRepo.AddUsedDays(ctx, allowance.ID, existing.Days)
		}
	}

	// Re-fetch with edges
	entity2, _ := s.leaveRequestRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.ApproveLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) RejectLeaveRequest(ctx context.Context, req *hrV1.RejectLeaveRequestRequest) (*hrV1.RejectLeaveRequestResponse, error) {
	existing, err := s.leaveRequestRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
	}
	if existing.Status.String() != "pending" {
		return nil, hrV1.ErrorBadRequest("only pending requests can be rejected")
	}

	reviewNotes := ""
	if req.ReviewNotes != nil {
		reviewNotes = *req.ReviewNotes
	}

	entity, err := s.leaveRequestRepo.UpdateStatus(ctx, req.GetId(), "rejected", getUserID(ctx), getUsername(ctx), reviewNotes)
	if err != nil {
		return nil, err
	}

	// Re-fetch with edges
	entity2, _ := s.leaveRequestRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.RejectLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) CancelLeaveRequest(ctx context.Context, req *hrV1.CancelLeaveRequestRequest) (*hrV1.CancelLeaveRequestResponse, error) {
	existing, err := s.leaveRequestRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorLeaveRequestNotFound("leave request not found")
	}

	wasApproved := existing.Status.String() == "approved"

	entity, err := s.leaveRequestRepo.UpdateStatus(ctx, req.GetId(), "cancelled", 0, "", "")
	if err != nil {
		return nil, err
	}

	// Refund allowance if was previously approved and deducts from allowance
	if wasApproved && existing.Edges.AbsenceType != nil && existing.Edges.AbsenceType.DeductsFromAllowance {
		var tid uint32
		if existing.TenantID != nil {
			tid = *existing.TenantID
		}
		allowance, _ := s.allowanceRepo.GetByUserAndTypeAndYear(ctx, tid, existing.UserID, existing.AbsenceTypeID, existing.StartDate.Year())
		if allowance != nil {
			_ = s.allowanceRepo.AddUsedDays(ctx, allowance.ID, -existing.Days)
		}
	}

	// Re-fetch with edges
	entity2, _ := s.leaveRequestRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.CancelLeaveRequestResponse{
		LeaveRequest: leaveRequestToProto(entity),
	}, nil
}

func (s *LeaveService) GetCalendarEvents(ctx context.Context, req *hrV1.GetCalendarEventsRequest) (*hrV1.GetCalendarEventsResponse, error) {
	startDate, _ := time.Parse(time.RFC3339, req.GetStartDate())
	endDate, _ := time.Parse(time.RFC3339, req.GetEndDate())
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, -1, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now().AddDate(0, 2, 0)
	}
	orgUnitName := ""
	if req.OrgUnitName != nil {
		orgUnitName = *req.OrgUnitName
	}
	var userID uint32
	if req.UserId != nil {
		userID = *req.UserId
	}

	entities, err := s.leaveRequestRepo.GetCalendarEvents(ctx, req.GetTenantId(), startDate, endDate, orgUnitName, userID)
	if err != nil {
		return nil, err
	}

	events := make([]*hrV1.CalendarEvent, len(entities))
	for i, e := range entities {
		event := &hrV1.CalendarEvent{
			Id:            e.ID,
			UserId:        e.UserID,
			UserName:      e.UserName,
			OrgUnitName:   e.OrgUnitName,
			AbsenceTypeId: e.AbsenceTypeID,
			StartDate:     timestamppb.New(e.StartDate),
			EndDate:       timestamppb.New(e.EndDate),
			Days:          e.Days,
			Status:        leaveStatusToProto(e.Status.String()),
		}

		if e.Edges.AbsenceType != nil {
			event.AbsenceTypeName = e.Edges.AbsenceType.Name
			event.Color = e.Edges.AbsenceType.Color
		}

		events[i] = event
	}

	return &hrV1.GetCalendarEventsResponse{
		Events: events,
	}, nil
}

func leaveRequestToProto(e *ent.LeaveRequest) *hrV1.LeaveRequest {
	if e == nil {
		return nil
	}

	result := &hrV1.LeaveRequest{
		Id:            &e.ID,
		TenantId:      e.TenantID,
		UserId:        &e.UserID,
		AbsenceTypeId: ptrString(e.AbsenceTypeID),
		StartDate:     timestamppb.New(e.StartDate),
		EndDate:       timestamppb.New(e.EndDate),
		Days:          ptrFloat64(e.Days),
		Status:        leaveStatusToProtoPtr(e.Status.String()),
		Reason:        ptrString(e.Reason),
		ReviewNotes:   ptrString(e.ReviewNotes),
		ReviewedBy:    &e.ReviewedBy,
		ReviewerName:  ptrString(e.ReviewerName),
		UserName:      ptrString(e.UserName),
		OrgUnitName:   ptrString(e.OrgUnitName),
		Notes:         ptrString(e.Notes),
		Metadata:      mapToStruct(e.Metadata),
		CreatedBy:     e.CreateBy,
		UpdatedBy:     e.UpdateBy,
	}

	if e.ReviewedAt != nil {
		result.ReviewedAt = timestamppb.New(*e.ReviewedAt)
	}
	if e.CreateTime != nil {
		result.CreatedAt = timestamppb.New(*e.CreateTime)
	}
	if e.UpdateTime != nil {
		result.UpdatedAt = timestamppb.New(*e.UpdateTime)
	}

	if e.SigningRequestID != "" {
		result.SigningRequestId = ptrString(e.SigningRequestID)
	}

	// Denormalized fields from edges
	if e.Edges.AbsenceType != nil {
		result.AbsenceTypeName = &e.Edges.AbsenceType.Name
		result.AbsenceTypeColor = ptrString(e.Edges.AbsenceType.Color)
	}

	return result
}

func leaveStatusToProto(status string) hrV1.LeaveRequestStatus {
	switch status {
	case "pending":
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_PENDING
	case "approved":
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_APPROVED
	case "rejected":
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_REJECTED
	case "cancelled":
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_CANCELLED
	case "awaiting_signing":
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_AWAITING_SIGNING
	default:
		return hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_UNSPECIFIED
	}
}

func leaveStatusToProtoPtr(status string) *hrV1.LeaveRequestStatus {
	s := leaveStatusToProto(status)
	return &s
}

func leaveStatusToString(status hrV1.LeaveRequestStatus) string {
	switch status {
	case hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_PENDING:
		return "pending"
	case hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_APPROVED:
		return "approved"
	case hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_REJECTED:
		return "rejected"
	case hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_CANCELLED:
		return "cancelled"
	case hrV1.LeaveRequestStatus_LEAVE_REQUEST_STATUS_AWAITING_SIGNING:
		return "awaiting_signing"
	default:
		return ""
	}
}
