package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/data"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type AllowanceService struct {
	hrV1.UnimplementedHrAllowanceServiceServer

	log             *log.Helper
	allowanceRepo   *data.LeaveAllowanceRepo
	absenceTypeRepo *data.AbsenceTypeRepo
}

func NewAllowanceService(ctx *bootstrap.Context, allowanceRepo *data.LeaveAllowanceRepo, absenceTypeRepo *data.AbsenceTypeRepo) *AllowanceService {
	return &AllowanceService{
		log:             ctx.NewLoggerHelper("hr/service/allowance"),
		allowanceRepo:   allowanceRepo,
		absenceTypeRepo: absenceTypeRepo,
	}
}

func (s *AllowanceService) CreateAllowance(ctx context.Context, req *hrV1.CreateAllowanceRequest) (*hrV1.CreateAllowanceResponse, error) {
	opts := []func(*ent.LeaveAllowanceCreate){}

	if req.CarriedOver != nil {
		opts = append(opts, func(c *ent.LeaveAllowanceCreate) { c.SetCarriedOver(*req.CarriedOver) })
	}
	if req.Notes != nil {
		opts = append(opts, func(c *ent.LeaveAllowanceCreate) { c.SetNotes(*req.Notes) })
	}
	if req.UserName != nil {
		opts = append(opts, func(c *ent.LeaveAllowanceCreate) { c.SetUserName(*req.UserName) })
	}

	entity, err := s.allowanceRepo.Create(ctx, req.GetTenantId(), req.GetUserId(), req.GetAbsenceTypeId(), int(req.GetYear()), req.GetTotalDays(), opts...)
	if err != nil {
		return nil, err
	}

	// Re-fetch with edges
	entity, _ = s.allowanceRepo.GetByID(ctx, entity.ID)

	return &hrV1.CreateAllowanceResponse{
		Allowance: allowanceToProto(entity),
	}, nil
}

func (s *AllowanceService) GetAllowance(ctx context.Context, req *hrV1.GetAllowanceRequest) (*hrV1.GetAllowanceResponse, error) {
	entity, err := s.allowanceRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, hrV1.ErrorAllowanceNotFound("leave allowance not found")
	}

	return &hrV1.GetAllowanceResponse{
		Allowance: allowanceToProto(entity),
	}, nil
}

func (s *AllowanceService) ListAllowances(ctx context.Context, req *hrV1.ListAllowancesRequest) (*hrV1.ListAllowancesResponse, error) {
	filters := make(map[string]interface{})
	if req.UserId != nil {
		filters["user_id"] = *req.UserId
	}
	if req.Year != nil {
		filters["year"] = int(*req.Year)
	}
	if req.AbsenceTypeId != nil {
		filters["absence_type_id"] = *req.AbsenceTypeId
	}

	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if req.GetNoPaging() {
		page = 0
		pageSize = 0
	}

	entities, total, err := s.allowanceRepo.List(ctx, req.GetTenantId(), page, pageSize, filters)
	if err != nil {
		return nil, err
	}

	items := make([]*hrV1.LeaveAllowance, len(entities))
	for i, e := range entities {
		items[i] = allowanceToProto(e)
	}

	return &hrV1.ListAllowancesResponse{
		Items: items,
		Total: ptrInt32(int32(total)),
	}, nil
}

func (s *AllowanceService) UpdateAllowance(ctx context.Context, req *hrV1.UpdateAllowanceRequest) (*hrV1.UpdateAllowanceResponse, error) {
	updates := make(map[string]interface{})

	if req.Data != nil {
		if req.Data.TotalDays != nil {
			updates["total_days"] = *req.Data.TotalDays
		}
		if req.Data.CarriedOver != nil {
			updates["carried_over"] = *req.Data.CarriedOver
		}
		if req.Data.Notes != nil {
			updates["notes"] = *req.Data.Notes
		}
	}

	entity, err := s.allowanceRepo.Update(ctx, req.GetId(), updates)
	if err != nil {
		return nil, err
	}

	// Re-fetch with edges
	entity2, _ := s.allowanceRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.UpdateAllowanceResponse{
		Allowance: allowanceToProto(entity),
	}, nil
}

func (s *AllowanceService) DeleteAllowance(ctx context.Context, req *hrV1.DeleteAllowanceRequest) (*emptypb.Empty, error) {
	err := s.allowanceRepo.Delete(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AllowanceService) GetUserBalance(ctx context.Context, req *hrV1.GetUserBalanceRequest) (*hrV1.GetUserBalanceResponse, error) {
	year := int(req.GetYear())
	if year == 0 {
		year = time.Now().Year()
	}

	tenantID := getTenantID(ctx)

	allowances, err := s.allowanceRepo.GetByUserAndYear(ctx, tenantID, req.GetUserId(), year)
	if err != nil {
		return nil, err
	}

	entries := make([]*hrV1.BalanceEntry, len(allowances))
	for i, a := range allowances {
		entry := &hrV1.BalanceEntry{
			AbsenceTypeId: a.AbsenceTypeID,
			TotalDays:     a.TotalDays,
			UsedDays:      a.UsedDays,
			CarriedOver:   a.CarriedOver,
			RemainingDays: a.TotalDays + a.CarriedOver - a.UsedDays,
		}

		if a.Edges.AbsenceType != nil {
			entry.AbsenceTypeName = a.Edges.AbsenceType.Name
			entry.Color = a.Edges.AbsenceType.Color
		}

		entries[i] = entry
	}

	return &hrV1.GetUserBalanceResponse{
		UserId:  req.GetUserId(),
		Year:    int32(year),
		Entries: entries,
	}, nil
}

func allowanceToProto(e *ent.LeaveAllowance) *hrV1.LeaveAllowance {
	if e == nil {
		return nil
	}

	year := int32(e.Year)
	result := &hrV1.LeaveAllowance{
		Id:            &e.ID,
		TenantId:      e.TenantID,
		UserId:        &e.UserID,
		AbsenceTypeId: ptrString(e.AbsenceTypeID),
		Year:          &year,
		TotalDays:     ptrFloat64(e.TotalDays),
		UsedDays:      ptrFloat64(e.UsedDays),
		CarriedOver:   ptrFloat64(e.CarriedOver),
		Notes:         ptrString(e.Notes),
		UserName:      ptrString(e.UserName),
		CreatedBy:     e.CreateBy,
		UpdatedBy:     e.UpdateBy,
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
	}

	return result
}
