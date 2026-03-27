package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/go-tangra/go-tangra-common/backup"
	"github.com/go-tangra/go-tangra-common/grpcx"

	hrV1 "github.com/go-tangra/go-tangra-hr/gen/go/hr/service/v1"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/absencetype"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/allowancepool"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/leaveallowance"
	"github.com/go-tangra/go-tangra-hr/internal/data/ent/leaverequest"
)

const (
	backupModule        = "hr"
	backupSchemaVersion = 1
)

var backupMigrations = backup.NewMigrationRegistry(backupModule)

type BackupService struct {
	hrV1.UnimplementedBackupServiceServer

	log       *log.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewBackupService(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *BackupService {
	return &BackupService{
		log:       ctx.NewLoggerHelper("hr/service/backup"),
		entClient: entClient,
	}
}

func (s *BackupService) ExportBackup(ctx context.Context, req *hrV1.ExportBackupRequest) (*hrV1.ExportBackupResponse, error) {
	tenantID := grpcx.GetTenantIDFromContext(ctx)
	full := false

	if grpcx.IsPlatformAdmin(ctx) && req.TenantId != nil && *req.TenantId == 0 {
		full = true
		tenantID = 0
	} else if req.TenantId != nil && *req.TenantId != 0 && grpcx.IsPlatformAdmin(ctx) {
		tenantID = *req.TenantId
	}

	client := s.entClient.Client()
	a := backup.NewArchive(backupModule, backupSchemaVersion, tenantID, full)

	// Export absence types
	atQuery := client.AbsenceType.Query()
	if !full {
		atQuery = atQuery.Where(absencetype.TenantIDEQ(tenantID))
	}
	absenceTypes, err := atQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("export absence types: %w", err)
	}
	if err := backup.SetEntities(a, "absenceTypes", absenceTypes); err != nil {
		return nil, fmt.Errorf("set absence types: %w", err)
	}

	// Export allowance pools
	apQuery := client.AllowancePool.Query()
	if !full {
		apQuery = apQuery.Where(allowancepool.TenantIDEQ(tenantID))
	}
	pools, err := apQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("export allowance pools: %w", err)
	}
	if err := backup.SetEntities(a, "allowancePools", pools); err != nil {
		return nil, fmt.Errorf("set allowance pools: %w", err)
	}

	// Export leave allowances
	laQuery := client.LeaveAllowance.Query()
	if !full {
		laQuery = laQuery.Where(leaveallowance.TenantIDEQ(tenantID))
	}
	allowances, err := laQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("export leave allowances: %w", err)
	}
	if err := backup.SetEntities(a, "leaveAllowances", allowances); err != nil {
		return nil, fmt.Errorf("set leave allowances: %w", err)
	}

	// Export leave requests
	lrQuery := client.LeaveRequest.Query()
	if !full {
		lrQuery = lrQuery.Where(leaverequest.TenantIDEQ(tenantID))
	}
	requests, err := lrQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("export leave requests: %w", err)
	}
	if err := backup.SetEntities(a, "leaveRequests", requests); err != nil {
		return nil, fmt.Errorf("set leave requests: %w", err)
	}

	data, err := backup.Pack(a)
	if err != nil {
		return nil, fmt.Errorf("pack backup: %w", err)
	}

	s.log.Infof("exported backup: module=%s tenant=%d full=%v entities=%v", backupModule, tenantID, full, a.Manifest.EntityCounts)

	return &hrV1.ExportBackupResponse{
		Data:          data,
		Module:        backupModule,
		Version:       fmt.Sprintf("%d", backupSchemaVersion),
		ExportedAt:    timestamppb.New(a.Manifest.ExportedAt),
		TenantId:      tenantID,
		EntityCounts:  a.Manifest.EntityCounts,
		SchemaVersion: int32(backupSchemaVersion),
	}, nil
}

