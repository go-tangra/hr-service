package client

import (
	"context"
	"os"
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	userpb "github.com/go-tangra/go-tangra-portal/api/gen/go/user/service/v1"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
)

// AdminClient wraps the gRPC connection to the admin-service for user listing
type AdminClient struct {
	log        *log.Helper
	conn       *grpc.ClientConn
	userClient userpb.UserServiceClient
}

// NewAdminClient creates a new AdminClient using insecure credentials (internal network)
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

	userClient := userpb.NewUserServiceClient(conn)

	cleanup := func() {
		if conn != nil {
			conn.Close()
		}
	}

	l.Infof("Admin gRPC client configured for endpoint: %s", endpoint)

	return &AdminClient{
		log:        l,
		conn:       conn,
		userClient: userClient,
	}, cleanup, nil
}

// ListUsers calls the admin-service to list users, forwarding tenant context
func (c *AdminClient) ListUsers(ctx context.Context, tenantID uint32) (*userpb.ListUserResponse, error) {
	// Forward tenant context as gRPC metadata
	md := metadata.New(map[string]string{
		"x-md-global-tenant-id": strconv.FormatUint(uint64(tenantID), 10),
	})
	outCtx := metadata.NewOutgoingContext(ctx, md)

	noPaging := true
	req := &pagination.PagingRequest{
		NoPaging: &noPaging,
	}

	resp, err := c.userClient.List(outCtx, req)
	if err != nil {
		c.log.Errorf("Failed to list users from admin-service: %v", err)
		return nil, err
	}

	return resp, nil
}
