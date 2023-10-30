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
)

func configureEchoRouters(e *echo.Echo, grpcServer *grpc.Server, mux *grpcruntime.ServeMux) {
	// Embed frontend.
	embedFrontend(e)

	e.HideBanner = true
	e.HidePort = true
	e.Use(recoverMiddleware)
	grpcSkipper := func(c echo.Context) bool {
		// Skip grpc and webhook calls.
		return strings.HasPrefix(c.Request().URL.Path, "/bytebase.v1.") ||
			strings.HasPrefix(c.Request().URL.Path, "/v1:adminExecute") ||
			strings.HasPrefix(c.Request().URL.Path, webhookAPIPrefix)
	}
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Skipper: grpcSkipper,
		Timeout: 30 * time.Second,
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
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}))
	pprof.Register(e)
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
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
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
