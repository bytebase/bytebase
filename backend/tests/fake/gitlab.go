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
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/gitlab"
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
	repository *gitlab.Repository
	webhooks   []*gitlab.WebhookCreate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// branches is the map for project branch.
	// the map key is the project name, like "refs/heads/main".
	branches map[string]*gitlab.Branch
	// mergeRequests is the map for project merge request.
	// the map key is the merge request id.
	mergeRequests map[int]struct {
		*gitlab.MergeRequestChange
		*gitlab.MergeRequest
	}
}

// NewGitLab creates a new fake implementation of GitLab VCS provider.
func NewGitLab(port int) VCSProvider {
	e := newEchoServer()
	gl := &GitLab{
		port:          port,
		echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		projects:      map[string]*projectData{},
	}

	// Routes
	projectGroup := e.Group("/api/v4")
	projectGroup.GET("/projects", gl.listRepositories)
	projectGroup.POST("/projects/:id/hooks", gl.createProjectHook)
	projectGroup.DELETE("/projects/:id/hooks/:hook", gl.deleteRepositoryWebhook)
	projectGroup.GET("/projects/:id/repository/commits/:commitID", gl.getFakeCommit)
	projectGroup.GET("/projects/:id/repository/files/:filePath/raw", gl.readProjectFile)
	projectGroup.GET("/projects/:id/repository/branches/:branchName", gl.getProjectBranch)
	projectGroup.POST("/projects/:id/repository/branches", gl.createProjectBranch)
	projectGroup.POST("/projects/:id/merge_requests", gl.createProjectPullRequest)
	projectGroup.GET("/projects/:id/merge_requests/:mrID/changes", gl.getMergeRequestChanges)
	projectGroup.GET("/projects/:id/merge_requests/:mrID/notes", gl.createMergeRequestComment)

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

// CreateRepository creates a GitLab project with given ID.
func (gl *GitLab) CreateRepository(repository *vcs.Repository) error {
	// GitLab repo id should be int64.
	id, err := strconv.ParseInt(repository.ID, 10, 64)
	if err != nil {
		return err
	}
	gl.projects[repository.ID] = &projectData{
		repository: &gitlab.Repository{
			ID:                id,
			Name:              repository.Name,
			PathWithNamespace: repository.FullPath,
		},
		files:    map[string]string{},
		branches: map[string]*gitlab.Branch{},
		mergeRequests: map[int]struct {
			*gitlab.MergeRequestChange
			*gitlab.MergeRequest
		}{},
	}
	return nil
}

// CreateBranch creates a new branch with the given name.
func (gl *GitLab) CreateBranch(id, branchName string) error {
	pd, ok := gl.projects[id]
	if !ok {
		return errors.Errorf("gitlab project %q doesn't exist", id)
	}

	if _, ok := pd.branches[fmt.Sprintf("refs/heads/%s", branchName)]; ok {
		return errors.Errorf("branch %q already exists", branchName)
	}

	pd.branches[fmt.Sprintf("refs/heads/%s", branchName)] = &gitlab.Branch{
		Name: branchName,
		Commit: gitlab.Commit{
			ID:         "fake_gitlab_commit_id",
			AuthorName: "fake_gitlab_bot",
			CreatedAt:  time.Now(),
		},
	}
	return nil
}

func (gl *GitLab) listRepositories(c echo.Context) error {
	repoList := []*gitlab.Repository{}
	for _, repoData := range gl.projects {
		repoList = append(repoList, repoData.repository)
	}
	buf, err := json.Marshal(repoList)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for list repository: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*GitLab) deleteRepositoryWebhook(c echo.Context) error {
	return c.String(http.StatusOK, "")
}

// createProjectHook creates a project webhook.
func (gl *GitLab) createProjectHook(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	c.Logger().Infof("Create webhook for project %q", c.Param("id"))
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read create project hook request body")
	}
	webhookCreate := &gitlab.WebhookCreate{}
	if err := json.Unmarshal(b, webhookCreate); err != nil {
		return errors.Wrap(err, "failed to unmarshal create project hook request body")
	}

	pd.webhooks = append(pd.webhooks, webhookCreate)

	gl.nextWebhookID++

	return c.JSON(http.StatusCreated, &gitlab.WebhookInfo{
		ID: gl.nextWebhookID,
	})
}

