package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
)

// rolePermissions defines which permission codes each role grants.
// Roles with "*" have access to all permissions.
var rolePermissions = map[string][]string{
	"platform:admin": {"*"},
	"tenant:manager": {"*"},
	"hr.admin": {
		"hr.calendar.view",
		"hr.request.view",
		"hr.request.manage",
		"hr.request.delete",
		"hr.request.approve",
		"hr.absence_type.view",
		"hr.absence_type.manage",
		"hr.allowance.view",
		"hr.allowance.manage",
		"hr.allowance_pool.manage",
		"hr.users.list",
	},
	"hr.employee": {
		"hr.calendar.view",
		"hr.request.view",
		"hr.request.manage",
		"hr.allowance.view",
		"hr.users.list",
	},
	"hr.viewer": {
		"hr.calendar.view",
		"hr.request.view",
		"hr.absence_type.view",
		"hr.allowance.view",
	},
	"hr.client": {
		"hr.calendar.view",
		"hr.request.view",
	},
}

// hasPermission checks if the current user (from context) has the given permission code.
func hasPermission(ctx context.Context, code string) bool {
	roles := getRoles(ctx)
	for _, role := range roles {
		perms, ok := rolePermissions[role]
		if !ok {
			continue
		}
		for _, p := range perms {
			if p == "*" || p == code {
				return true
			}
		}
	}
	return false
}

// checkPermission returns a PermissionDenied error if the user lacks the required permission.
func checkPermission(ctx context.Context, required string) error {
	if hasPermission(ctx, required) {
		return nil
	}
	return errors.New(403, "PERMISSION_DENIED", fmt.Sprintf("permission denied: requires %s", required))
}
