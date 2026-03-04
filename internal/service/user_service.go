package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/internal/client"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

type UserService struct {
	hrV1.UnimplementedHrUserServiceServer

	log         *log.Helper
	adminClient *client.AdminClient
}

func NewUserService(ctx *bootstrap.Context, adminClient *client.AdminClient) *UserService {
	return &UserService{
		log:         ctx.NewLoggerHelper("hr/service/user"),
		adminClient: adminClient,
	}
}

func (s *UserService) ListUsers(ctx context.Context, req *hrV1.ListHrUsersRequest) (*hrV1.ListHrUsersResponse, error) {
	tenantID := getTenantID(ctx)

	resp, err := s.adminClient.ListUsers(ctx, tenantID)
	if err != nil {
		s.log.Errorf("Failed to list users from admin-service: %v", err)
		return nil, err
	}

	items := make([]*hrV1.HrUser, 0, len(resp.GetItems()))
	for _, u := range resp.GetItems() {
		items = append(items, &hrV1.HrUser{
			Id:            u.GetId(),
			Username:      u.GetUsername(),
			Realname:      u.GetRealname(),
			Email:         u.GetEmail(),
			OrgUnitNames:  u.GetOrgUnitNames(),
			PositionNames: u.GetPositionNames(),
		})
	}

	return &hrV1.ListHrUsersResponse{
		Items: items,
		Total: int32(len(items)),
	}, nil
}
