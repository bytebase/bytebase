package fake

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// GitLab is a fake implementation of GitLab.
type GitLab struct {
	port int
	Echo *echo.Echo
}

// NewGitLab creates a fake GitLab.
func NewGitLab(port int) *GitLab {
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	// TODO(d-bytebase): implement routes

	return &GitLab{
		Echo: e,
		port: port,
	}
}

// Run runs a GitLab server.
func (g *GitLab) Run() error {
	return g.Echo.Start(fmt.Sprintf(":%d", g.port))
}

// Close close a GitLab server.
func (g *GitLab) Close() error {
	return g.Echo.Close()
}
