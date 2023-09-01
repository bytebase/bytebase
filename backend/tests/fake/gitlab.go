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
	"strings"
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
	webhooks []*gitlab.WebhookCreate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// branches is the map for project branch.
	// the map key is the project name, like "refs/heads/main".
	branches map[string]*gitlab.Branch
	// variables is the map for project environment variable.
	// the map key is the variable name, like "public-key".
	variables map[string]*gitlab.EnvironmentVariable
	// mergeRequests is the map for project merge request.
	// the map key is the merge request id.
	mergeRequests map[int]struct {
		*gitlab.MergeRequestChange
		*gitlab.MergeRequest
	}
	// commitsDiff is the map for commits compare.
	// The map key has the format "from...to" which is SHA or branch name.
	commitsDiff map[string]*gitlab.CommitsDiff
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
	projectGroup.POST("/projects/:id/hooks", gl.createProjectHook)
	projectGroup.GET("/projects/:id/repository/commits/:commitID", gl.getFakeCommit)
	projectGroup.GET("/projects/:id/repository/tree", gl.readProjectTree)
	projectGroup.GET("/projects/:id/repository/files/:filePath/raw", gl.readProjectFile)
	projectGroup.POST("/projects/:id/repository/files/:filePath", gl.createProjectFile)
	projectGroup.PUT("/projects/:id/repository/files/:filePath", gl.createProjectFile)
	projectGroup.GET("/projects/:id/repository/branches/:branchName", gl.getProjectBranch)
	projectGroup.POST("/projects/:id/repository/branches", gl.createProjectBranch)
	projectGroup.POST("/projects/:id/merge_requests", gl.createProjectPullRequest)
	projectGroup.GET("/projects/:id/variables/:variableKey", gl.getProjectEnvironmentVariable)
	projectGroup.POST("/projects/:id/variables", gl.createProjectEnvironmentVariable)
	projectGroup.PUT("/projects/:id/variables/:variableKey", gl.updateProjectEnvironmentVariable)
	projectGroup.GET("/projects/:id/merge_requests/:mrID/changes", gl.getMergeRequestChanges)
	projectGroup.GET("/projects/:id/repository/compare", gl.compareCommits)

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
		files:     map[string]string{},
		branches:  map[string]*gitlab.Branch{},
		variables: map[string]*gitlab.EnvironmentVariable{},
		mergeRequests: map[int]struct {
			*gitlab.MergeRequestChange
			*gitlab.MergeRequest
		}{},
		commitsDiff: map[string]*gitlab.CommitsDiff{},
	}
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

// readProjectTree reads a project file nodes.
func (gl *GitLab) readProjectTree(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	path := c.QueryParam("path")
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
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal fileNodes, error: %v", err))
	}

	return c.String(http.StatusOK, string(buf))
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

// createProjectFile creates a project file.
func (gl *GitLab) createProjectFile(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	filePathEscaped := c.Param("filePath")
	filePath, err := url.QueryUnescape(filePathEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to query unescape %q, error: %v", filePathEscaped, err))
	}
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read create project file request body, error: %v", err))
	}
	fileCommit := &gitlab.FileCommit{}
	if err := json.Unmarshal(b, fileCommit); err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal create project file request body, error: %v", err))
	}

	// Save file.
	pd.files[filePath] = fileCommit.Content

	return c.String(http.StatusOK, "")
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

func (gl *GitLab) getProjectEnvironmentVariable(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	variableKey := c.Param("variableKey")
	variable, ok := pd.variables[variableKey]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("failed to find variable %s", variableKey))
	}

	buf, err := json.Marshal(variable)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for getting project environment variable: %v", err))
	}
	return c.String(http.StatusOK, string(buf))
}

func (gl *GitLab) createProjectEnvironmentVariable(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating project environment variable: %v", err))
	}

	var environmentVariable gitlab.EnvironmentVariable
	if err = json.Unmarshal(body, &environmentVariable); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating project environment variable: %v", err))
	}

	if _, ok := pd.variables[environmentVariable.Key]; ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("environment variable already exists %s", environmentVariable.Key))
	}

	pd.variables[environmentVariable.Key] = &environmentVariable
	return c.String(http.StatusOK, "")
}

func (gl *GitLab) updateProjectEnvironmentVariable(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating project environment variable: %v", err))
	}

	var environmentVariable gitlab.EnvironmentVariable
	if err = json.Unmarshal(body, &environmentVariable); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating project environment variable: %v", err))
	}

	variableKey := c.Param("variableKey")
	if _, ok := pd.variables[variableKey]; !ok {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to find environment variable %s", variableKey))
	}
	if variableKey != environmentVariable.Key {
		return c.String(http.StatusBadRequest, fmt.Sprintf("invalid environment variable %s", environmentVariable.Key))
	}

	pd.variables[environmentVariable.Key] = &environmentVariable
	return c.String(http.StatusOK, "")
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

func (gl *GitLab) compareCommits(c echo.Context) error {
	pd, err := gl.validProject(c)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s...%s", c.QueryParam("from"), c.QueryParam("to"))
	diff, ok := pd.commitsDiff[key]
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot find the diff key %s", key))
	}
	buf, err := json.Marshal(diff)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal response body").SetInternal(err)
	}
	return c.String(http.StatusOK, string(buf))
}

// AddCommitsDiff adds a commits diff.
func (gl *GitLab) AddCommitsDiff(projectID, fromCommit, toCommit string, fileDiffList []vcs.FileDiff) error {
	pd, ok := gl.projects[projectID]
	if !ok {
		return errors.Errorf("GitLab project %s doesn't exist", projectID)
	}
	key := fmt.Sprintf("%s...%s", fromCommit, toCommit)
	commitsDiff := &gitlab.CommitsDiff{
		FileDiffList: []gitlab.MergeRequestFile{},
	}
	for _, fileDiff := range fileDiffList {
		mrFile := gitlab.MergeRequestFile{
			NewPath: fileDiff.Path,
		}
		switch fileDiff.Type {
		case vcs.FileDiffTypeAdded:
			mrFile.NewFile = true
		case vcs.FileDiffTypeRemoved:
			mrFile.DeletedFile = true
		}
		commitsDiff.FileDiffList = append(commitsDiff.FileDiffList, mrFile)
	}
	pd.commitsDiff[key] = commitsDiff
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
