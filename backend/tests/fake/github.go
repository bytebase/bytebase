package fake

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/github"
)

// GitHub is a fake implementation of GitHub VCS provider.
type GitHub struct {
	port int
	echo *echo.Echo

	client *http.Client

	nextWebhookID int
	repositories  map[string]*githubRepositoryData
}

type githubRepositoryData struct {
	repository *github.Repository
	webhooks   []*github.WebhookCreateOrUpdate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// refs is the map for repository branch.
	// the map key is the branch ref, like "refs/heads/main".
	refs map[string]*github.Branch
	// pullRequests is the map for repository pull request.
	// the map key is the pull request id.
	pullRequests map[int]struct {
		Files []*github.PullRequestFile
		*github.PullRequest
	}
}

// NewGitHub creates a new fake implementation of GitHub VCS provider.
func NewGitHub(port int) VCSProvider {
	e := newEchoServer()
	gh := &GitHub{
		port:          port,
		echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		repositories:  make(map[string]*githubRepositoryData),
	}

	g := e.Group("/api/v3")
	g.GET("/user/repos", gh.listRepositories)
	g.POST("/repos/:owner/:repo/hooks", gh.createRepositoryWebhook)
	g.DELETE("/repos/:owner/:repo/hooks/:hook", gh.deleteRepositoryWebhook)
	g.GET("/repos/:owner/:repo/git/commits/:commitID", gh.getRepositoryCommit)
	g.GET("/repos/:owner/:repo/contents/:filePath", gh.readRepositoryFile)
	g.GET("/repos/:owner/:repo/git/ref/heads/:branchName", gh.getRepositoryBranch)
	g.GET("/repos/:owner/:repo/pulls/:prID/files", gh.listPullRequestFile)
	g.POST("/repos/:owner/:repo/issues/:prID/comments", gh.createIssueComment)
	return gh
}

func (gh *GitHub) listRepositories(c echo.Context) error {
	repoList := []*github.Repository{}
	for _, repoData := range gh.repositories {
		repoList = append(repoList, repoData.repository)
	}
	buf, err := json.Marshal(repoList)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for list repository: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*GitHub) deleteRepositoryWebhook(c echo.Context) error {
	return c.String(http.StatusOK, "")
}

func (gh *GitHub) createRepositoryWebhook(c echo.Context) error {
	r, err := gh.validRepository(c)
	if err != nil {
		return err
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
	return c.String(http.StatusCreated, string(buf))
}

func (gh *GitHub) getRepositoryCommit(c echo.Context) error {
	if _, err := gh.validRepository(c); err != nil {
		return err
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

func (gh *GitHub) readRepositoryFile(c echo.Context) error {
	r, err := gh.validRepository(c)
	if err != nil {
		return err
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

	if c.Request().Header.Get("Accept") == "application/vnd.github.raw" {
		return c.String(http.StatusOK, content)
	}

	return c.String(http.StatusBadRequest, "must accept vnd.github.raw")
}

func (gh *GitHub) getRepositoryBranch(c echo.Context) error {
	r, err := gh.validRepository(c)
	if err != nil {
		return err
	}

	branchName := c.Param("branchName")
	if _, ok := r.refs[fmt.Sprintf("refs/heads/%s", branchName)]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("branch not found: %v", branchName))
	}

	buf, err := json.Marshal(r.refs[fmt.Sprintf("refs/heads/%s", branchName)])
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository branch: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gh *GitHub) listPullRequestFile(c echo.Context) error {
	r, err := gh.validRepository(c)
	if err != nil {
		return err
	}

	prNumber, err := strconv.Atoi(c.Param("prID"))
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("The pull request id is invalid: %v", c.Param("prID")))
	}

	pullRequest, ok := r.pullRequests[prNumber]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("Cannot found the pull request: %v", c.Param("prID")))
	}

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("Invalid page parameter %v", c.Param("page")))
	}

	prFiles := []*github.PullRequestFile{}
	if page == 1 {
		prFiles = pullRequest.Files
	}

	buf, err := json.Marshal(prFiles)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*GitHub) createIssueComment(echo.Context) error {
	return nil
}

