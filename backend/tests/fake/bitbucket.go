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
	"strings"
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
	webhooks []*bitbucket.WebhookCreateOrUpdate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// refs is the map for repository branch. The map key is the branch ref, like
	// "refs/heads/main".
	refs map[string]*bitbucket.Branch
	// diffStats is the map for commits compare. The map key has the format
	// "to..from" which is SHA or branch name.
	diffStats map[string][]*bitbucket.CommitDiffStat
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
	g.POST("/repositories/:owner/:repo/hooks", bb.createRepositoryWebhook)
	g.GET("/repositories/:owner/:repo/commit/:commitID", bb.getRepositoryCommit)
	g.GET("/repositories/:owner/:repo/src/:ref/:filepath", bb.getRepositoryContent)
	g.POST("/repositories/:owner/:repo/src", bb.createRepositoryFile)
	g.GET("/repositories/:owner/:repo/refs/branches/:branchName", bb.getRepositoryBranch)
	g.GET("/repositories/:owner/:repo/diffstat/:baseHead", bb.compareCommits)
	return bb
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

	if strings.Contains(c.QueryString(), `type="commit_file"`) {
		var treeEntries []*bitbucket.TreeEntry
		for filePath := range r.files {
			treeEntries = append(treeEntries,
				&bitbucket.TreeEntry{
					Type: "blob",
					Path: filePath,
				},
			)
		}
		resp, err := json.Marshal(map[string]any{"values": treeEntries})
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository content: %v", err))
		}
		return c.String(http.StatusOK, string(resp))
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

	if strings.Contains(c.QueryString(), `format=meta`) {
		resp, err := json.Marshal(
			bitbucket.TreeEntry{
				Type: "commit_file",
				Path: filePath,
				Size: int64(len(content)),
			},
		)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting repository file meta: %v", err))
		}
		return c.String(http.StatusOK, string(resp))
	}
	return c.String(http.StatusOK, content)
}

func (bb *Bitbucket) createRepositoryFile(c echo.Context) error {
	r, err := bb.validRepository(c)
	if err != nil {
		return err
	}

	params, err := c.FormParams()
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to get form params: %v", err))
	}

	for k, v := range params {
		if k != "parents" && k != "branch" && k != "message" {
			r.files[k] = v[0]
		}
	}
	return nil
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

func (bb *Bitbucket) compareCommits(c echo.Context) error {
	r, err := bb.validRepository(c)
	if err != nil {
		return err
	}

	key := c.Param("baseHead")
	diff, ok := r.diffStats[key]
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot find the diff key %s", key))
	}

	buf, err := json.Marshal(map[string]any{"values": diff})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal response body").SetInternal(err)
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

// APIURL returns the Bitbucket VCS provider API URL.
func (*Bitbucket) APIURL(string) string {
	return "https://api.bitbucket.org/2.0"
}

// CreateRepository creates a Bitbucket repository with given ID.
func (bb *Bitbucket) CreateRepository(id string) {
	bb.repositories[id] = &bitbucketRepositoryData{
		files:     make(map[string]string),
		refs:      map[string]*bitbucket.Branch{},
		diffStats: map[string][]*bitbucket.CommitDiffStat{},
	}
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

// AddCommitsDiff adds a commits diff.
func (bb *Bitbucket) AddCommitsDiff(repositoryID, fromCommit, toCommit string, fileDiffList []vcs.FileDiff) error {
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Bitbucket repository %s doesn't exist", repositoryID)
	}
	key := fmt.Sprintf("%s..%s", toCommit, fromCommit)
	var diffStats []*bitbucket.CommitDiffStat
	for _, fileDiff := range fileDiffList {
		diffStat := &bitbucket.CommitDiffStat{
			New: bitbucket.CommitFile{
				Path: fileDiff.Path,
			},
		}
		switch fileDiff.Type {
		case vcs.FileDiffTypeAdded:
			diffStat.Status = "added"
		case vcs.FileDiffTypeModified:
			diffStat.Status = "modified"
		case vcs.FileDiffTypeRemoved:
			diffStat.Status = "removed"
		}
		diffStats = append(diffStats, diffStat)
	}
	r.diffStats[key] = diffStats
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
			req.Header.Set("X-Event-Key", "repo:push")
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

// GetFiles returns files with given paths from the Bitbucket repository.
func (bb *Bitbucket) GetFiles(repositoryID string, filePaths ...string) (map[string]string, error) {
	r, ok := bb.repositories[repositoryID]
	if !ok {
		return nil, errors.Errorf("Bitbucket repository %q does not exist", repositoryID)
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

// AddPullRequest creates a new pull request and add changed files to it.
func (*Bitbucket) AddPullRequest(string, int, []*vcs.PullRequestFile) error {
	return errors.New("not implemented yet")
}
