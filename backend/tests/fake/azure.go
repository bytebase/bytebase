package fake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/azure"
)

// Azure is a fake implementation of Azure VCS provider.
type Azure struct {
	port int
	echo *echo.Echo

	client *http.Client

	nextWebhookID int
	repositories  map[string]*azureRepositoryData
}

type azureRepositoryData struct {
	repository *azure.Repository
	webhooks   []*azure.WebhookCreateOrUpdate
	// files is a map that the full file path is the key and the file content is the
	// value.
	files map[string]string
	// refs is the map for repository branch.
	// the map key is the branch ref, like "refs/heads/main".
	refs    map[string]*azure.Branch
	commits map[string]struct {
		Changes []*azure.CommitChange
	}
}

// NewAzure creates a new fake implementation of Azure VCS provider.
func NewAzure(port int) VCSProvider {
	e := newEchoServer()
	az := &Azure{
		port:          port,
		echo:          e,
		client:        &http.Client{},
		nextWebhookID: 20210113,
		repositories:  make(map[string]*azureRepositoryData),
	}

	g := e.Group("")
	g.GET("/_apis/profile/profiles/me", az.getProfile)
	g.GET("/_apis/accounts", az.listOrgs)
	g.GET("/:organization/_apis/git/repositories", az.listRepositories)
	g.POST("/:organization/_apis/hooks/subscriptions", az.createRepositoryWebhook)
	g.DELETE("/:organization/_apis/hooks/subscriptions/:subscription", az.deleteRepositoryWebhook)

	repo := e.Group("/:organization/:project/_apis/git/repositories/:repo")
	repo.GET("/stats/branches", az.getRepositoryBranch)
	repo.POST("/pullRequests/:pr/threads", az.createIssueComment)
	repo.GET("/commits/:commit/changes", az.getCommitChanges)
	repo.GET("/items", az.readRepositoryFile)

	return az
}

func (*Azure) getResponse(c echo.Context, data any) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusOK, string(buf))
}

func (az *Azure) getProfile(c echo.Context) error {
	resp := &azure.Profile{
		PublicAlias: "mock-az-profile",
	}
	return az.getResponse(c, resp)
}

func (az *Azure) listOrgs(c echo.Context) error {
	type accountsResponse struct {
		Count int                   `json:"count"`
		Value []*azure.Organization `json:"value"`
	}

	resp := &accountsResponse{}

	for name := range az.repositories {
		parts := strings.Split(name, "/")
		if len(parts) != 3 {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("invalid repo name: %s", name))
		}
		resp.Value = append(resp.Value, &azure.Organization{
			AccountName: parts[0],
		})
	}
	resp.Count = len(resp.Value)

	return az.getResponse(c, resp)
}

func (az *Azure) listRepositories(c echo.Context) error {
	type listRepositoriesResponse struct {
		Count int                 `json:"count"`
		Value []*azure.Repository `json:"value"`
	}

	orgID := c.Param("organization")
	resp := &listRepositoriesResponse{}
	for name, item := range az.repositories {
		parts := strings.Split(name, "/")
		if len(parts) != 3 {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("invalid repo name: %s", name))
		}
		if parts[0] != orgID {
			continue
		}
		resp.Value = append(resp.Value, item.repository)
	}
	resp.Count = len(resp.Value)

	return az.getResponse(c, resp)
}

func (az *Azure) getCommitChanges(c echo.Context) error {
	r, err := az.validRepository(c)
	if err != nil {
		return err
	}
	commitID := c.Param("commit")
	if _, ok := r.commits[commitID]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("commit not found: %v", commitID))
	}

	return az.getResponse(c, r.commits[commitID])
}

func (*Azure) deleteRepositoryWebhook(c echo.Context) error {
	return c.String(http.StatusNoContent, "")
}

func (az *Azure) createRepositoryWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to read request body for creating repository webhook: %v", err))
	}

	var webhookCreate azure.WebhookCreateOrUpdate
	if err = json.Unmarshal(body, &webhookCreate); err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshal request body for creating repository webhook: %v", err))
	}

	orgID := c.Param("organization")
	projectID := webhookCreate.PublisherInputs.ProjectID
	repositoryID := webhookCreate.PublisherInputs.Repository
	repositoryName := fmt.Sprintf("%s/%s/%s", orgID, projectID, repositoryID)
	r, ok := az.repositories[repositoryName]
	if !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("Azure repository %q does not exist", repositoryName))
	}

	r.webhooks = append(r.webhooks, &webhookCreate)

	type createServiceResponse struct {
		ID string `json:"id"`
	}
	buf, err := json.Marshal(createServiceResponse{ID: fmt.Sprintf("%d", az.nextWebhookID)})
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal response body for creating repository webhook: %v", err))
	}
	az.nextWebhookID++
	return c.String(http.StatusCreated, string(buf))
}

func (az *Azure) readRepositoryFile(c echo.Context) error {
	r, err := az.validRepository(c)
	if err != nil {
		return err
	}

	filePathEscaped := c.QueryParam("path")
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

func (az *Azure) getRepositoryBranch(c echo.Context) error {
	r, err := az.validRepository(c)
	if err != nil {
		return err
	}

	branchNameEscaped := c.QueryParam("name")
	branchName, err := url.QueryUnescape(branchNameEscaped)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to unescape branch %q: %v", branchNameEscaped, err))
	}
	if _, ok := r.refs[fmt.Sprintf("refs/heads/%s", branchName)]; !ok {
		return c.String(http.StatusNotFound, fmt.Sprintf("branch not found: %v", branchName))
	}

	return az.getResponse(c, r.refs[fmt.Sprintf("refs/heads/%s", branchName)])
}

