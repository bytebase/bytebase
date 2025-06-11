package server

import (
	"log/slog"
	"net/http"
	"strings"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"

	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

func configureEchoRouters(
	e *echo.Echo,
	grpcServer *grpc.Server,
	lspServer *lsp.Server,
	directorySyncServer *directorysync.Service,
	mux *grpcruntime.ServeMux,
	profile *config.Profile,
	connectHandlers map[string]http.Handler,
) {
	e.Use(recoverMiddleware)

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogError:  true,
		LogValuesFunc: func(_ echo.Context, values middleware.RequestLoggerValues) error {
			if values.Error != nil {
				slog.Error("echo request logger", "method", values.Method, "uri", values.URI, "status", values.Status, log.BBError(values.Error))
			}
			return nil
		},
	}))

	// Embed frontend.
	embedFrontend(e)

	e.HideBanner = true
	e.HidePort = true

	grpcSkipper := func(c echo.Context) bool {
		// Skip grpc and webhook calls.
		return strings.HasPrefix(c.Request().URL.Path, "/bytebase.v1.") ||
			strings.HasPrefix(c.Request().URL.Path, "/v1:adminExecute") ||
			strings.HasPrefix(c.Request().URL.Path, lspAPI) ||
			strings.HasPrefix(c.Request().URL.Path, webhookAPIPrefix)
	}
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper: grpcSkipper,
		Timeout: 0, // unlimited
	}))

	registerPprof(e, &profile.RuntimeDebug)

	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/v1:adminExecute", echo.WrapHandler(wsproxy.WebsocketProxy(
		mux,
		wsproxy.WithTokenCookieName("access-token"),
		// 100M.
		wsproxy.WithMaxRespBodyBufferSize(100*1024*1024),
	)))
	e.Any("/v1/*", echo.WrapHandler(mux))

	// Register Connect RPC handlers
	for path, handler := range connectHandlers {
		e.Any(path+"*", echo.WrapHandler(handler))
	}

	// GRPC web proxy.
	options := []grpcweb.Option{
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(_ string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(_ *http.Request) bool {
			return true
		}),
	}
	wrappedGrpc := grpcweb.WrapServer(grpcServer, options...)
	e.Any("/bytebase.v1.*", echo.WrapHandler(wrappedGrpc))

	// LSP server.
	e.GET(lspAPI, lspServer.Router)

	hookGroup := e.Group(webhookAPIPrefix)
	scimGroup := hookGroup.Group(scimAPIPrefix)
	directorySyncServer.RegisterDirectorySyncRoutes(scimGroup)
}

func recoverMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = errors.Errorf("%v", r)
				}
				slog.Error("Middleware PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))

				c.Error(err)
			}
		}()
		return next(c)
	}
}
