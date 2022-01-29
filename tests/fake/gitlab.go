package fake

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// GitLab is a fake implementation of GitLab.
type GitLab struct {
	port int
	Echo *echo.Echo

	nextWebhookID int
}

// NewGitLab creates a fake GitLab.
func NewGitLab(port int) *GitLab {
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	gl := &GitLab{
		Echo:          e,
		port:          port,
		nextWebhookID: 20210113,
	}

	// Routes
	projectGroup := e.Group("/api/v4")
	projectGroup.POST("/projects/:id/hooks", gl.createProjectHook)

	return gl
}

// Run runs a GitLab server.
func (gl *GitLab) Run() error {
	return gl.Echo.Start(fmt.Sprintf(":%d", gl.port))
}

// Close close a GitLab server.
func (gl *GitLab) Close() error {
	return gl.Echo.Close()
}

// createProjectHook creates a project webhook.
func (gl *GitLab) createProjectHook(c echo.Context) error {
	c.Logger().Info("create webhook for project %q", c.Param("id"))

	if err := json.NewEncoder(c.Response().Writer).Encode(&gitlab.WebhookInfo{
		ID: gl.nextWebhookID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal WebhookInfo response").SetInternal(err)
	}
	gl.nextWebhookID++

	return nil
}
