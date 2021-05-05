package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// const DSN = ":memory:"
const DSN = "./data/bytebase_dev.db"

type Main struct {
	server *Server

	db *sqlite.DB
}

func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	// Execute program.
	if err := m.Run(); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up program.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type Server struct {
	DebugService       api.DebugService
	AuthService        api.AuthService
	EnvironmentService api.EnvironmentService

	e *echo.Echo
}

//go:embed dist
var embededFiles embed.FS

//go:embed dist/index.html
var indexContent string

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func NewMain() *Main {
	return &Main{
		server: NewServer(),
		db:     sqlite.NewDB(DSN),
	}
}

func (m *Main) Run() error {
	if err := m.db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	if err := m.server.Run(); err != nil {
		return err
	}

	return nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m.server != nil {
		m.server.Close(ctx)
	}

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

func NewServer() *Server {
	e := echo.New()

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Path(), "/api")
		},
		Format: `{"time":"${time_rfc3339}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"}` + "\n",
	}))
	e.Use(middleware.Recover())

	// Catch-all route to return index.html, this is to prevent 404 when accessing non-root url.
	// See https://stackoverflow.com/questions/27928372/react-router-urls-dont-work-when-refreshing-or-writing-manually
	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexContent)
	})

	assetHandler := http.FileServer(getFileSystem())
	e.GET("/assets/*", echo.WrapHandler(assetHandler))

	s := &Server{
		e: e,
	}

	g := e.Group("/api")

	s.DebugService.RegisterRoutes(g)

	s.AuthService.RegisterRoutes(g)

	s.EnvironmentService.RegisterRoutes(g)

	return s
}

func (server *Server) Run() error {
	return server.e.Start(":8080")
}

func (server *Server) Close(ctx context.Context) {
	if err := server.e.Shutdown(ctx); err != nil {
		server.e.Logger.Fatal(err)
	}
}
