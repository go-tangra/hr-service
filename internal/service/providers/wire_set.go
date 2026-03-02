//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package providers

import (
	"github.com/google/wire"

	"github.com/go-tangra/go-tangra-hr/internal/client"
	"github.com/go-tangra/go-tangra-hr/internal/event"
	"github.com/go-tangra/go-tangra-hr/internal/service"
)

var ProviderSet = wire.NewSet(
	service.NewSystemService,
	service.NewAbsenceTypeService,
	service.NewLeaveService,
	service.NewAllowanceService,
	client.NewPaperlessClient,
	event.NewHandler,
	event.NewSubscriber,
)
