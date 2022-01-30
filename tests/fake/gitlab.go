package fake

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// GitLab is a fake implementation of GitLab.
type GitLab struct {
	port int
	Echo *echo.Echo

	client *http.Client

	nextWebhookID int
	projectHooks  map[string][]*gitlab.WebhookPost
}

// NewGitLab creates a fake GitLab.
func NewGitLab(port int) *GitLab {
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	gl := &GitLab{
		port:          port,
		Echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		projectHooks:  map[string][]*gitlab.WebhookPost{},
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

// CreateProject creates a GitLab project.
func (gl *GitLab) CreateProject(id string) {
	gl.projectHooks[id] = nil
}

// createProjectHook creates a project webhook.
func (gl *GitLab) createProjectHook(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	c.Logger().Info("create webhook for project %q", c.Param("id"))
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("failed to read create project hook request body, error %w", err)
	}
	webhookPost := &gitlab.WebhookPost{}
	if err := json.Unmarshal(b, webhookPost); err != nil {
		return fmt.Errorf("failed to unmarshal create project hook request body, error %w", err)
	}
	if _, ok := gl.projectHooks[gitlabProjectID]; !ok {
		return fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}
	gl.projectHooks[gitlabProjectID] = append(gl.projectHooks[gitlabProjectID], webhookPost)

	if err := json.NewEncoder(c.Response().Writer).Encode(&gitlab.WebhookInfo{
		ID: gl.nextWebhookID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal WebhookInfo response").SetInternal(err)
	}
	gl.nextWebhookID++

	return nil
}

// SendCommits sends comments to webhooks.
func (gl *GitLab) SendCommits(gitlabProjectID string, webhookPushEvent *gitlab.WebhookPushEvent) error {
	webhookPosts, ok := gl.projectHooks[gitlabProjectID]
	if !ok {
		return fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}

	for _, webhookPost := range webhookPosts {
		// Send post request.
		buf, err := json.Marshal(webhookPushEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal webhookPushEvent, error %w", err)
		}
		req, err := http.NewRequest("POST", webhookPost.URL, strings.NewReader(string(buf)))
		if err != nil {
			return fmt.Errorf("fail to create a new POST request(%q), error: %w", webhookPost.URL, err)
		}
		req.Header.Set("X-Gitlab-Token", webhookPost.SecretToken)
		resp, err := gl.client.Do(req)
		if err != nil {
			return fmt.Errorf("fail to send a POST request(%q), error: %w", webhookPost.URL, err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read http response body, error: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
		}
		gl.Echo.Logger.Infof("SendCommits response body %s\n", body)
	}

	return nil
}
