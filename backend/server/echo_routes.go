package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	connectcors "connectrpc.com/cors"

	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

func configureEchoRouters(
	e *echo.Echo,
	lspServer *lsp.Server,
	directorySyncServer *directorysync.Service,
	profile *config.Profile,
) {
	e.Use(recoverMiddleware)

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: func(string) (bool, error) {
			return true, nil
		},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		AllowHeaders:     connectcors.AllowedHeaders(),
		ExposeHeaders:    connectcors.ExposedHeaders(),
		AllowCredentials: true,
	}))

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

	registerPprof(e, &profile.RuntimeDebug)

	p := prometheus.NewPrometheus("api", nil)
	p.RequestCounterURLLabelMappingFunc = func(c echo.Context) string {
		return c.Request().URL.Path
	}
	p.Use(e)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

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
