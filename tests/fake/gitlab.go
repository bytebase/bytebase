package fake

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytebase/bytebase/plugin/vcs"
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
	projects      map[string]*projectData
}

type projectData struct {
	webhooks []*gitlab.WebhookPost
	// files is a map that the full file path is the key and the file content is the value.
	files map[string]string
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
		projects:      map[string]*projectData{},
	}

	// Routes
	projectGroup := e.Group("/api/v4")
	projectGroup.POST("/projects/:id/hooks", gl.createProjectHook)
	projectGroup.GET("/projects/:id/repository/commits/:commitID", gl.getFakeCommit)
	projectGroup.GET("/projects/:id/repository/tree", gl.readProjectTree)
	projectGroup.GET("/projects/:id/repository/files/:filePath/raw", gl.readProjectFile)
	projectGroup.GET("/projects/:id/repository/files/:filePath", gl.readProjectFileMetadata)
	projectGroup.POST("/projects/:id/repository/files/:filePath", gl.createProjectFile)
	projectGroup.PUT("/projects/:id/repository/files/:filePath", gl.createProjectFile)

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
	gl.projects[id] = &projectData{
		files: map[string]string{},
	}
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
	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}
	pd.webhooks = append(pd.webhooks, webhookPost)

	if err := json.NewEncoder(c.Response().Writer).Encode(&gitlab.WebhookInfo{
		ID: gl.nextWebhookID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal WebhookInfo response").SetInternal(err)
	}
	gl.nextWebhookID++

	return nil
}

// readProjectTree reads a project file nodes
func (gl *GitLab) readProjectTree(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	path := c.QueryParam("path")
	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", gitlabProjectID))
	}

	fileNodes := []*vcs.RepositoryTreeNode{}

	for filePath := range pd.files {
		if strings.HasPrefix(filePath, path) {
			fileNodes = append(fileNodes, &vcs.RepositoryTreeNode{
				Path: filePath,
				Type: "blob",
			})
		}
	}

	buf, err := json.Marshal(&fileNodes)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal fileNodes, error %v", err))
	}

	return c.String(http.StatusOK, string(buf))
}

// readProjectFile reads a project file
func (gl *GitLab) readProjectFile(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}

	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", gitlabProjectID))
	}

	content, ok := pd.files[filePath]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}

	return c.String(http.StatusOK, content)
}

// readProjectFileMetadata reads a project file metadata
func (gl *GitLab) readProjectFileMetadata(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}

	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", gitlabProjectID))
	}

	if _, ok := pd.files[filePath]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}

	fileName := filepath.Base(filePath)
	content := pd.files[filePath]

	buf, err := json.Marshal(&gitlab.File{
		FileName:     fileName,
		FilePath:     filePath,
		Content:      content,
		LastCommitID: "fake_gitlab_commit_id",
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal FileMeta, error %v", err))
	}

	return c.String(http.StatusOK, string(buf))
}

// getFakeCommit get a fake commit data
func (gl *GitLab) getFakeCommit(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	_, ok := gl.projects[gitlabProjectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", gitlabProjectID))
	}

	commit := gitlab.Commit{
		ID:         "fake_gitlab_commit_id",
		AuthorName: "fake_gitlab_bot",
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	buf, err := json.Marshal(commit)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal commit, error %v", err))
	}

	return c.String(http.StatusOK, string(buf))
}

// createProjectFile creates a project file.
func (gl *GitLab) createProjectFile(c echo.Context) error {
	gitlabProjectID := c.Param("id")
	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read create project file request body, error %v", err))
	}
	fileCommit := &gitlab.FileCommit{}
	if err := json.Unmarshal(b, fileCommit); err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal create project file request body, error %v", err))
	}

	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", gitlabProjectID))
	}

	// Save file.
	pd.files[filePath] = fileCommit.Content

	return c.String(http.StatusOK, "")
}

// SendCommits sends comments to webhooks.
func (gl *GitLab) SendCommits(gitlabProjectID string, webhookPushEvent *gitlab.WebhookPushEvent) error {
	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}

	// Trigger webhooks.
	for _, webhook := range pd.webhooks {
		// Send post request.
		buf, err := json.Marshal(webhookPushEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal webhookPushEvent, error %w", err)
		}
		req, err := http.NewRequest("POST", webhook.URL, strings.NewReader(string(buf)))
		if err != nil {
			return fmt.Errorf("fail to create a new POST request(%q), error: %w", webhook.URL, err)
		}
		req.Header.Set("X-Gitlab-Token", webhook.SecretToken)
		resp, err := gl.client.Do(req)
		if err != nil {
			return fmt.Errorf("fail to send a POST request(%q), error: %w", webhook.URL, err)
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

// AddFiles add files to repository.
func (gl *GitLab) AddFiles(gitlabProjectID string, files map[string]string) error {
	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}

	// Save files
	for path, content := range files {
		pd.files[path] = content
	}
	return nil
}

// GetFiles get files from repository.
func (gl *GitLab) GetFiles(gitlabProjectID string, filePaths ...string) (map[string]string, error) {
	pd, ok := gl.projects[gitlabProjectID]
	if !ok {
		return nil, fmt.Errorf("gitlab project %q doesn't exist", gitlabProjectID)
	}

	// Get files
	files := make(map[string]string, len(filePaths))
	for _, path := range filePaths {
		if content, ok := pd.files[path]; ok {
			files[path] = content
		}
	}
	return files, nil
}
