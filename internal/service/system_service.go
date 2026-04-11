package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/client"
	"github.com/go-tangra/go-tangra-hr/internal/data"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

var version = "1.0.0"

type SystemService struct {
	hrV1.UnimplementedHrSystemServiceServer

	log              *log.Helper
	absenceTypeRepo  *data.AbsenceTypeRepo
	leaveRequestRepo *data.LeaveRequestRepo
	signingClient    *client.SigningClient
}

func NewSystemService(ctx *bootstrap.Context, absenceTypeRepo *data.AbsenceTypeRepo, leaveRequestRepo *data.LeaveRequestRepo, signingClient *client.SigningClient) *SystemService {
	return &SystemService{
		log:              ctx.NewLoggerHelper("hr/service/system"),
		absenceTypeRepo:  absenceTypeRepo,
		leaveRequestRepo: leaveRequestRepo,
		signingClient:    signingClient,
	}
}

func (s *SystemService) HealthCheck(ctx context.Context, req *hrV1.HealthCheckRequest) (*hrV1.HealthCheckResponse, error) {
	return &hrV1.HealthCheckResponse{
		Status:    "healthy",
		Version:   version,
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

func (s *SystemService) GetStats(ctx context.Context, req *hrV1.GetStatsRequest) (*hrV1.GetStatsResponse, error) {
	if err := checkPermission(ctx, "hr.request.view"); err != nil {
		return nil, err
	}

	tenantID := getTenantID(ctx)
	response := &hrV1.GetStatsResponse{}

	if count, err := s.absenceTypeRepo.Count(ctx, tenantID); err == nil {
		response.TotalAbsenceTypes = int64(count)
	}
	if count, err := s.leaveRequestRepo.CountByStatus(ctx, tenantID, "pending"); err == nil {
		response.PendingRequests = int64(count)
	}
	if count, err := s.leaveRequestRepo.CountByStatus(ctx, tenantID, "approved"); err == nil {
		response.ApprovedRequests = int64(count)
	}
	if count, err := s.leaveRequestRepo.CountByStatus(ctx, tenantID, "rejected"); err == nil {
		response.RejectedRequests = int64(count)
	}
	if count, err := s.leaveRequestRepo.Count(ctx, tenantID); err == nil {
		response.TotalRequests = int64(count)
	}

	return response, nil
}

func (s *SystemService) ListSigningTemplates(ctx context.Context, req *hrV1.ListSigningTemplatesRequest) (*hrV1.ListSigningTemplatesResponse, error) {
	templates, err := s.signingClient.ListTemplates(ctx)
	if err != nil {
		s.log.Errorf("Failed to list signing templates: %v", err)
		return nil, err
	}

	result := make([]*hrV1.SigningTemplate, 0, len(templates))
	for _, t := range templates {
		result = append(result, &hrV1.SigningTemplate{
			Id:     t.GetId(),
			Name:   t.GetName(),
			Status: t.GetStatus(),
		})
	}

	return &hrV1.ListSigningTemplatesResponse{Templates: result}, nil
}
