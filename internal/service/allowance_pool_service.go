package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/data"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type AllowancePoolService struct {
	hrV1.UnimplementedHrAllowancePoolServiceServer

	log             *log.Helper
	poolRepo        *data.AllowancePoolRepo
	absenceTypeRepo *data.AbsenceTypeRepo
}

func NewAllowancePoolService(ctx *bootstrap.Context, poolRepo *data.AllowancePoolRepo, absenceTypeRepo *data.AbsenceTypeRepo) *AllowancePoolService {
	return &AllowancePoolService{
		log:             ctx.NewLoggerHelper("hr/service/allowance_pool"),
		poolRepo:        poolRepo,
		absenceTypeRepo: absenceTypeRepo,
	}
}

func (s *AllowancePoolService) CreateAllowancePool(ctx context.Context, req *hrV1.CreateAllowancePoolRequest) (*hrV1.CreateAllowancePoolResponse, error) {
	if err := checkPermission(ctx, "hr.allowance_pool.manage"); err != nil {
		return nil, err
	}

	tenantID := getTenantID(ctx)

	opts := []func(*ent.AllowancePoolCreate){}
	if req.Description != nil {
		opts = append(opts, func(c *ent.AllowancePoolCreate) { c.SetDescription(*req.Description) })
	}
	if req.Color != nil {
		opts = append(opts, func(c *ent.AllowancePoolCreate) { c.SetColor(*req.Color) })
	}
	if req.Icon != nil {
		opts = append(opts, func(c *ent.AllowancePoolCreate) { c.SetIcon(*req.Icon) })
	}

	// If absence type IDs are provided, link them via the edge
	if len(req.AbsenceTypeIds) > 0 {
		opts = append(opts, func(c *ent.AllowancePoolCreate) {
			c.AddAbsenceTypeIDs(req.AbsenceTypeIds...)
		})
	}

	entity, err := s.poolRepo.Create(ctx, tenantID, req.GetName(), opts...)
	if err != nil {
		return nil, err
	}

	// If we assigned absence types, also set allowance_pool_id on them
	if len(req.AbsenceTypeIds) > 0 {
		for _, typeID := range req.AbsenceTypeIds {
			if _, err := s.absenceTypeRepo.Update(ctx, typeID, map[string]interface{}{
				"allowance_pool_id": entity.ID,
			}); err != nil {
				s.log.Errorf("Failed to set allowance_pool_id on absence type %s: %v", typeID, err)
			}
		}
	}

	// Re-fetch with edges
	entity, _ = s.poolRepo.GetByID(ctx, entity.ID)

	return &hrV1.CreateAllowancePoolResponse{
		Pool: allowancePoolToProto(entity),
	}, nil
}

func (s *AllowancePoolService) GetAllowancePool(ctx context.Context, req *hrV1.GetAllowancePoolRequest) (*hrV1.GetAllowancePoolResponse, error) {
	if err := checkPermission(ctx, "hr.allowance.view"); err != nil {
		return nil, err
	}

	entity, err := s.poolRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
	}
	if err := checkTenantAccess(ctx, entity.TenantID, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")); err != nil {
		return nil, err
	}

	return &hrV1.GetAllowancePoolResponse{
		Pool: allowancePoolToProto(entity),
	}, nil
}

func (s *AllowancePoolService) ListAllowancePools(ctx context.Context, req *hrV1.ListAllowancePoolsRequest) (*hrV1.ListAllowancePoolsResponse, error) {
	if err := checkPermission(ctx, "hr.allowance.view"); err != nil {
		return nil, err
	}

	filters := make(map[string]interface{})
	if req.Query != nil {
		filters["query"] = *req.Query
	}

	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if req.GetNoPaging() {
		page = 0
		pageSize = 0
	}

	entities, total, err := s.poolRepo.List(ctx, getTenantID(ctx), page, pageSize, filters)
	if err != nil {
		return nil, err
	}

	items := make([]*hrV1.AllowancePool, len(entities))
	for i, e := range entities {
		items[i] = allowancePoolToProto(e)
	}

	return &hrV1.ListAllowancePoolsResponse{
		Items: items,
		Total: ptrInt32(int32(total)),
	}, nil
}

func (s *AllowancePoolService) UpdateAllowancePool(ctx context.Context, req *hrV1.UpdateAllowancePoolRequest) (*hrV1.UpdateAllowancePoolResponse, error) {
	if err := checkPermission(ctx, "hr.allowance_pool.manage"); err != nil {
		return nil, err
	}

	existing, err := s.poolRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
	}
	if err := checkTenantAccess(ctx, existing.TenantID, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Data != nil {
		if req.Data.Name != nil {
			updates["name"] = *req.Data.Name
		}
		if req.Data.Description != nil {
			updates["description"] = *req.Data.Description
		}
		if req.Data.Color != nil {
			updates["color"] = *req.Data.Color
		}
		if req.Data.Icon != nil {
			updates["icon"] = *req.Data.Icon
		}
	}

	entity, err := s.poolRepo.Update(ctx, req.GetId(), updates)
	if err != nil {
		return nil, err
	}

	// Re-fetch with edges
	entity2, _ := s.poolRepo.GetByID(ctx, entity.ID)
	if entity2 != nil {
		entity = entity2
	}

	return &hrV1.UpdateAllowancePoolResponse{
		Pool: allowancePoolToProto(entity),
	}, nil
}

func (s *AllowancePoolService) DeleteAllowancePool(ctx context.Context, req *hrV1.DeleteAllowancePoolRequest) (*emptypb.Empty, error) {
	if err := checkPermission(ctx, "hr.allowance_pool.manage"); err != nil {
		return nil, err
	}

	existing, err := s.poolRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")
	}
	if err := checkTenantAccess(ctx, existing.TenantID, hrV1.ErrorAllowancePoolNotFound("allowance pool not found")); err != nil {
		return nil, err
	}

	err = s.poolRepo.Delete(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func allowancePoolToProto(e *ent.AllowancePool) *hrV1.AllowancePool {
	if e == nil {
		return nil
	}

	result := &hrV1.AllowancePool{
		Id:          &e.ID,
		TenantId:    e.TenantID,
		Name:        ptrString(e.Name),
		Description: ptrString(e.Description),
		Color:       ptrString(e.Color),
		Icon:        ptrString(e.Icon),
		CreatedBy:   e.CreateBy,
		UpdatedBy:   e.UpdateBy,
	}

	if e.CreateTime != nil {
		result.CreatedAt = timestamppb.New(*e.CreateTime)
	}
	if e.UpdateTime != nil {
		result.UpdatedAt = timestamppb.New(*e.UpdateTime)
	}

	// Populate member absence type IDs from edge
	for _, at := range e.Edges.AbsenceTypes {
		result.AbsenceTypeIds = append(result.AbsenceTypeIds, at.ID)
	}

	return result
}