// readProjectFile reads a project file.
func (gl *GitLab) readProjectFile(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}

	fileName := filepath.Base(filePath)

	content, ok := pd.files[filePath]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("file %q not found", filePath))
	}

	c.Response().Header().Set("x-gitlab-file-name", fileName)
	c.Response().Header().Set("x-gitlab-file-path", filePath)
	c.Response().Header().Set("x-gitlab-size", strconv.Itoa(len([]byte(content))))
	c.Response().Header().Set("x-gitlab-last-commit-id", "fake_gitlab_commit_id")

	return c.String(http.StatusOK, content)
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
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal commit, error: %v", err))
	}

	return c.String(http.StatusOK, string(buf))
}

func (gl *GitLab) validProject(c echo.Context) (*projectData, error) {
	projectID := c.Param("id")
	project, ok := gl.projects[projectID]
	if !ok {
		return nil, c.String(http.StatusBadRequest, fmt.Sprintf("gitlab project %q doesn't exist", projectID))
	}

	return project, nil
}

func (gl *GitLab) getProjectBranch(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	branchName := c.Param("branchName")
	if _, ok := pd.branches[fmt.Sprintf("refs/heads/%s", branchName)]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("branch not found: %v", branchName))
	}

	buf, err := json.Marshal(pd.branches[fmt.Sprintf("refs/heads/%s", branchName)])
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting project branch: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gl *GitLab) createProjectBranch(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating project branch: %v", err))
	}

	var branchCreate gitlab.BranchCreate
	if err = json.Unmarshal(body, &branchCreate); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating project branch: %v", err))
	}

	if _, ok := pd.branches[branchCreate.Branch]; ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("the branch already exists: %v", branchCreate.Ref))
	}

	pd.branches[branchCreate.Branch] = &gitlab.Branch{
		Name: branchCreate.Branch,
		Commit: gitlab.Commit{
			ID:         branchCreate.Ref,
			AuthorName: "fake_gitlab_bot",
			CreatedAt:  time.Now(),
		},
	}

	return c.String(http.StatusOK, "")
}

func (gl *GitLab) createProjectPullRequest(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating project merge request: %v", err))
	}

	var mergeRequestCreate gitlab.MergeRequestCreate
	if err = json.Unmarshal(body, &mergeRequestCreate); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating project merge request: %v", err))
	}

	if _, ok := pd.branches[mergeRequestCreate.SourceBranch]; !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("the source branch not exists: %v", mergeRequestCreate.SourceBranch))
	}

	mrID := len(pd.mergeRequests) + 1
	pd.mergeRequests[mrID] = struct {
		*gitlab.MergeRequestChange
		*gitlab.MergeRequest
	}{
		MergeRequestChange: &gitlab.MergeRequestChange{},
		MergeRequest: &gitlab.MergeRequest{
			// TODO: the URL for merge request is invalid.
			WebURL: fmt.Sprintf("http://gitlab.example.com/my-group/my-project/merge_requests/%d", mrID),
		},
	}

	buf, err := json.Marshal(
		pd.mergeRequests[mrID],
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for creating project merge request: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gl *GitLab) getMergeRequestChanges(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	mrNumber, err := strconv.Atoi(c.Param("mrID"))
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("The merge request id is invalid: %v", c.Param("mrID")))
	}

	mergeRequest, ok := pd.mergeRequests[mrNumber]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("Cannot find the merge request: %v", c.Param("mrID")))
	}

	buf, err := json.Marshal(mergeRequest.MergeRequestChange)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (*GitLab) createMergeRequestComment(echo.Context) error {
	return nil
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
		if err := func() error {
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
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read http response body")
			}

			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("http response error code %v body %q", resp.StatusCode, string(body))
			}
			gl.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
			return nil
		}(); err != nil {
			return err
		}
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

// AddPullRequest creates a new merge request and add changed files to it.
func (gl *GitLab) AddPullRequest(projectID string, mrID int, files []*vcs.PullRequestFile) error {
	pd, ok := gl.projects[projectID]
	if !ok {
		return errors.Errorf("gitlab project %q doesn't exist", projectID)
	}

	if len(files) == 0 {
		return nil
	}

	changes := []gitlab.MergeRequestFile{}
	for _, file := range files {
		changes = append(changes, gitlab.MergeRequestFile{
			NewPath:     file.Path,
			NewFile:     true,
			RenamedFile: false,
			DeletedFile: file.IsDeleted,
		})
	}

	pd.mergeRequests[mrID] = struct {
		*gitlab.MergeRequestChange
		*gitlab.MergeRequest
	}{
		MergeRequestChange: &gitlab.MergeRequestChange{
			SHA:     files[0].LastCommitID,
			Changes: changes,
		},
		MergeRequest: &gitlab.MergeRequest{
			// TODO: the URL for merge request is invalid.
			WebURL: fmt.Sprintf("http://gitlab.example.com/my-group/my-project/merge_requests/%d", mrID),
		},
	}
	return nil
}
