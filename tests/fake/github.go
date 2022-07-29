package fake

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/vcs/github"
)

// GitHub is a fake implementation of GitHub VCS provider.
type GitHub struct {
	port int
	echo *echo.Echo

	client *http.Client

	nextWebhookID int
	repositories  map[string]*repositoryData
}

type repositoryData struct {
	webhooks []*github.WebhookCreateOrUpdate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
}

// NewGitHub creates a new fake implementation of GitHub VCS provider.
func NewGitHub(port int) VCSProvider {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	gh := &GitHub{
		port:          port,
		echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		repositories:  make(map[string]*repositoryData),
	}

	g := e.Group("/api/v3")
	g.POST("/repos/:owner/:repo/hooks", gh.createRepositoryWebhook)
	g.GET("/repos/:owner/:repo/git/commits/:commitID", gh.getRepositoryCommit)
	g.GET("/repos/:owner/:repo/git/trees/:ref", gh.getRepositoryTree)
	g.GET("/repos/:owner/:repo/contents/:filePath", gh.readRepositoryFile)
	g.PUT("/repos/:owner/:repo/contents/:filePath", gh.createRepositoryFile)
	return gh
}

func (gh *GitHub) createRepositoryWebhook(c echo.Context) error {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating repository webhook: %v", err))
	}

	var webhookCreate github.WebhookCreateOrUpdate
	if err = json.Unmarshal(body, &webhookCreate); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating repository webhook: %v", err))
	}
	r.webhooks = append(r.webhooks, &webhookCreate)

	buf, err := json.Marshal(github.WebhookInfo{ID: gh.nextWebhookID})
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for creating repository webhook: %v", err))
	}
	gh.nextWebhookID++
	return c.String(http.StatusOK, string(buf))
}

func (gh *GitHub) getRepositoryCommit(c echo.Context) error {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	if _, ok := gh.repositories[repositoryID]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	buf, err := json.Marshal(
		github.Commit{
			SHA: "fake_github_commit_sha",
			Author: github.CommitAuthor{
				Date: time.Now(),
				Name: "fake_github_author",
			},
		},
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository commit: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gh *GitHub) getRepositoryTree(c echo.Context) error {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	var treeNodes []github.RepositoryTreeNode
	for filePath := range r.files {
		treeNodes = append(treeNodes,
			github.RepositoryTreeNode{
				Path: filePath,
				Type: "blob",
			},
		)
	}
	buf, err := json.Marshal(
		github.RepositoryTree{
			Tree: treeNodes,
		},
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository tree: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gh *GitHub) readRepositoryFile(c echo.Context) error {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unescape file path %q: %v", filePathEscaped, err))
	}

	content, ok := r.files[filePath]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}

	buf, err := json.Marshal(
		github.File{
			Encoding: "base64",
			Size:     int64(len(content)),
			Name:     path.Base(filePath),
			Path:     filePath,
			Content:  base64.StdEncoding.EncodeToString([]byte(content)),
			SHA:      "fake_github_commit_sha",
		},
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository file: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gh *GitHub) createRepositoryFile(c echo.Context) error {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unescape file path %q: %v", filePathEscaped, err))
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating repository file: %v", err))
	}

	var fileCommit github.FileCommit
	if err = json.Unmarshal(body, &fileCommit); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating repository webhook: %v", err))
	}

	content, err := base64.StdEncoding.DecodeString(fileCommit.Content)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to decode file content for %q: %v", filePathEscaped, err))
	}
	r.files[filePath] = string(content)
	return nil
}

// Run starts the GitHub VCS provider server.
func (gh *GitHub) Run() error {
	return gh.echo.Start(fmt.Sprintf(":%d", gh.port))
}

// Close shuts down the GitHub VCS provider server.
func (gh *GitHub) Close() error {
	return gh.echo.Close()
}

// ListenerAddr returns the GitHub VCS provider server listener address.
func (gh *GitHub) ListenerAddr() net.Addr {
	return gh.echo.ListenerAddr()
}

// APIURL returns the GitHub VCS provider API URL.
func (*GitHub) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/api/v3", instanceURL)
}

// CreateRepository creates a GitHub repository with given ID.
func (gh *GitHub) CreateRepository(id string) {
	gh.repositories[id] = &repositoryData{
		files: make(map[string]string),
	}
}

// SendWebhookPush sends out a webhook for a push event for the GitHub
// repository using given payload.
func (gh *GitHub) SendWebhookPush(repositoryID string, payload []byte) error {
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return errors.Errorf("GitHub repository %q does not exist", repositoryID)
	}

	// Trigger all webhooks
	for _, webhook := range r.webhooks {
		req, err := http.NewRequest("POST", webhook.Config.URL, bytes.NewReader(payload))
		if err != nil {
			return errors.Wrapf(err, "failed to create a new POST request to %q", webhook.Config.URL)
		}

		m := hmac.New(sha256.New, []byte(webhook.Config.Secret))
		m.Write(payload)
		signature := "sha256=" + hex.EncodeToString(m.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", signature)
		req.Header.Set("X-GitHub-Event", "push")

		resp, err := gh.client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "failed to send POST request to %q", webhook.Config.URL)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read response body")
		}
		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("unexpected response status code %d, body: %s", resp.StatusCode, body)
		}
		gh.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
	}
	return nil
}

// AddFiles adds given files to the GitHub repository.
func (gh *GitHub) AddFiles(repositoryID string, files map[string]string) error {
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return errors.Errorf("GitHub repository %q does not exist", repositoryID)
	}

	// Save or overwrite files
	for path, content := range files {
		r.files[path] = content
	}
	return nil
}

// GetFiles returns files with given paths from the GitHub repository.
func (gh *GitHub) GetFiles(repositoryID string, filePaths ...string) (map[string]string, error) {
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return nil, errors.Errorf("GitHub repository %q does not exist", repositoryID)
	}

	// Get files
	files := make(map[string]string)
	for _, path := range filePaths {
		if content, ok := r.files[path]; ok {
			files[path] = content
		}
	}
	return files, nil
}
