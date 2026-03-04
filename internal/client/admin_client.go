package client

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	adminV1 "github.com/go-tangra/go-tangra-portal/api/gen/go/admin/service/v1"
	userV1 "github.com/go-tangra/go-tangra-portal/api/gen/go/user/service/v1"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
)

// AdminClient calls the admin-service gRPC API for user listing
type AdminClient struct {
	log    *log.Helper
	conn   *grpc.ClientConn
	client adminV1.UserServiceClient
}

// NewAdminClient creates a new AdminClient
func NewAdminClient(ctx *bootstrap.Context) (*AdminClient, func(), error) {
	l := ctx.NewLoggerHelper("hr/client/admin")

	endpoint := os.Getenv("ADMIN_GRPC_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:7787"
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := adminV1.NewUserServiceClient(conn)

	cleanup := func() {
		if conn != nil {
			conn.Close()
		}
	}

	l.Infof("Admin gRPC client configured for endpoint: %s", endpoint)

	return &AdminClient{
		log:    l,
		conn:   conn,
		client: client,
	}, cleanup, nil
}

// ListUsers calls the admin-service gRPC API to list users
func (c *AdminClient) ListUsers(ctx context.Context) (*userV1.ListUserResponse, error) {
	noPaging := true
	resp, err := c.client.List(ctx, &paginationV1.PagingRequest{
		NoPaging: &noPaging,
	})
	if err != nil {
		c.log.Errorf("Failed to list users from admin-service: %v", err)
		return nil, err
	}

	return resp, nil
}