func (s *BackupService) ImportBackup(ctx context.Context, req *hrV1.ImportBackupRequest) (*hrV1.ImportBackupResponse, error) {
	tenantID := grpcx.GetTenantIDFromContext(ctx)
	isPlatformAdmin := grpcx.IsPlatformAdmin(ctx)
	mode := mapHrRestoreMode(req.GetMode())

	a, err := backup.Unpack(req.GetData())
	if err != nil {
		return nil, fmt.Errorf("unpack backup: %w", err)
	}

	if err := backup.Validate(a, backupModule, backupSchemaVersion); err != nil {
		return nil, err
	}

	if a.Manifest.FullBackup && !isPlatformAdmin {
		return nil, fmt.Errorf("only platform admins can restore full backups")
	}

	sourceVersion := a.Manifest.SchemaVersion
	applied, err := backupMigrations.RunMigrations(a, backupSchemaVersion)
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	if !isPlatformAdmin || !a.Manifest.FullBackup {
		tenantID = grpcx.GetTenantIDFromContext(ctx)
	} else {
		tenantID = 0
	}

	client := s.entClient.Client()
	result := backup.NewRestoreResult(sourceVersion, backupSchemaVersion, applied)

	// Import in FK dependency order: pools → absenceTypes → allowances → requests
	s.importAllowancePools(ctx, client, a, tenantID, a.Manifest.FullBackup, mode, result)
	s.importAbsenceTypes(ctx, client, a, tenantID, a.Manifest.FullBackup, mode, result)
	s.importLeaveAllowances(ctx, client, a, tenantID, a.Manifest.FullBackup, mode, result)
	s.importLeaveRequests(ctx, client, a, tenantID, a.Manifest.FullBackup, mode, result)

	s.log.Infof("imported backup: module=%s tenant=%d migrations=%d results=%d",
		backupModule, tenantID, applied, len(result.Results))

	protoResults := make([]*hrV1.EntityImportResult, len(result.Results))
	for i, r := range result.Results {
		protoResults[i] = &hrV1.EntityImportResult{
			EntityType: r.EntityType,
			Total:      r.Total,
			Created:    r.Created,
			Updated:    r.Updated,
			Skipped:    r.Skipped,
			Failed:     r.Failed,
		}
	}

	return &hrV1.ImportBackupResponse{
		Success:           result.Success,
		Results:           protoResults,
		Warnings:          result.Warnings,
		SourceVersion:     int32(result.SourceVersion),
		TargetVersion:     int32(result.TargetVersion),
		MigrationsApplied: int32(result.MigrationsApplied),
	}, nil
}

func mapHrRestoreMode(m hrV1.RestoreMode) backup.RestoreMode {
	if m == hrV1.RestoreMode_RESTORE_MODE_OVERWRITE {
		return backup.RestoreModeOverwrite
	}
	return backup.RestoreModeSkip
}

// --- Import helpers ---

