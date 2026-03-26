package server

import (
	"io/fs"
	"net/http"
	"os"
	"strconv"

	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/go-tangra/go-tangra-hr/cmd/server/assets"
	"github.com/go-tangra/go-tangra-hr/internal/service"
)

// NewHTTPServer creates a simple HTTP server for serving the frontend assets
// and API endpoints that need to return binary data (e.g., PDF downloads).
func NewHTTPServer(ctx *bootstrap.Context, leaveSvc *service.LeaveService) *kratosHttp.Server {
	l := ctx.NewLoggerHelper("hr/http")

	addr := os.Getenv("HR_HTTP_ADDR")
	if addr == "" {
		addr = "0.0.0.0:10201"
	}

	srv := kratosHttp.NewServer(kratosHttp.Address(addr))

	route := srv.Route("/")
	route.GET("/health", func(ctx kratosHttp.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Signed document download — streams PDF to the browser.
	// Path: /api/v1/leave-requests/download-signed?id={leave_request_id}
	route.GET("/api/v1/leave-requests/download-signed", func(httpCtx kratosHttp.Context) error {
		leaveRequestID := httpCtx.Request().URL.Query().Get("id")
		if leaveRequestID == "" {
			return httpCtx.JSON(http.StatusBadRequest, map[string]string{"error": "missing id parameter"})
		}

		// Extract caller user ID from gateway-injected metadata headers
		uidStr := httpCtx.Request().Header.Get("x-md-global-user-id")
		callerID, _ := strconv.ParseUint(uidStr, 10, 32)
		if callerID == 0 {
			return httpCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		pdfBytes, err := leaveSvc.DownloadSignedDocument(httpCtx.Request().Context(), leaveRequestID, uint32(callerID))
		if err != nil {
			l.Errorf("download signed document: %v", err)
			return httpCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to download document"})
		}

		httpCtx.Response().Header().Set("Content-Type", "application/pdf")
		httpCtx.Response().Header().Set("Content-Disposition", "attachment; filename=\"signed-document.pdf\"")
		_, writeErr := httpCtx.Response().Write(pdfBytes)
		return writeErr
	})

	fsys, err := fs.Sub(assets.FrontendDist, "frontend-dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(fsys))
		srv.HandlePrefix("/", fileServer)
		l.Infof("Serving embedded frontend assets")
	} else {
		l.Warnf("Failed to load embedded frontend assets: %v", err)
	}

	l.Infof("HTTP server listening on %s", addr)
	return srv
}
