package fake

import (
	"bytes"
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
	"github.com/bytebase/bytebase/backend/plugin/vcs/bitbucket"
)

// Bitbucket is a fake implementation of Bitbucket VCS provider.
type Bitbucket struct {
	port int
	echo *echo.Echo

	client *http.Client

	nextWebhookID int
	repositories  map[string]*bitbucketRepositoryData
}

type bitbucketRepositoryData struct {
	repository *bitbucket.Repository
	webhooks   []*bitbucket.WebhookCreateOrUpdate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// refs is the map for repository branch. The map key is the branch ref, like
	// "refs/heads/main".
	refs map[string]*bitbucket.Branch
	// pullRequests is the map for repository pull request.
	// the map key is the pull request id.
	pullRequests map[int]struct {
		Files []*bitbucket.CommitDiffStat
	}
}

// NewBitbucket creates a new fake implementation of Bitbucket VCS provider.
func NewBitbucket(port int) VCSProvider {
	e := newEchoServer()
	bb := &Bitbucket{
		port:          port,
		echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		repositories:  make(map[string]*bitbucketRepositoryData),
	}

	g := e.Group("/2.0")
	g.GET("/user/permissions/repositories", bb.listRepositories)
	g.POST("/repositories/:owner/:repo/hooks", bb.createRepositoryWebhook)
	g.DELETE("/repositories/:owner/:repo/hooks/:hook", bb.deleteRepositoryWebhook)
	g.GET("/repositories/:owner/:repo/commit/:commitID", bb.getRepositoryCommit)
	g.GET("/repositories/:owner/:repo/src/:ref/:filepath", bb.getRepositoryContent)
	g.GET("/repositories/:owner/:repo/refs/branches/:branchName", bb.getRepositoryBranch)
	g.GET("/repositories/:owner/:repo/pullrequests/:prID/diffstat", bb.listPullRequestFile)
	g.GET("/repositories/:owner/:repo/pullrequests/:prID/comments", bb.createPullRequestComment)
	return bb
}

func (bb *Bitbucket) listRepositories(c echo.Context) error {
	var resp struct {
		Values []*bitbucket.RepositoryPermission `json:"values"`
		Next   string                            `json:"next"`
	}

	for _, repoData := range bb.repositories {
		resp.Values = append(resp.Values, &bitbucket.RepositoryPermission{
			Repository: repoData.repository,
		})
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for list repository: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*Bitbucket) deleteRepositoryWebhook(c echo.Context) error {
	return c.String(http.StatusOK, "")
}

func (bb *Bitbucket) createRepositoryWebhook(c echo.Context) error {
	r, err := bb.validRepository(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating repository webhook: %v", err))
	}

	var webhookCreate bitbucket.WebhookCreateOrUpdate
	if err = json.Unmarshal(body, &webhookCreate); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating repository webhook: %v", err))
	}
	r.webhooks = append(r.webhooks, &webhookCreate)

	buf, err := json.Marshal(bitbucket.Webhook{UUID: strconv.Itoa(bb.nextWebhookID)})
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for creating repository webhook: %v", err))
	}
	bb.nextWebhookID++
	return c.String(http.StatusCreated, string(buf))
}

func (bb *Bitbucket) getRepositoryCommit(c echo.Context) error {
	if _, err := bb.validRepository(c); err != nil {
		return err
	}

	buf, err := json.Marshal(
		bitbucket.Commit{
			Hash: "fake_bitbucket_commit_sha",
			Author: bitbucket.CommitAuthor{
				User: bitbucket.User{
					Nickname: "fake_bitbucket_author",
				},
			},
			Date: time.Now(),
		},
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository commit: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (bb *Bitbucket) getRepositoryContent(c echo.Context) error {
	r, err := bb.validRepository(c)
	if err != nil {
		return err
	}

	filePathEscaped := c.Param("filepath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unescape file path %q: %v", filePathEscaped, err))
	}

	content, ok := r.files[filePath]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}
	return c.String(http.StatusOK, content)
}

func (bb *Bitbucket) getRepositoryBranch(c echo.Context) error {
	r, err := bb.validRepository(c)
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

func (bb *Bitbucket) validRepository(c echo.Context) (*bitbucketRepositoryData, error) {
	repositoryID := fmt.Sprintf("%s/%s", c.Param("owner"), c.Param("repo"))
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return nil, c.String(http.StatusNotFound, fmt.Sprintf("Bitbucket repository %q does not exist", repositoryID))
	}

	return r, nil
}

