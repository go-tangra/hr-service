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
	poolRepo        *data.AllowancePoolRepo
}

func NewAllowanceService(ctx *bootstrap.Context, allowanceRepo *data.LeaveAllowanceRepo, absenceTypeRepo *data.AbsenceTypeRepo, poolRepo *data.AllowancePoolRepo) *AllowanceService {
	return &AllowanceService{
		log:             ctx.NewLoggerHelper("hr/service/allowance"),
		allowanceRepo:   allowanceRepo,
		absenceTypeRepo: absenceTypeRepo,
		poolRepo:        poolRepo,
	}
}

func (s *AllowanceService) CreateAllowance(ctx context.Context, req *hrV1.CreateAllowanceRequest) (*hrV1.CreateAllowanceResponse, error) {
	if err := checkPermission(ctx, "hr.allowance.manage"); err != nil {
		return nil, err
	}

	// Validate: individual allowance must not target a type that belongs to a pool
	absenceTypeID := req.GetAbsenceTypeId()
	isPoolBased := req.AllowancePoolId != nil && *req.AllowancePoolId != ""

	if !isPoolBased && absenceTypeID != "" {
		absType, err := s.absenceTypeRepo.GetByID(ctx, absenceTypeID)
		if err != nil {
			return nil, err
		}
		if absType == nil {
			return nil, hrV1.ErrorAbsenceTypeNotFound("absence type not found")
		}
		if absType.AllowancePoolID != "" {
			poolEntity, _ := s.poolRepo.GetByID(ctx, absType.AllowancePoolID)
			poolName := absType.AllowancePoolID
			if poolEntity != nil {
				poolName = poolEntity.Name
			}
			return nil, hrV1.ErrorBadRequest(
				"absence type %q belongs to pool %q — create a pool-based allowance instead",
				absType.Name, poolName,
			)
		}
	}

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
	if isPoolBased {
		poolID := *req.AllowancePoolId
		pool, err := s.poolRepo.GetByID(ctx, poolID)
		if err != nil {
			return nil, err
		}
		if pool == nil {
			return nil, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
		}
		opts = append(opts, func(c *ent.LeaveAllowanceCreate) { c.SetAllowancePoolID(poolID) })
	}

	entity, err := s.allowanceRepo.Create(ctx, getTenantID(ctx), req.GetUserId(), req.GetAbsenceTypeId(), int(req.GetYear()), req.GetTotalDays(), opts...)
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
	if err := checkPermission(ctx, "hr.allowance.view"); err != nil {
		return nil, err
	}

	entity, err := s.allowanceRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, hrV1.ErrorAllowanceNotFound("leave allowance not found")
	}
	if err := checkTenantAccess(ctx, entity.TenantID, hrV1.ErrorAllowanceNotFound("leave allowance not found")); err != nil {
		return nil, err
	}

	return &hrV1.GetAllowanceResponse{
		Allowance: allowanceToProto(entity),
	}, nil
}

func (s *AllowanceService) ListAllowances(ctx context.Context, req *hrV1.ListAllowancesRequest) (*hrV1.ListAllowancesResponse, error) {
	if err := checkPermission(ctx, "hr.allowance.view"); err != nil {
		return nil, err
	}

	filters := make(map[string]interface{})

	// Non-admin users can only see their own allowances
	if !hasPermission(ctx, "hr.allowance.manage") {
		filters["user_id"] = getUserID(ctx)
	} else if req.UserId != nil {
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

	entities, total, err := s.allowanceRepo.List(ctx, getTenantID(ctx), page, pageSize, filters)
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
	if err := checkPermission(ctx, "hr.allowance.manage"); err != nil {
		return nil, err
	}

	// Verify tenant access before updating
	existing, err := s.allowanceRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorAllowanceNotFound("leave allowance not found")
	}
	if err := checkTenantAccess(ctx, existing.TenantID, hrV1.ErrorAllowanceNotFound("leave allowance not found")); err != nil {
		return nil, err
	}

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
		if req.Data.UserName != nil {
			updates["user_name"] = *req.Data.UserName
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
	if err := checkPermission(ctx, "hr.allowance.manage"); err != nil {
		return nil, err
	}

	// Verify tenant access before deleting
	existing, err := s.allowanceRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorAllowanceNotFound("leave allowance not found")
	}
	if err := checkTenantAccess(ctx, existing.TenantID, hrV1.ErrorAllowanceNotFound("leave allowance not found")); err != nil {
		return nil, err
	}

	err = s.allowanceRepo.Delete(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AllowanceService) GetUserBalance(ctx context.Context, req *hrV1.GetUserBalanceRequest) (*hrV1.GetUserBalanceResponse, error) {
	if err := checkPermission(ctx, "hr.allowance.view"); err != nil {
		return nil, err
	}

	// Non-admin users can only view their own balance
	if !hasPermission(ctx, "hr.allowance.manage") && req.GetUserId() != getUserID(ctx) {
		return nil, hrV1.ErrorBadRequest("you can only view your own leave balance")
	}

	year := int(req.GetYear())
	if year == 0 {
		year = time.Now().Year()
	}

	tenantID := getTenantID(ctx)

	allowances, err := s.allowanceRepo.GetByUserAndYear(ctx, tenantID, req.GetUserId(), year)
	if err != nil {
		return nil, err
	}

	var entries []*hrV1.BalanceEntry
	for _, a := range allowances {
		entry := &hrV1.BalanceEntry{
			AbsenceTypeId: derefString(a.AbsenceTypeID),
			TotalDays:     a.TotalDays,
			UsedDays:      a.UsedDays,
			CarriedOver:   a.CarriedOver,
			RemainingDays: a.TotalDays + a.CarriedOver - a.UsedDays,
		}

		if a.Edges.AbsenceType != nil {
			entry.AbsenceTypeName = a.Edges.AbsenceType.Name
			entry.Color = a.Edges.AbsenceType.Color
		}

		// Pool-based allowance
		poolID := derefString(a.AllowancePoolID)
		if poolID != "" {
			entry.AllowancePoolId = ptrString(poolID)
			// Fetch pool details for display name and member types
			pool, poolErr := s.poolRepo.GetByID(ctx, poolID)
			if poolErr == nil && pool != nil {
				entry.AllowancePoolName = ptrString(pool.Name)
				if pool.Color != "" {
					entry.Color = pool.Color
				}
				for _, at := range pool.Edges.AbsenceTypes {
					entry.MemberAbsenceTypeIds = append(entry.MemberAbsenceTypeIds, at.ID)
				}
			}
		}

		entries = append(entries, entry)
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
		Id:              &e.ID,
		TenantId:        e.TenantID,
		UserId:          &e.UserID,
		AbsenceTypeId:   e.AbsenceTypeID,
		AllowancePoolId: e.AllowancePoolID,
		Year:            &year,
		TotalDays:       ptrFloat64(e.TotalDays),
		UsedDays:        ptrFloat64(e.UsedDays),
		CarriedOver:     ptrFloat64(e.CarriedOver),
		Notes:           ptrString(e.Notes),
		UserName:        ptrString(e.UserName),
		CreatedBy:       e.CreateBy,
		UpdatedBy:       e.UpdateBy,
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
