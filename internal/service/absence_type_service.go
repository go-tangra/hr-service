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

type AbsenceTypeService struct {
	hrV1.UnimplementedHrAbsenceTypeServiceServer

	log             *log.Helper
	absenceTypeRepo *data.AbsenceTypeRepo
}

func NewAbsenceTypeService(ctx *bootstrap.Context, absenceTypeRepo *data.AbsenceTypeRepo) *AbsenceTypeService {
	return &AbsenceTypeService{
		log:             ctx.NewLoggerHelper("hr/service/absence_type"),
		absenceTypeRepo: absenceTypeRepo,
	}
}

func (s *AbsenceTypeService) CreateAbsenceType(ctx context.Context, req *hrV1.CreateAbsenceTypeRequest) (*hrV1.CreateAbsenceTypeResponse, error) {
	opts := []func(*ent.AbsenceTypeCreate){}

	if req.Description != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetDescription(*req.Description) })
	}
	if req.Color != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetColor(*req.Color) })
	}
	if req.Icon != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetIcon(*req.Icon) })
	}
	if req.DeductsFromAllowance != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetDeductsFromAllowance(*req.DeductsFromAllowance) })
	}
	if req.RequiresApproval != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetRequiresApproval(*req.RequiresApproval) })
	}
	if req.IsActive != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetIsActive(*req.IsActive) })
	}
	if req.SortOrder != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetSortOrder(int(*req.SortOrder)) })
	}
	if req.Metadata != nil {
		opts = append(opts, func(c *ent.AbsenceTypeCreate) { c.SetMetadata(req.Metadata.AsMap()) })
	}

	entity, err := s.absenceTypeRepo.Create(ctx, req.GetTenantId(), req.GetName(), opts...)
	if err != nil {
		return nil, err
	}

	return &hrV1.CreateAbsenceTypeResponse{
		AbsenceType: absenceTypeToProto(entity),
	}, nil
}

func (s *AbsenceTypeService) GetAbsenceType(ctx context.Context, req *hrV1.GetAbsenceTypeRequest) (*hrV1.GetAbsenceTypeResponse, error) {
	entity, err := s.absenceTypeRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, hrV1.ErrorAbsenceTypeNotFound("absence type not found")
	}

	return &hrV1.GetAbsenceTypeResponse{
		AbsenceType: absenceTypeToProto(entity),
	}, nil
}

func (s *AbsenceTypeService) ListAbsenceTypes(ctx context.Context, req *hrV1.ListAbsenceTypesRequest) (*hrV1.ListAbsenceTypesResponse, error) {
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

	entities, total, err := s.absenceTypeRepo.List(ctx, req.GetTenantId(), page, pageSize, filters)
	if err != nil {
		return nil, err
	}

	items := make([]*hrV1.AbsenceType, len(entities))
	for i, e := range entities {
		items[i] = absenceTypeToProto(e)
	}

	return &hrV1.ListAbsenceTypesResponse{
		Items: items,
		Total: ptrInt32(int32(total)),
	}, nil
}

func (s *AbsenceTypeService) UpdateAbsenceType(ctx context.Context, req *hrV1.UpdateAbsenceTypeRequest) (*hrV1.UpdateAbsenceTypeResponse, error) {
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
		if req.Data.DeductsFromAllowance != nil {
			updates["deducts_from_allowance"] = *req.Data.DeductsFromAllowance
		}
		if req.Data.RequiresApproval != nil {
			updates["requires_approval"] = *req.Data.RequiresApproval
		}
		if req.Data.IsActive != nil {
			updates["is_active"] = *req.Data.IsActive
		}
		if req.Data.SortOrder != nil {
			updates["sort_order"] = int(*req.Data.SortOrder)
		}
		if req.Data.Metadata != nil {
			updates["metadata"] = req.Data.Metadata.AsMap()
		}
	}

	entity, err := s.absenceTypeRepo.Update(ctx, req.GetId(), updates)
	if err != nil {
		return nil, err
	}

	return &hrV1.UpdateAbsenceTypeResponse{
		AbsenceType: absenceTypeToProto(entity),
	}, nil
}

func (s *AbsenceTypeService) DeleteAbsenceType(ctx context.Context, req *hrV1.DeleteAbsenceTypeRequest) (*emptypb.Empty, error) {
	err := s.absenceTypeRepo.Delete(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func absenceTypeToProto(e *ent.AbsenceType) *hrV1.AbsenceType {
	if e == nil {
		return nil
	}

	sortOrder := int32(e.SortOrder)
	result := &hrV1.AbsenceType{
		Id:                    &e.ID,
		TenantId:              e.TenantID,
		Name:                  ptrString(e.Name),
		Description:           ptrString(e.Description),
		Color:                 ptrString(e.Color),
		Icon:                  ptrString(e.Icon),
		DeductsFromAllowance:  ptrBool(e.DeductsFromAllowance),
		RequiresApproval:      ptrBool(e.RequiresApproval),
		IsActive:              ptrBool(e.IsActive),
		SortOrder:             &sortOrder,
		Metadata:              mapToStruct(e.Metadata),
		CreatedBy:             e.CreateBy,
		UpdatedBy:             e.UpdateBy,
	}

	if e.CreateTime != nil {
		result.CreatedAt = timestamppb.New(*e.CreateTime)
	}
	if e.UpdateTime != nil {
		result.UpdatedAt = timestamppb.New(*e.UpdateTime)
	}

	return result
}