// Run starts the Bitbucket VCS provider server.
func (bb *Bitbucket) Run() error {
	return bb.echo.Start(fmt.Sprintf(":%d", bb.port))
}

// Close shuts down the Bitbucket VCS provider server.
func (bb *Bitbucket) Close() error {
	return bb.echo.Close()
}

// ListenerAddr returns the Bitbucket VCS provider server listener address.
func (bb *Bitbucket) ListenerAddr() net.Addr {
	return bb.echo.ListenerAddr()
}

// CreateRepository creates a Bitbucket repository with given ID.
func (bb *Bitbucket) CreateRepository(repository *vcs.Repository) error {
	bb.repositories[repository.ID] = &bitbucketRepositoryData{
		repository: &bitbucket.Repository{
			UUID:     repository.ID,
			Name:     repository.Name,
			FullName: repository.FullPath,
		},
		files: make(map[string]string),
		refs:  map[string]*bitbucket.Branch{},
		pullRequests: map[int]struct {
			Files []*bitbucket.CommitDiffStat
		}{},
	}
	return nil
}

// CreateBranch creates a new branch with the given name.
func (bb *Bitbucket) CreateBranch(id, branchName string) error {
	pd, ok := bb.repositories[id]
	if !ok {
		return errors.Errorf("bitbucket project %q doesn't exist", id)
	}

	if _, ok := pd.refs[fmt.Sprintf("refs/heads/%s", branchName)]; ok {
		return errors.Errorf("branch %q already exists", branchName)
	}

	pd.refs[fmt.Sprintf("refs/heads/%s", branchName)] = &bitbucket.Branch{
		Name: branchName,
		Target: bitbucket.Target{
			Hash: "fake_bitbucket_commit_sha",
		},
	}
	return nil
}

// SendWebhookPush sends out a webhook for a push event for the Bitbucket
// repository using given payload.
func (bb *Bitbucket) SendWebhookPush(repositoryID string, payload []byte) error {
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Bitbucket repository %q does not exist", repositoryID)
	}

	// Trigger all webhooks
	for _, webhook := range r.webhooks {
		if err := func() error {
			req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payload))
			if err != nil {
				return errors.Wrapf(err, "failed to create a new POST request to %q", webhook.URL)
			}
			req.Header.Set("X-Event-Key", "pullrequest:fulfilled")
			resp, err := bb.client.Do(req)
			if err != nil {
				return errors.Wrapf(err, "failed to send POST request to %q", webhook.URL)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read response body")
			}
			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("unexpected response status code %d, body: %s", resp.StatusCode, body)
			}
			bb.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
			return nil
		}(); err != nil {
			return err
		}
	}
	return nil
}

// AddFiles adds given files to the Bitbucket repository.
func (bb *Bitbucket) AddFiles(repositoryID string, files map[string]string) error {
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Bitbucket repository %q does not exist", repositoryID)
	}

	// Save or overwrite files
	for path, content := range files {
		r.files[path] = content
	}
	return nil
}

// AddPullRequest creates a new pull request and add changed files to it.
func (bb *Bitbucket) AddPullRequest(repositoryID string, prID int, files []*vcs.PullRequestFile) error {
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Bitbucket repository %q does not exist", repositoryID)
	}
	var pullRequestFiles []*bitbucket.CommitDiffStat
	for _, file := range files {
		status := "added"
		if file.IsDeleted {
			status = "removed"
		}
		pullRequestFiles = append(pullRequestFiles, &bitbucket.CommitDiffStat{
			Status: status,
			New: bitbucket.CommitFile{
				Path: file.Path,
			},
		})
	}

	r.pullRequests[prID] = struct {
		Files []*bitbucket.CommitDiffStat
	}{
		Files: pullRequestFiles,
	}
	return nil
}

func (bb *Bitbucket) listPullRequestFile(c echo.Context) error {
	r, err := bb.validRepository(c)
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

	rep := bitbucket.PullRequestResponse{
		Values: pullRequest.Files,
	}

	buf, err := json.Marshal(rep)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*Bitbucket) createPullRequestComment(echo.Context) error {
	return nil
}