func (*Azure) createIssueComment(echo.Context) error {
	return nil
}

func (az *Azure) validRepository(c echo.Context) (*azureRepositoryData, error) {
	repositoryID := fmt.Sprintf("%s/%s/%s", c.Param("organization"), c.Param("project"), c.Param("repo"))
	r, ok := az.repositories[repositoryID]
	if !ok {
		return nil, c.String(http.StatusNotFound, fmt.Sprintf("Azure repository %q does not exist", repositoryID))
	}

	return r, nil
}

// Run starts the Azure VCS provider server.
func (az *Azure) Run() error {
	return az.echo.Start(fmt.Sprintf(":%d", az.port))
}

// Close shuts down the Azure VCS provider server.
func (az *Azure) Close() error {
	return az.echo.Close()
}

// ListenerAddr returns the Azure VCS provider server listener address.
func (az *Azure) ListenerAddr() net.Addr {
	return az.echo.ListenerAddr()
}

// CreateRepository creates a Azure repository with given ID.
func (az *Azure) CreateRepository(repository *vcs.Repository) error {
	parts := strings.Split(repository.ID, "/")
	if len(parts) != 3 {
		return errors.Errorf("invalid repo id: %s", repository.ID)
	}
	az.repositories[repository.FullPath] = &azureRepositoryData{
		repository: &azure.Repository{
			ID:   parts[2],
			Name: repository.Name,
			Project: &azure.Project{
				State: "wellFormed",
			},
		},
		files:   make(map[string]string),
		refs:    map[string]*azure.Branch{},
		commits: make(map[string]struct{ Changes []*azure.CommitChange }),
	}
	return nil
}

// CreateBranch creates a new branch with the given name.
func (az *Azure) CreateBranch(id, branchName string) error {
	pd, ok := az.repositories[id]
	if !ok {
		return errors.Errorf("repo %q doesn't exist", id)
	}

	if _, ok := pd.refs[fmt.Sprintf("refs/heads/%s", branchName)]; ok {
		return errors.Errorf("branch %q already exists", branchName)
	}

	pd.refs[fmt.Sprintf("refs/heads/%s", branchName)] = &azure.Branch{
		Name: branchName,
		Commit: &azure.BranchCommit{
			CommitID: "fake_azure_commit_sha",
		},
	}

	return nil
}

// SendWebhookPush sends out a webhook for a push event for the Azure
// repository using given payload.
func (az *Azure) SendWebhookPush(repositoryID string, payload []byte) error {
	r, ok := az.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Azure repository %q does not exist", repositoryID)
	}

	// Trigger all webhooks
	for _, webhook := range r.webhooks {
		if err := func() error {
			url := webhook.ConsumerInputs.URL
			req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
			if err != nil {
				return errors.Wrapf(err, "failed to create a new POST request to %q", url)
			}

			headers := strings.Split(webhook.ConsumerInputs.HTTPHeaders, ": ")
			req.Header.Set(headers[0], headers[1])

			resp, err := az.client.Do(req)
			if err != nil {
				return errors.Wrapf(err, "failed to send POST request to %q", url)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read response body")
			}
			if resp.StatusCode != http.StatusOK {
				return errors.Errorf("unexpected response status code %d, body: %s", resp.StatusCode, body)
			}
			az.echo.Logger.Infof("SendWebhookPush response body %s\n", body)
			return nil
		}(); err != nil {
			return err
		}
	}
	return nil
}

// AddFiles adds given files to the Azure repository.
func (az *Azure) AddFiles(repositoryID string, files map[string]string) error {
	r, ok := az.repositories[repositoryID]
	if !ok {
		return errors.Errorf("Azure repository %q does not exist", repositoryID)
	}

	// Save or overwrite files
	for path, content := range files {
		filePath := az.validFilePath(path)
		r.files[filePath] = content
	}
	return nil
}

// AddPullRequest creates a new pull request and add changed files to it.
func (az *Azure) AddPullRequest(repositoryID string, commitID int, files []*vcs.PullRequestFile) error {
	r, ok := az.repositories[repositoryID]
	if !ok {
		return errors.Errorf("repository %q does not exist", repositoryID)
	}

	changes := []*azure.CommitChange{}
	for _, file := range files {
		changeType := ""
		if file.IsDeleted {
			changeType = "delete"
		}
		changes = append(changes, &azure.CommitChange{
			ChangeType: changeType,
			Item: &azure.CommitChangeItem{
				CommitID:      fmt.Sprintf("%d", commitID),
				Path:          az.validFilePath(file.Path),
				GitObjectType: "blob",
			},
		})
	}

	r.commits[fmt.Sprintf("%d", commitID)] = struct {
		Changes []*azure.CommitChange
	}{
		Changes: changes,
	}

	return nil
}

func (*Azure) validFilePath(filePath string) string {
	if !strings.HasPrefix(filePath, "/") {
		return fmt.Sprintf("/%s", filePath)
	}
	return filePath
}