func (gh *GitHub) validRepository(c echo.Context) (*githubRepositoryData, error) {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return nil, c.String(http.StatusNotFound, fmt.Sprintf("GitHub repository %q does not exist", repositoryID))
	}

	return r, nil
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

// CreateRepository creates a GitHub repository with given ID.
func (gh *GitHub) CreateRepository(repository *vcs.Repository) error {
	gh.repositories[repository.FullPath] = &githubRepositoryData{
		repository: &github.Repository{
			ID:       int64(len(gh.repositories) + 1),
			Name:     repository.Name,
			FullName: repository.FullPath,
		},
		files: make(map[string]string),
		refs:  map[string]*github.Branch{},
		pullRequests: map[int]struct {
			Files []*github.PullRequestFile
			*github.PullRequest
		}{},
	}
	return nil
}

// CreateBranch creates a new branch with the given name.
func (gh *GitHub) CreateBranch(id, branchName string) error {
	pd, ok := gh.repositories[id]
	if !ok {
		return errors.Errorf("github project %q doesn't exist", id)
	}

	if _, ok := pd.refs[fmt.Sprintf("refs/heads/%s", branchName)]; ok {
		return errors.Errorf("branch %q already exists", branchName)
	}

	pd.refs[fmt.Sprintf("refs/heads/%s", branchName)] = &github.Branch{
		Ref: fmt.Sprintf("refs/heads/%s", branchName),
		Object: github.ReferenceObject{
			SHA: "fake_github_commit_sha",
		},
	}

	return nil
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
		if err := func() error {
			req, err := http.NewRequest("POST", webhook.Config.URL, bytes.NewReader(payload))
			if err != nil {
				return errors.Wrapf(err, "failed to create a new POST request to %q", webhook.Config.URL)
			}

			m := hmac.New(sha256.New, []byte(webhook.Config.Secret))
			if _, err := m.Write(payload); err != nil {
				return errors.Wrap(err, "failed to calculate SHA256 of the webhook secret")
			}
			signature := "sha256=" + hex.EncodeToString(m.Sum(nil))
			req.Header.Set("X-Hub-Signature-256", signature)
			req.Header.Set("X-GitHub-Event", "pull_request")

			resp, err := gh.client.Do(req)
			if err != nil {
				return errors.Wrapf(err, "failed to send POST request to %q", webhook.Config.URL)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read response body")
			}
			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("unexpected response status code %d, body: %s", resp.StatusCode, body)
			}
			gh.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
			return nil
		}(); err != nil {
			return err
		}
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

// AddPullRequest creates a new pull request and add changed files to it.
func (gh *GitHub) AddPullRequest(repositoryID string, prID int, files []*vcs.PullRequestFile) error {
	r, ok := gh.repositories[repositoryID]
	if !ok {
		return errors.Errorf("github repository %q does not exist", repositoryID)
	}

	pullRequestFiles := []*github.PullRequestFile{}
	for _, file := range files {
		status := ""
		if file.IsDeleted {
			status = "removed"
		}
		pullRequestFiles = append(pullRequestFiles, &github.PullRequestFile{
			FileName:    file.Path,
			Status:      status,
			SHA:         file.LastCommitID,
			ContentsURL: fmt.Sprintf("https://github.com/%s/%s?ref=%s", repositoryID, url.QueryEscape(file.Path), file.LastCommitID),
		})
	}

	r.pullRequests[prID] = struct {
		Files []*github.PullRequestFile
		*github.PullRequest
	}{
		Files: pullRequestFiles,
		PullRequest: &github.PullRequest{
			HTMLURL: fmt.Sprintf("https://github.com/%s/pull/%d", repositoryID, prID),
		},
	}

	return nil
}
