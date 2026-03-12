package main

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-common/registration"
	pkgService "github.com/go-tangra/go-tangra-common/service"
	"github.com/go-tangra/go-tangra-hr/cmd/server/assets"
	hrCnf "github.com/go-tangra/go-tangra-hr/internal/conf"
	"github.com/go-tangra/go-tangra-hr/internal/event"
)

var (
	// Module info
	moduleID    = "hr"
	moduleName  = "Human Resources"
	version     = "1.0.0"
	description = "Human Resources service for managing employees, leave requests, absence types, and vacation planning"
)

// Global references for cleanup
var globalRegHelper *registration.RegistrationHelper
var globalEventSubscriber *event.Subscriber

func newApp(
	ctx *bootstrap.Context,
	gs *grpc.Server,
	hs *kratosHttp.Server,
	eventSubscriber *event.Subscriber,
	regClient *registration.Client,
) *kratos.App {
	// Start the event subscriber and store reference for cleanup
	globalEventSubscriber = eventSubscriber
	if eventSubscriber != nil {
		if err := eventSubscriber.Start(); err != nil {
			log.Warnf("Failed to start event subscriber: %v", err)
		}
	}

	if regClient != nil {
		// Populate the full registration config on the pre-created client
		regClient.SetConfig(&registration.Config{
			ModuleID:         moduleID,
			ModuleName:       moduleName,
			Version:          version,
			Description:      description,
			GRPCEndpoint:     registration.GetGRPCAdvertiseAddr(ctx, "0.0.0.0:10200"),
			FrontendEntryUrl: registration.GetEnvOrDefault("FRONTEND_ENTRY_URL", ""),
			HttpEndpoint:     registration.GetEnvOrDefault("HTTP_ADVERTISE_ADDR", ""),
			OpenapiSpec:      assets.OpenApiData,
			ProtoDescriptor:  assets.DescriptorData,
			MenusYaml:        assets.MenusData,
		})
		globalRegHelper = registration.StartRegistrationWithClient(ctx.GetLogger(), regClient)
	}

	return bootstrap.NewApp(ctx, gs, hs)
}

// stopServices stops background services
func stopServices() {
	if globalEventSubscriber != nil {
		if err := globalEventSubscriber.Stop(); err != nil {
			log.Warnf("Failed to stop event subscriber: %v", err)
		}
	}
}

func runApp() error {
	ctx := bootstrap.NewContext(
		context.Background(),
		&conf.AppInfo{
			Project: pkgService.Project,
			AppId:   "hr.service",
			Version: version,
		},
	)
	ctx.RegisterCustomConfig("hr", &hrCnf.HR{})

	defer stopServices()
	defer func() {
		if globalRegHelper != nil {
			globalRegHelper.Stop()
		}
	}()

	return bootstrap.RunApp(ctx, initApp)
}

func main() {
	if err := runApp(); err != nil {
		panic(err)
	}
}
