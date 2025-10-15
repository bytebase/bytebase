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
	"github.com/bytebase/bytebase/backend/common"
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
	e.Use(securityHeadersMiddleware)

	if profile.Mode == common.ReleaseModeDev {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOriginFunc: func(string) (bool, error) {
				return true, nil
			},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowHeaders:     connectcors.AllowedHeaders(),
			ExposeHeaders:    connectcors.ExposedHeaders(),
			AllowCredentials: true,
		}))
	}

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

func securityHeadersMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Allow popups to maintain window.opener for OAuth flows
		c.Response().Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		// Prevent being embedded in iframes from different origins
		c.Response().Header().Set("X-Frame-Options", "SAMEORIGIN")
		// Prevent MIME-type sniffing
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		// Force HTTPS in production (only if request is already HTTPS)
		if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
			// max-age=31536000 (1 year), includeSubDomains for all subdomains
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		// Content Security Policy - strict, no unsafe-inline
		// csp := "default-src 'self'; " +
		// 	"script-src 'self'; " +
		// 	"style-src 'self'; " +
		// 	"img-src 'self' data: blob:; " +
		// 	"connect-src 'self' ws: wss:; " +
		// 	"font-src 'self'; " +
		// 	"object-src 'none'; " +
		// 	"base-uri 'self'; " +
		// 	"form-action 'self'; " +
		// 	"frame-ancestors 'self'"
		// c.Response().Header().Set("Content-Security-Policy", csp)
		return next(c)
	}
}
