package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"

	"github.com/go-tangra/go-tangra-common/grpcx"

	paperlesspb "buf.build/gen/go/go-tangra/paperless/protocolbuffers/go/paperless/service/v1"
	paperlessgrpc "buf.build/gen/go/go-tangra/paperless/grpc/go/paperless/service/v1/servicev1grpc"
)

// PaperlessClient wraps the gRPC connection to the paperless signing service.
// It resolves the paperless endpoint lazily via ModuleDialer on first use.
type PaperlessClient struct {
	dialer *grpcx.ModuleDialer
	log    *log.Helper

	mu     sync.Mutex
	conn   *grpc.ClientConn
	client paperlessgrpc.PaperlessSigningRequestServiceClient
}

// NewPaperlessClient creates a new Paperless gRPC client that resolves via ModuleDialer.
func NewPaperlessClient(ctx *bootstrap.Context, dialer *grpcx.ModuleDialer) (*PaperlessClient, func(), error) {
	l := ctx.NewLoggerHelper("hr/client/paperless")

	c := &PaperlessClient{
		dialer: dialer,
		log:    l,
	}

	cleanup := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				l.Errorf("Failed to close Paperless connection: %v", err)
			}
		}
	}

	l.Info("Paperless client created (will resolve endpoint on first use)")
	return c, cleanup, nil
}

// resolve lazily connects to the paperless service via ModuleDialer.
// Uses a mutex instead of sync.Once so that transient failures can be retried.
func (c *PaperlessClient) resolve() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		return nil // already connected
	}

	c.log.Info("Resolving paperless module endpoint...")
	conn, err := c.dialer.DialModule(context.Background(), "paperless", 30, 5*time.Second)
	if err != nil {
		c.log.Errorf("Failed to resolve paperless: %v", err)
		return fmt.Errorf("resolve paperless: %w", err)
	}
	c.conn = conn
	c.client = paperlessgrpc.NewPaperlessSigningRequestServiceClient(conn)
	c.log.Info("Paperless client connected via ModuleDialer")
	return nil
}

// CreateSigningRequest creates a signing request in paperless
func (c *PaperlessClient) CreateSigningRequest(
	ctx context.Context,
	templateID string,
	name string,
	recipients []*paperlesspb.SigningRecipientInput,
	fieldValues []*paperlesspb.SigningFieldValueInput,
	message string,
	signingType paperlesspb.SigningRequestType,
) (string, error) {
	if err := c.resolve(); err != nil {
		return "", err
	}

	resp, err := c.client.CreateSigningRequest(ctx, &paperlesspb.CreateSigningRequestRequest{
		TemplateId:  templateID,
		Name:        name,
		Recipients:  recipients,
		FieldValues: fieldValues,
		Message:     message,
		SigningType:  signingType,
	})
	if err != nil {
		c.log.Errorf("Failed to create signing request: %v", err)
		return "", err
	}

	if resp.Request == nil {
		return "", fmt.Errorf("paperless service returned nil signing request")
	}

	c.log.Infof("Created signing request: %s", resp.Request.Id)
	return resp.Request.Id, nil
}

// RevokeSigningRequest revokes a completed signing request in paperless
func (c *PaperlessClient) RevokeSigningRequest(ctx context.Context, signingRequestID string, reason string) error {
	if err := c.resolve(); err != nil {
		return err
	}

	_, err := c.client.RevokeSigningRequest(ctx, &paperlesspb.RevokeSigningRequestRequest{
		Id:     signingRequestID,
		Reason: reason,
	})
	if err != nil {
		c.log.Errorf("Failed to revoke signing request %s: %v", signingRequestID, err)
		return err
	}

	c.log.Infof("Revoked signing request: %s", signingRequestID)
	return nil
}

// DownloadSignedDocument returns a presigned download URL for a signed document
func (c *PaperlessClient) DownloadSignedDocument(ctx context.Context, signingRequestID string) (string, error) {
	if err := c.resolve(); err != nil {
		return "", err
	}

	resp, err := c.client.DownloadSignedDocument(ctx, &paperlesspb.DownloadSignedDocumentRequest{
		Id: signingRequestID,
	})
	if err != nil {
		c.log.Errorf("Failed to download signed document: %v", err)
		return "", err
	}
	return resp.GetUrl(), nil
}
