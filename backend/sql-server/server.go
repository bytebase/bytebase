// Package sqlserver is the binary for SQL review server.
package sqlserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/bytebase/bytebase/backend/plugin/metric/segment"
	_ "github.com/bytebase/bytebase/docs/sqlservice" // initial the swagger doc

	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
)

// Server is the Bytebase server.
type Server struct {
	profile        Profile
	e              *echo.Echo
	startedTs      int64
	metricReporter *metricReporter
}

// Use following cmd to generate swagger doc
// swag init -g ./backend/server.go -d ./backend/sql-server --output docs/sqlservice --parseDependency

// @title Bytebase SQL Service
// @version 1.0
// @description The OpenAPI for Bytebase SQL Service.
// @termsOfService https://www.bytebase.com/terms

// @contact.name API Support
// @contact.url https://github.com/bytebase/bytebase/
// @contact.email support@bytebase.com

// @license.name MIT
// @license.url https://github.com/bytebase/bytebase/blob/main/LICENSE

// @host localhost:8081
// @BasePath /v1/
// @schemes http

// NewServer creates a server.
func NewServer(ctx context.Context, prof Profile) (*Server, error) {
	s := &Server{
		profile:   prof,
		startedTs: time.Now().Unix(),
		metricReporter: &metricReporter{
			workspaceID: prof.WorkspaceID,
			reporter:    segment.NewReporter(prof.MetricConnectionKey),
		},
	}

	// Display config
	log.Info("-----Config BEGIN-----")
	log.Info(fmt.Sprintf("mode=%s", prof.Mode))
	log.Info(fmt.Sprintf("server=%s:%d", prof.BackendHost, prof.BackendPort))
	log.Info(fmt.Sprintf("debug=%t", prof.Debug))
	log.Info(fmt.Sprintf("workspaceID=%s", prof.WorkspaceID))
	log.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	e := echo.New()
	e.Debug = prof.Debug
	s.e = e

	// Middleware
	if prof.Mode == common.ReleaseModeDev || prof.Debug {
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Skipper: func(c echo.Context) bool {
				return !common.HasPrefixes(c.Path(), "/v1")
			},
			Format: `{"time":"${time_rfc3339}",` +
				`"method":"${method}","uri":"${uri}",` +
				`"status":${status},"error":"${error}"}` + "\n",
		}))
	}
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{}))
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	apiGroup := e.Group("/v1")
	apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return metricMiddleware(s, next)
	})
	s.registerAdvisorRoutes(apiGroup)

	// Register healthz endpoint.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!\n")
	})
	// Register pprof endpoints.
	pprof.Register(e)
	// Register prometheus metrics endpoint.
	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	serverStarted = true
	return s, nil
}

// Run will run the server.
func (s *Server) Run() error {
	return s.e.Start(fmt.Sprintf(":%d", s.profile.BackendPort))
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Trying to stop Bytebase SQL Service ....")
	log.Info("Trying to gracefully shutdown server")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown echo
	if s.e != nil {
		if err := s.e.Shutdown(ctx); err != nil {
			s.e.Logger.Fatal(err)
		}
	}

	log.Info("Bytebase SQL Service stopped properly")

	return nil
}
