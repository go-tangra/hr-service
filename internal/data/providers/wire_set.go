//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package providers

import (
	"github.com/google/wire"

	"github.com/go-tangra/go-tangra-hr/internal/data"
)

var ProviderSet = wire.NewSet(
	data.NewRedisClient,
	data.NewEntClient,
	data.NewAbsenceTypeRepo,
	data.NewLeaveAllowanceRepo,
	data.NewLeaveRequestRepo,
	data.NewAuditLogRepo,
)
