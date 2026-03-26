package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"

	"github.com/go-tangra/go-tangra-common/grpcx"

	signingpb "buf.build/gen/go/go-tangra/signing/protocolbuffers/go/signing/service/v1"
	signinggrpc "buf.build/gen/go/go-tangra/signing/grpc/go/signing/service/v1/servicev1grpc"
)

// SigningClient wraps the gRPC connection to the signing service.
// It resolves the signing endpoint lazily via ModuleDialer on first use.
type SigningClient struct {
	dialer *grpcx.ModuleDialer
	log    *log.Helper

	mu         sync.Mutex
	conn       *grpc.ClientConn
	submission signinggrpc.SigningSubmissionServiceClient
}

// NewSigningClient creates a new Signing gRPC client that resolves via ModuleDialer.
func NewSigningClient(ctx *bootstrap.Context, dialer *grpcx.ModuleDialer) (*SigningClient, func(), error) {
	l := ctx.NewLoggerHelper("hr/client/signing")

	c := &SigningClient{
		dialer: dialer,
		log:    l,
	}

	cleanup := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				l.Errorf("Failed to close Signing connection: %v", err)
			}
		}
	}

	l.Info("Signing client created (will resolve endpoint on first use)")
	return c, cleanup, nil
}

// resolve lazily connects to the signing service via ModuleDialer.
func (c *SigningClient) resolve() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.submission != nil {
		return nil
	}

	c.log.Info("Resolving signing module endpoint...")
	conn, err := c.dialer.DialModule(context.Background(), "signing", 30, 5*time.Second)
	if err != nil {
		c.log.Errorf("Failed to resolve signing: %v", err)
		return fmt.Errorf("resolve signing: %w", err)
	}
	c.conn = conn
	c.submission = signinggrpc.NewSigningSubmissionServiceClient(conn)
	c.log.Info("Signing client connected via ModuleDialer")
	return nil
}

// SubmitterInput defines a signer for a submission.
type SubmitterInput struct {
	Name  string
	Email string
	Phone string
	Role  string
}

// CreateAndSendSubmission creates a submission from a template and immediately sends it.
// Returns the submission ID.
func (c *SigningClient) CreateAndSendSubmission(
	ctx context.Context,
	templateID string,
	signingMode string,
	source string,
	submitters []SubmitterInput,
	prefillValues map[string]string,
) (string, error) {
	if err := c.resolve(); err != nil {
		return "", err
	}

	// Convert to proto submitters
	protoSubmitters := make([]*signingpb.SubmitterInput, len(submitters))
	for i, s := range submitters {
		protoSubmitters[i] = &signingpb.SubmitterInput{
			Name:  s.Name,
			Email: s.Email,
			Phone: s.Phone,
			Role:  s.Role,
		}
	}

	// Create submission
	createResp, err := c.submission.CreateSubmission(ctx, &signingpb.CreateSubmissionRequest{
		TemplateId:    templateID,
		SigningMode:   signingMode,
		Source:        source,
		Submitters:    protoSubmitters,
		PrefillValues: prefillValues,
	})
	if err != nil {
		c.log.Errorf("Failed to create submission: %v", err)
		return "", fmt.Errorf("create submission: %w", err)
	}

	if createResp.Submission == nil {
		return "", fmt.Errorf("signing service returned nil submission")
	}

	submissionID := createResp.Submission.Id
	c.log.Infof("Created signing submission: %s", submissionID)

	// Send submission (triggers email invitations)
	_, err = c.submission.SendSubmission(ctx, &signingpb.SendSubmissionRequest{
		Id: submissionID,
	})
	if err != nil {
		c.log.Errorf("Failed to send submission %s: %v", submissionID, err)
		return submissionID, fmt.Errorf("send submission: %w", err)
	}

	c.log.Infof("Sent signing submission: %s", submissionID)
	return submissionID, nil
}

// CancelSubmission cancels an in-progress submission.
func (c *SigningClient) CancelSubmission(ctx context.Context, submissionID, reason string) error {
	if err := c.resolve(); err != nil {
		return err
	}

	_, err := c.submission.CancelSubmission(ctx, &signingpb.CancelSubmissionRequest{
		Id:     submissionID,
		Reason: reason,
	})
	if err != nil {
		c.log.Errorf("Failed to cancel submission %s: %v", submissionID, err)
		return err
	}

	c.log.Infof("Cancelled signing submission: %s", submissionID)
	return nil
}

// DownloadSignedDocument fetches the signed PDF bytes from the signing service.
// Gets the storage key via gRPC, then downloads the PDF from signing service HTTP proxy.
func (c *SigningClient) DownloadSignedDocument(ctx context.Context, submissionID string) ([]byte, error) {
	if err := c.resolve(); err != nil {
		return nil, err
	}

	resp, err := c.submission.GetSubmissionDocumentUrl(ctx, &signingpb.GetSubmissionDocumentUrlRequest{
		Id: submissionID,
	})
	if err != nil {
		c.log.Errorf("Failed to get signed document key: %v", err)
		return nil, err
	}

	storageKey := resp.GetUrl()
	if storageKey == "" {
		return nil, fmt.Errorf("signing service returned empty storage key")
	}

	signingHTTP := os.Getenv("SIGNING_HTTP_ENDPOINT")
	if signingHTTP == "" {
		signingHTTP = "http://signing-service:10401"
	}

	pdfURL := signingHTTP + "/api/v1/signing/templates/pdf?key=" + url.QueryEscape(storageKey)
	httpResp, err := http.Get(pdfURL)
	if err != nil {
		c.log.Errorf("Failed to download signed PDF: %v", err)
		return nil, fmt.Errorf("download signed PDF: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("signing service returned HTTP %d", httpResp.StatusCode)
	}

	return io.ReadAll(httpResp.Body)
}
