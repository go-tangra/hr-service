package client

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	paperlesspb "github.com/go-tangra/go-tangra-paperless/gen/go/paperless/service/v1"
)

// PaperlessClient wraps the gRPC connection to the paperless signing service
type PaperlessClient struct {
	log    *log.Helper
	conn   *grpc.ClientConn
	client paperlesspb.PaperlessSigningRequestServiceClient
}

// NewPaperlessClient creates a new PaperlessClient
func NewPaperlessClient(ctx *bootstrap.Context) (*PaperlessClient, func(), error) {
	l := ctx.NewLoggerHelper("hr/client/paperless")

	endpoint := os.Getenv("PAPERLESS_GRPC_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9500"
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := paperlesspb.NewPaperlessSigningRequestServiceClient(conn)

	cleanup := func() {
		if conn != nil {
			conn.Close()
		}
	}

	l.Infof("Paperless gRPC client configured for endpoint: %s", endpoint)

	return &PaperlessClient{
		log:    l,
		conn:   conn,
		client: client,
	}, cleanup, nil
}

// CreateSigningRequest creates a signing request in paperless
func (c *PaperlessClient) CreateSigningRequest(
	ctx context.Context,
	templateID string,
	name string,
	recipients []*paperlesspb.SigningRecipientInput,
	fieldValues []*paperlesspb.SigningFieldValueInput,
	message string,
) (string, error) {
	resp, err := c.client.CreateSigningRequest(ctx, &paperlesspb.CreateSigningRequestRequest{
		TemplateId:  templateID,
		Name:        name,
		Recipients:  recipients,
		FieldValues: fieldValues,
		Message:     message,
	})
	if err != nil {
		c.log.Errorf("Failed to create signing request: %v", err)
		return "", err
	}

	if resp.Request == nil {
		return "", nil
	}

	c.log.Infof("Created signing request: %s", resp.Request.Id)
	return resp.Request.Id, nil
}
