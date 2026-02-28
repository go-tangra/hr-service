package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/data"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type LeaveService struct {
	hrV1.UnimplementedHrLeaveServiceServer

	log              *log.Helper
	leaveRequestRepo *data.LeaveRequestRepo
	allowanceRepo    *data.LeaveAllowanceRepo
	absenceTypeRepo  *data.AbsenceTypeRepo
}

func NewLeaveService(ctx *bootstrap.Context, leaveRequestRepo *data.LeaveRequestRepo, allowanceRepo *data.LeaveAllowanceRepo, absenceTypeRepo *data.AbsenceTypeRepo) *LeaveService {
	return &LeaveService{
		log:              ctx.NewLoggerHelper("hr/service/leave"),
		leaveRequestRepo: leaveRequestRepo,
		allowanceRepo:    allowanceRepo,
		absenceTypeRepo:  absenceTypeRepo,
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
		filters["start_date"] = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		filters["end_date"] = req.EndDate.AsTime()
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

	entity, err := s.leaveRequestRepo.UpdateStatus(ctx, req.GetId(), "approved", getUserID(ctx), getUsername(ctx), reviewNotes)
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
	startDate := req.GetStartDate().AsTime()
	endDate := req.GetEndDate().AsTime()
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
	default:
		return ""
	}
}