func (s *BackupService) importAllowancePools(ctx context.Context, client *ent.Client, a *backup.Archive, tenantID uint32, full bool, mode backup.RestoreMode, result *backup.RestoreResult) {
	pools, err := backup.GetEntities[ent.AllowancePool](a, "allowancePools")
	if err != nil {
		result.AddWarning(fmt.Sprintf("allowancePools: unmarshal error: %v", err))
		return
	}
	if len(pools) == 0 {
		return
	}

	er := backup.EntityResult{EntityType: "allowancePools", Total: int64(len(pools))}

	for _, e := range pools {
		tid := tenantID
		if full && e.TenantID != nil {
			tid = *e.TenantID
		}

		existing, getErr := client.AllowancePool.Get(ctx, e.ID)
		if getErr != nil && !ent.IsNotFound(getErr) {
			result.AddWarning(fmt.Sprintf("allowancePools: lookup %s: %v", e.ID, getErr))
			er.Failed++
			continue
		}

		if existing != nil {
			if mode == backup.RestoreModeSkip {
				er.Skipped++
				continue
			}
			_, err := client.AllowancePool.UpdateOneID(e.ID).
				SetName(e.Name).
				SetDescription(e.Description).
				SetColor(e.Color).
				SetIcon(e.Icon).
				SetNillableCreateBy(e.CreateBy).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("allowancePools: update %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Updated++
		} else {
			_, err := client.AllowancePool.Create().
				SetID(e.ID).
				SetNillableTenantID(&tid).
				SetName(e.Name).
				SetDescription(e.Description).
				SetColor(e.Color).
				SetIcon(e.Icon).
				SetNillableCreateBy(e.CreateBy).
				SetNillableCreateTime(e.CreateTime).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("allowancePools: create %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Created++
		}
	}

	result.AddResult(er)
}

func (s *BackupService) importAbsenceTypes(ctx context.Context, client *ent.Client, a *backup.Archive, tenantID uint32, full bool, mode backup.RestoreMode, result *backup.RestoreResult) {
	types, err := backup.GetEntities[ent.AbsenceType](a, "absenceTypes")
	if err != nil {
		result.AddWarning(fmt.Sprintf("absenceTypes: unmarshal error: %v", err))
		return
	}
	if len(types) == 0 {
		return
	}

	er := backup.EntityResult{EntityType: "absenceTypes", Total: int64(len(types))}

	for _, e := range types {
		tid := tenantID
		if full && e.TenantID != nil {
			tid = *e.TenantID
		}

		existing, getErr := client.AbsenceType.Get(ctx, e.ID)
		if getErr != nil && !ent.IsNotFound(getErr) {
			result.AddWarning(fmt.Sprintf("absenceTypes: lookup %s: %v", e.ID, getErr))
			er.Failed++
			continue
		}

		if existing != nil {
			if mode == backup.RestoreModeSkip {
				er.Skipped++
				continue
			}
			_, err := client.AbsenceType.UpdateOneID(e.ID).
				SetName(e.Name).
				SetDescription(e.Description).
				SetColor(e.Color).
				SetIcon(e.Icon).
				SetDeductsFromAllowance(e.DeductsFromAllowance).
				SetRequiresApproval(e.RequiresApproval).
				SetIsActive(e.IsActive).
				SetSortOrder(e.SortOrder).
				SetMetadata(e.Metadata).
				SetRequiresSigning(e.RequiresSigning).
				SetSigningTemplateID(e.SigningTemplateID).
				SetAllowancePoolID(e.AllowancePoolID).
				SetNillableCreateBy(e.CreateBy).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("absenceTypes: update %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Updated++
		} else {
			_, err := client.AbsenceType.Create().
				SetID(e.ID).
				SetNillableTenantID(&tid).
				SetName(e.Name).
				SetDescription(e.Description).
				SetColor(e.Color).
				SetIcon(e.Icon).
				SetDeductsFromAllowance(e.DeductsFromAllowance).
				SetRequiresApproval(e.RequiresApproval).
				SetIsActive(e.IsActive).
				SetSortOrder(e.SortOrder).
				SetMetadata(e.Metadata).
				SetRequiresSigning(e.RequiresSigning).
				SetSigningTemplateID(e.SigningTemplateID).
				SetAllowancePoolID(e.AllowancePoolID).
				SetNillableCreateBy(e.CreateBy).
				SetNillableCreateTime(e.CreateTime).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("absenceTypes: create %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Created++
		}
	}

	result.AddResult(er)
}

func (s *BackupService) importLeaveAllowances(ctx context.Context, client *ent.Client, a *backup.Archive, tenantID uint32, full bool, mode backup.RestoreMode, result *backup.RestoreResult) {
	allowances, err := backup.GetEntities[ent.LeaveAllowance](a, "leaveAllowances")
	if err != nil {
		result.AddWarning(fmt.Sprintf("leaveAllowances: unmarshal error: %v", err))
		return
	}
	if len(allowances) == 0 {
		return
	}

	er := backup.EntityResult{EntityType: "leaveAllowances", Total: int64(len(allowances))}

	for _, e := range allowances {
		tid := tenantID
		if full && e.TenantID != nil {
			tid = *e.TenantID
		}

		existing, getErr := client.LeaveAllowance.Get(ctx, e.ID)
		if getErr != nil && !ent.IsNotFound(getErr) {
			result.AddWarning(fmt.Sprintf("leaveAllowances: lookup %s: %v", e.ID, getErr))
			er.Failed++
			continue
		}

		if existing != nil {
			if mode == backup.RestoreModeSkip {
				er.Skipped++
				continue
			}
			_, err := client.LeaveAllowance.UpdateOneID(e.ID).
				SetUserID(e.UserID).
				SetUserName(e.UserName).
				SetNillableAbsenceTypeID(e.AbsenceTypeID).
				SetNillableAllowancePoolID(e.AllowancePoolID).
				SetYear(e.Year).
				SetTotalDays(e.TotalDays).
				SetUsedDays(e.UsedDays).
				SetCarriedOver(e.CarriedOver).
				SetNotes(e.Notes).
				SetNillableCreateBy(e.CreateBy).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("leaveAllowances: update %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Updated++
		} else {
			_, err := client.LeaveAllowance.Create().
				SetID(e.ID).
				SetNillableTenantID(&tid).
				SetUserID(e.UserID).
				SetUserName(e.UserName).
				SetNillableAbsenceTypeID(e.AbsenceTypeID).
				SetNillableAllowancePoolID(e.AllowancePoolID).
				SetYear(e.Year).
				SetTotalDays(e.TotalDays).
				SetUsedDays(e.UsedDays).
				SetCarriedOver(e.CarriedOver).
				SetNotes(e.Notes).
				SetNillableCreateBy(e.CreateBy).
				SetNillableCreateTime(e.CreateTime).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("leaveAllowances: create %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Created++
		}
	}

	result.AddResult(er)
}

func (s *BackupService) importLeaveRequests(ctx context.Context, client *ent.Client, a *backup.Archive, tenantID uint32, full bool, mode backup.RestoreMode, result *backup.RestoreResult) {
	requests, err := backup.GetEntities[ent.LeaveRequest](a, "leaveRequests")
	if err != nil {
		result.AddWarning(fmt.Sprintf("leaveRequests: unmarshal error: %v", err))
		return
	}
	if len(requests) == 0 {
		return
	}

	er := backup.EntityResult{EntityType: "leaveRequests", Total: int64(len(requests))}

	for _, e := range requests {
		tid := tenantID
		if full && e.TenantID != nil {
			tid = *e.TenantID
		}

		existing, getErr := client.LeaveRequest.Get(ctx, e.ID)
		if getErr != nil && !ent.IsNotFound(getErr) {
			result.AddWarning(fmt.Sprintf("leaveRequests: lookup %s: %v", e.ID, getErr))
			er.Failed++
			continue
		}

		if existing != nil {
			if mode == backup.RestoreModeSkip {
				er.Skipped++
				continue
			}
			_, err := client.LeaveRequest.UpdateOneID(e.ID).
				SetUserID(e.UserID).
				SetUserName(e.UserName).
				SetUserEmail(e.UserEmail).
				SetOrgUnitName(e.OrgUnitName).
				SetAbsenceTypeID(e.AbsenceTypeID).
				SetStartDate(e.StartDate).
				SetEndDate(e.EndDate).
				SetDays(e.Days).
				SetStatus(e.Status).
				SetSigningRequestID(e.SigningRequestID).
				SetReason(e.Reason).
				SetReviewNotes(e.ReviewNotes).
				SetReviewedBy(e.ReviewedBy).
				SetReviewerName(e.ReviewerName).
				SetNillableReviewedAt(e.ReviewedAt).
				SetNotes(e.Notes).
				SetMetadata(e.Metadata).
				SetDeductedAllowanceID(e.DeductedAllowanceID).
				SetNillableCreateBy(e.CreateBy).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("leaveRequests: update %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Updated++
		} else {
			_, err := client.LeaveRequest.Create().
				SetID(e.ID).
				SetNillableTenantID(&tid).
				SetUserID(e.UserID).
				SetUserName(e.UserName).
				SetUserEmail(e.UserEmail).
				SetOrgUnitName(e.OrgUnitName).
				SetAbsenceTypeID(e.AbsenceTypeID).
				SetStartDate(e.StartDate).
				SetEndDate(e.EndDate).
				SetDays(e.Days).
				SetStatus(e.Status).
				SetSigningRequestID(e.SigningRequestID).
				SetReason(e.Reason).
				SetReviewNotes(e.ReviewNotes).
				SetReviewedBy(e.ReviewedBy).
				SetReviewerName(e.ReviewerName).
				SetNillableReviewedAt(e.ReviewedAt).
				SetNotes(e.Notes).
				SetMetadata(e.Metadata).
				SetDeductedAllowanceID(e.DeductedAllowanceID).
				SetNillableCreateBy(e.CreateBy).
				SetNillableCreateTime(e.CreateTime).
				Save(ctx)
			if err != nil {
				result.AddWarning(fmt.Sprintf("leaveRequests: create %s: %v", e.ID, err))
				er.Failed++
				continue
			}
			er.Created++
		}
	}

	result.AddResult(er)
}
