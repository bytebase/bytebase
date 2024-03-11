package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

func configureEchoRouters(e *echo.Echo, grpcServer *grpc.Server, mux *grpcruntime.ServeMux, profile config.Profile) {
	// Embed frontend.
	embedFrontend(e)

	e.HideBanner = true
	e.HidePort = true
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
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: grpcSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 30, Burst: 60, ExpiresIn: 3 * time.Minute},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, _ error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, _ string, _ error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}))
	if profile.Debug {
		pprof.Register(e)
	}
	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/v1:adminExecute", echo.WrapHandler(wsproxy.WebsocketProxy(
		mux,
		wsproxy.WithTokenCookieName("access-token"),
		// 10M.
		wsproxy.WithMaxRespBodyBufferSize(10*1024*1024),
	)))
	e.Any("/v1/*", echo.WrapHandler(mux))

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
