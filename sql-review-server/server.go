package sqlreviewserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server is the Bytebase server.
type Server struct {
	profile   Profile
	e         *echo.Echo
	startedTs int64
	secret    string
}

// NewServer creates a server.
func NewServer(ctx context.Context, prof Profile) (*Server, error) {
	s := &Server{
		profile:   prof,
		startedTs: time.Now().Unix(),
	}

	// Display config
	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", prof.Mode)
	fmt.Printf("server=%s:%d\n", prof.BackendHost, prof.BackendPort)
	fmt.Printf("debug=%t\n", prof.Debug)
	fmt.Println("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	e := echo.New()
	e.Debug = prof.Debug
	// Disallow to be embedded in an iFrame.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XFrameOptions: "DENY",
	}))
	s.e = e

	// Middleware
	if prof.Mode == common.ReleaseModeDev || prof.Debug {
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Skipper: func(c echo.Context) bool {
				return !common.HasPrefixes(c.Path(), "/api")
			},
			Format: `{"time":"${time_rfc3339}",` +
				`"method":"${method}","uri":"${uri}",` +
				`"status":${status},"error":"${error}"}` + "\n",
		}))
	}
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{}))

	apiGroup := e.Group("/api")
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
func (s *Server) Run(ctx context.Context) error {
	// Sleep for 1 sec to make sure port is released between runs.
	time.Sleep(time.Duration(1) * time.Second)

	return s.e.Start(fmt.Sprintf(":%d", s.profile.BackendPort))
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Trying to stop Bytebase ....")
	log.Info("Trying to gracefully shutdown server")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown echo
	if s.e != nil {
		if err := s.e.Shutdown(ctx); err != nil {
			s.e.Logger.Fatal(err)
		}
	}

	log.Info("Bytebase stopped properly")

	return nil
}
