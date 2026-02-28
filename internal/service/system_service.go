package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-tangra/go-tangra-hr/internal/data"
	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
)

var version = "1.0.0"

type SystemService struct {
	hrV1.UnimplementedHrSystemServiceServer

	log              *log.Helper
	absenceTypeRepo  *data.AbsenceTypeRepo
	leaveRequestRepo *data.LeaveRequestRepo
}

func NewSystemService(ctx *bootstrap.Context, absenceTypeRepo *data.AbsenceTypeRepo, leaveRequestRepo *data.LeaveRequestRepo) *SystemService {
	return &SystemService{
		log:              ctx.NewLoggerHelper("hr/service/system"),
		absenceTypeRepo:  absenceTypeRepo,
		leaveRequestRepo: leaveRequestRepo,
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
	tenantID := req.GetTenantId()
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
