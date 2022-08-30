package fake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/gitlab"
)

// GitLab is a fake implementation of GitLab VCS provider.
type GitLab struct {
	port int
	echo *echo.Echo

	client *http.Client

	nextWebhookID int
	projects      map[string]*projectData
}

type projectData struct {
	webhooks []*gitlab.WebhookCreate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
}

// NewGitLab creates a new fake implementation of GitLab VCS provider.
func NewGitLab(port int) VCSProvider {
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	gl := &GitLab{
		port:          port,
		echo:          e,
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

// Run starts the GitLab VCS provider server.
func (gl *GitLab) Run() error {
	return gl.echo.Start(fmt.Sprintf(":%d", gl.port))
}

// Close shuts down the GitLab VCS provider server.
func (gl *GitLab) Close() error {
	return gl.echo.Close()
}

// ListenerAddr returns the GitLab VCS provider server listener address.
func (gl *GitLab) ListenerAddr() net.Addr {
	return gl.echo.ListenerAddr()
}

// APIURL returns the GitLab VCS provider API URL.
func (*GitLab) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/api/v4", instanceURL)
}

// CreateRepository creates a GitLab project with given ID.
func (gl *GitLab) CreateRepository(id string) {
	gl.projects[id] = &projectData{
		files: map[string]string{},
	}
}

// createProjectHook creates a project webhook.
func (gl *GitLab) createProjectHook(c echo.Context) error {
	projectID := c.Param("id")
	c.Logger().Infof("Create webhook for project %q", c.Param("id"))
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read create project hook request body")
	}
	webhookCreate := &gitlab.WebhookCreate{}
	if err := json.Unmarshal(b, webhookCreate); err != nil {
		return errors.Wrap(err, "failed to unmarshal create project hook request body")
	}
	pd, ok := gl.projects[projectID]
	if !ok {
		return errors.Errorf("gitlab project %q doesn't exist", projectID)
	}
	pd.webhooks = append(pd.webhooks, webhookCreate)

	gl.nextWebhookID++

	return c.JSON(http.StatusCreated, &gitlab.WebhookInfo{
		ID: gl.nextWebhookID,
	})
}

// readProjectTree reads a project file nodes.
func (gl *GitLab) readProjectTree(c echo.Context) error {
	projectID := c.Param("id")
	path := c.QueryParam("path")
	pd, ok := gl.projects[projectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
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

// readProjectFile reads a project file.
func (gl *GitLab) readProjectFile(c echo.Context) error {
	projectID := c.Param("id")
	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}

	pd, ok := gl.projects[projectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
	}

	content, ok := pd.files[filePath]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}

	return c.String(http.StatusOK, content)
}

// readProjectFileMetadata reads a project file metadata.
func (gl *GitLab) readProjectFileMetadata(c echo.Context) error {
	projectID := c.Param("id")
	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}

	pd, ok := gl.projects[projectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
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

// getFakeCommit get a fake commit data.
func (gl *GitLab) getFakeCommit(c echo.Context) error {
	projectID := c.Param("id")
	_, ok := gl.projects[projectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
	}

	commit := gitlab.Commit{
		ID:         "fake_gitlab_commit_id",
		AuthorName: "fake_gitlab_bot",
		CreatedAt:  time.Now(),
	}
	buf, err := json.Marshal(commit)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal commit, error %v", err))
	}

	return c.String(http.StatusOK, string(buf))
}

// createProjectFile creates a project file.
func (gl *GitLab) createProjectFile(c echo.Context) error {
	projectID := c.Param("id")
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

	pd, ok := gl.projects[projectID]
	if !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
	}

	// Save file.
	pd.files[filePath] = fileCommit.Content

	return c.String(http.StatusOK, "")
}

// SendWebhookPush sends out a webhook for a push event for the GitLab project
// using given payload.
func (gl *GitLab) SendWebhookPush(projectID string, payload []byte) error {
	pd, ok := gl.projects[projectID]
	if !ok {
		return errors.Errorf("gitlab project %q doesn't exist", projectID)
	}

	// Trigger webhooks.
	for _, webhook := range pd.webhooks {
		// Send post request.
		req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payload))
		if err != nil {
			return errors.Wrapf(err, "fail to create a new POST request(%q)", webhook.URL)
		}
		req.Header.Set("X-Gitlab-Token", webhook.SecretToken)
		resp, err := gl.client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "fail to send a POST request(%q)", webhook.URL)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read http response body")
		}

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
		}
		gl.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
	}

	return nil
}

// AddFiles adds given files to the GitLab project.
func (gl *GitLab) AddFiles(projectID string, files map[string]string) error {
	pd, ok := gl.projects[projectID]
	if !ok {
		return errors.Errorf("gitlab project %q doesn't exist", projectID)
	}

	// Save files
	for path, content := range files {
		pd.files[path] = content
	}
	return nil
}

// GetFiles returns files with given paths from the GitLab project.
func (gl *GitLab) GetFiles(projectID string, filePaths ...string) (map[string]string, error) {
	pd, ok := gl.projects[projectID]
	if !ok {
		return nil, errors.Errorf("gitlab project %q doesn't exist", projectID)
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
