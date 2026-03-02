package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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

	creds, err := loadPaperlessClientTLSCredentials(l)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load TLS credentials for paperless client: %w", err)
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(creds),
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

	l.Infof("Paperless gRPC client configured for endpoint: %s (mTLS enabled)", endpoint)

	return &PaperlessClient{
		log:    l,
		conn:   conn,
		client: client,
	}, cleanup, nil
}

// loadPaperlessClientTLSCredentials loads mTLS credentials for connecting to paperless service
func loadPaperlessClientTLSCredentials(l *log.Helper) (credentials.TransportCredentials, error) {
	caCertPath := os.Getenv("HR_CA_CERT_PATH")
	if caCertPath == "" {
		caCertPath = "/app/certs/ca/ca.crt"
	}
	clientCertPath := os.Getenv("HR_CLIENT_CERT_PATH")
	if clientCertPath == "" {
		clientCertPath = "/app/certs/hr/client.crt"
	}
	clientKeyPath := os.Getenv("HR_CLIENT_KEY_PATH")
	if clientKeyPath == "" {
		clientKeyPath = "/app/certs/hr/client.key"
	}

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		l.Errorf("Failed to read CA cert from %s: %v", caCertPath, err)
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caCertPath)
	}

	clientCert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		l.Errorf("Failed to load client cert/key from %s, %s: %v", clientCertPath, clientKeyPath, err)
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		ServerName:   "localhost",
		MinVersion:   tls.VersionTLS12,
	}

	l.Infof("Loaded TLS credentials for paperless client: CA=%s, Cert=%s", caCertPath, clientCertPath)

	return credentials.NewTLS(tlsConfig), nil
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
