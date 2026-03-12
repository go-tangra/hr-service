package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/transport"
)

// getTenantID extracts tenant_id from gRPC metadata
func getTenantID(ctx context.Context) uint32 {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if v := tr.RequestHeader().Get("x-md-global-tenant-id"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 32); err == nil {
				return uint32(id)
			}
		}
	}
	return 0
}

// getUserID extracts user_id from gRPC metadata
func getUserID(ctx context.Context) uint32 {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if v := tr.RequestHeader().Get("x-md-global-user-id"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 32); err == nil {
				return uint32(id)
			}
		}
	}
	return 0
}

// getUsername extracts username from gRPC metadata
func getUsername(ctx context.Context) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		return tr.RequestHeader().Get("x-md-global-username")
	}
	return ""
}

// checkTenantAccess verifies the entity belongs to the caller's tenant.
// Returns a "not found" error if tenant doesn't match (to avoid leaking resource existence across tenants).
// System callers (tenantID == 0) bypass the check.
func checkTenantAccess(ctx context.Context, entityTenantID *uint32, notFoundErr error) error {
	callerTenantID := getTenantID(ctx)
	if callerTenantID == 0 {
		return nil // system caller, no tenant restriction
	}
	if entityTenantID == nil || *entityTenantID != callerTenantID {
		return notFoundErr
	}
	return nil
}

// getRoles extracts roles from gRPC metadata (comma-separated)
func getRoles(ctx context.Context) []string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if v := tr.RequestHeader().Get("x-md-global-roles"); v != "" {
			return strings.Split(v, ",")
		}
	}
	return nil
}
