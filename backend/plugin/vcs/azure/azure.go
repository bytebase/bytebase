// Package azure is the plugin for Azure DevOps.
package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	vcs.Register(storepb.VCSType_AZURE_DEVOPS, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a Azure DevOps VCS provider.
type Provider struct {
	instanceURL string
	authToken   string
	RootAPIURL  string
}

func newProvider(config vcs.ProviderConfig) vcs.Provider {
	return &Provider{
		instanceURL: config.InstanceURL,
		authToken:   config.AuthToken,
	}
}

// APIURL returns the API URL path of Azure DevOps.
func (*Provider) APIURL(instanceURL string) string {
	return instanceURL
}

func (p *Provider) rootAPIURL() string {
	if strings.HasPrefix(p.instanceURL, "http://localhost:") {
		// This is used for mock vcs server in test.
		// TODO: find better ways without changing the code.
		return p.instanceURL
	}
	return "https://app.vssps.visualstudio.com"
}

type Project struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// Repository is the API message for Azure repository.
type Repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// RemoteURL is the repo url in https://{org name}@dev.azure.com/{org name}/{project name}/_git/{repo name}
	// The pipeline ci will use this url, so we need this url
	RemoteURL string `json:"remoteUrl"`
	// WebURL is the repo url in https://dev.azure.com/{org name}/{project name}/_git/{repo name}
	WebURL  string   `json:"webUrl"`
	Project *Project `json:"project"`
}

// CommitChangeItem represents a Azure DevOps changes response change item.
type CommitChangeItem struct {
	GitObjectType string `json:"gitObjectType"`
	Path          string `json:"path"`
	CommitID      string `json:"commitId"`
}

// CommitChange represents a Azure DevOps changes response change.
type CommitChange struct {
	Item       *CommitChangeItem `json:"item"`
	ChangeType string            `json:"changeType"`
}

type changesResponse struct {
	Changes []*CommitChange `json:"changes"`
}

// getChangesByCommit gets the changes by commit ID, and returns the list of blob files changed in the specify commit.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/commits/get-changes?view=azure-devops-rest-7.0&tabs=HTTP
// TODO(zp): We should GET the changes pagenated, otherwise it may hit the Azure DevOps API limit.
func (p *Provider) getChangesByCommit(ctx context.Context, externalRepositoryID, commitID string) (*changesResponse, error) {
	apiURL, err := p.getRepositoryAPIURL(externalRepositoryID)
	if err != nil {
		return nil, err
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/commits/%s/changes?%s", apiURL, url.PathEscape(commitID), values.Encode())
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "commit %q does not exist in the repository %s", commitID, externalRepositoryID)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	changes := new(changesResponse)
	if err := json.Unmarshal([]byte(body), changes); err != nil {
		return nil, errors.Wrapf(err, "unmarshal body")
	}

	var result changesResponse
	for _, change := range changes.Changes {
		if change.Item.GitObjectType == "blob" {
			result.Changes = append(result.Changes, change)
		}
	}

	return &result, nil
}

// FetchRepositoryList fetches all projects where the authenticated use has permissions, which is required
// to create webhook in the repository.
//
// NOTE: Azure DevOps does not support listing all projects cross all organizations API yet, thus we need
// to follow the https://stackoverflow.com/questions/53608013/get-all-organizations-via-rest-api-for-azure-devops
// to get all projects.
// The request included in this function requires the following scopes:
// vso.profile, vso.project.
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/repositories/list?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) FetchRepositoryList(ctx context.Context, listAll bool) ([]*vcs.Repository, error) {
	publicAlias, err := p.getAuthenticatedProfilePublicAlias(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get authenticated profile public alias")
	}

	organizations, err := p.listOrganizationsForMember(ctx, publicAlias)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list organizations for member")
	}

	var result []*vcs.Repository

	type listRepositoriesResponse struct {
		Count int          `json:"count"`
		Value []Repository `json:"value"`
	}

	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	for _, organization := range organizations {
		if err := func() error {
			url := fmt.Sprintf("%s/%s/_apis/git/repositories?%s", p.APIURL(p.instanceURL), url.PathEscape(organization), urlParams.Encode())
			code, body, err := internal.Get(ctx, url, p.getAuthorization())
			if err != nil {
				return errors.Wrapf(err, "GET %s", url)
			}
			// If users do not have permission to list repositories, for example, do not open the switch of
			// `Third party application access via OAuth` in the organization settings, Azure DevOps will return
			// 401 Unauthorized.
			if code == http.StatusUnauthorized {
				return nil
			}

			if code != http.StatusOK {
				return errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
			}

			l := new(listRepositoriesResponse)
			if err := json.Unmarshal([]byte(body), l); err != nil {
				return errors.Wrapf(err, "failed to unmarshal list organizations for member response body %v, code %v", body, code)
			}

			for _, r := range l.Value {
				if r.Project.State != "wellFormed" {
					slog.Debug("Skip the repository whose project is not wellFormed", slog.String("organization", organization), slog.String("project", r.Project.Name), slog.String("repository", r.Name))
				}

				result = append(result, &vcs.Repository{
					ID:       fmt.Sprintf("%s/%s/%s", organization, r.Project.ID, r.ID),
					Name:     r.Name,
					FullPath: fmt.Sprintf("%s/%s/%s", organization, r.Project.Name, r.Name),
					WebURL:   r.WebURL,
				})
			}
			return nil
		}(); err != nil {
			return nil, errors.Wrapf(err, "failed to list repositories under the organization %s", organization)
		}
		if !listAll {
			break
		}
	}

	// Sort result by FullPath.
	sort.Slice(result, func(i, j int) bool {
		return result[i].FullPath < result[j].FullPath
	})

	return result, nil
}

type Profile struct {
	PublicAlias string `json:"publicAlias"`
}

// getAuthenticatedProfilePublicAlias gets the authenticated user's profile, and returns the public alias in the
// profile response.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/profile/profiles/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) getAuthenticatedProfilePublicAlias(ctx context.Context) (string, error) {
	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/_apis/profile/profiles/me?%s", p.rootAPIURL(), values.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	r := new(Profile)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal profile response body, code %v", code)
	}

	return r.PublicAlias, nil
}

type Organization struct {
	AccountName string `json:"accountName"`
}

// listOrganizationsForMember lists all organization for a given member.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/account/accounts/list?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) listOrganizationsForMember(ctx context.Context, memberID string) ([]string, error) {
	urlParams := &url.Values{}
	urlParams.Set("memberId", memberID)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/_apis/accounts?%s", p.rootAPIURL(), urlParams.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type accountsResponse struct {
		Count int             `json:"count"`
		Value []*Organization `json:"value"`
	}

	r := new(accountsResponse)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal list organizations for member response body, code %v", code)
	}

	result := make([]string, 0, len(r.Value))
	for _, v := range r.Value {
		result = append(result, v.AccountName)
	}

	return result, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/items/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) ReadFileContent(ctx context.Context, repositoryID, filePath string, refInfo vcs.RefInfo) (string, error) {
	apiURL, err := p.getRepositoryAPIURL(repositoryID)
	if err != nil {
		return "", err
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("download", "false")
	values.Set("resolveLfs", "true")
	values.Set("includeContent", "true")
	values.Set("path", filePath)
	var refType string
	switch refInfo.RefType {
	case vcs.RefTypeBranch:
		refType = "branch"
	case vcs.RefTypeTag:
		refType = "tag"
	case vcs.RefTypeCommit:
		refType = "commit"
	default:
		return "", errors.Errorf("invalid ref type %q", refInfo.RefType)
	}
	values.Set("versionDescriptor.versionType", refType)
	values.Set("versionDescriptor.version", refInfo.RefName)
	url := fmt.Sprintf("%s/items?%s", apiURL, values.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}
	return string(body), nil
}

type BranchCommit struct {
	CommitID string `json:"commitId"`
}

type Branch struct {
	Name   string        `json:"name"`
	Commit *BranchCommit `json:"commit"`
}

// GetBranch try to retrieve the branch from the repository, and returns the last commit ID of the branch, if the branch
// does not exist, it returns common.NotFound.
// Args:
// - repositoryID: The repository ID in the format of <organization>/<repository>.
// - branchName: The branch name.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/stats/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) GetBranch(ctx context.Context, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	apiURL, err := p.getRepositoryAPIURL(repositoryID)
	if err != nil {
		return nil, err
	}

	urlParams := &url.Values{}
	urlParams.Set("name", branchName)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/stats/branches?%s", apiURL, urlParams.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code >= 300 {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	r := new(Branch)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get the static of the branch %s of the repository %s with response body, body: %s", branchName, repositoryID, string(body))
	}
	return &vcs.BranchInfo{
		Name:         r.Name,
		LastCommitID: r.Commit.CommitID,
	}, nil
}

// ListPullRequestFileInCommit lists the changed files by last merge commit id.
func (p *Provider) ListPullRequestFileInCommit(ctx context.Context, repositoryID, lastMergeCommitID string, pullRequestID int) ([]*vcs.PullRequestFile, error) {
	organizationName, projectName, repoID, err := getAzureRepositoryIDs(repositoryID)
	if err != nil {
		return nil, err
	}

	var prURL string
	if pullRequestID != 0 {
		prURL = fmt.Sprintf("%s/%s/%s/_git/%s/pullrequest/%d", p.instanceURL, organizationName, projectName, repoID, pullRequestID)
	}
	changeResponse, err := p.getChangesByCommit(ctx, repositoryID, lastMergeCommitID)
	if err != nil {
		return nil, err
	}
	files := []*vcs.PullRequestFile{}
	for _, change := range changeResponse.Changes {
		var webURL string
		if prURL != "" {
			// Web URL for file in PR:
			// {PR web URL}?_a=files&path={file path}
			webURL = fmt.Sprintf("%s?_a=files&path=%s", webURL, url.QueryEscape(change.Item.Path))
		}
		files = append(files, &vcs.PullRequestFile{
			Path:         change.Item.Path,
			LastCommitID: change.Item.CommitID,
			IsDeleted:    change.ChangeType == "delete",
			WebURL:       webURL,
		})
	}
	return files, nil
}

// ListPullRequestFile lists the changed files by last merge commit id.
func (p *Provider) ListPullRequestFile(ctx context.Context, repositoryID, lastMergeCommitID string) ([]*vcs.PullRequestFile, error) {
	return p.ListPullRequestFileInCommit(ctx, repositoryID, lastMergeCommitID, 0)
}

type Comment struct {
	Content     string `json:"content"`
	CommentType string `json:"commentType"`
}

type PullRequestThread struct {
	ID       int64      `json:"id"`
	Comments []*Comment `json:"comments"`
	// https://learn.microsoft.com/en-us/rest/api/azure/devops/git/pull-request-threads/list?view=azure-devops-rest-7.1&tabs=HTTP#commentthreadstatus
	Status string `json:"status"`
}

// CreatePullRequestComment creates a pull request comment.
// We will create a thread in Azure pull request instead of a comment.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/pull-request-threads/create?view=azure-devops-rest-7.1&tabs=HTTP
func (p *Provider) CreatePullRequestComment(ctx context.Context, repositoryID, pullRequestID, comment string) error {
	thread := &PullRequestThread{
		Status: "active",
		Comments: []*Comment{
			{
				Content:     comment,
				CommentType: "text",
			},
		},
	}
	commentCreatePayload, err := json.Marshal(thread)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body for creating pull request comment")
	}

	apiURL, err := p.getRepositoryAPIURL(repositoryID)
	if err != nil {
		return err
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/pullRequests/%s/threads?%s", apiURL, pullRequestID, values.Encode())
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), commentCreatePayload)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return errors.Errorf("failed to create thread, code: %v, body: %s", code, string(body))
	}

	return nil
}

// ListPullRequestComments lists comments in a pull request.
//
// https://learn.microsoft.com/en-us/rest/api/azure/devops/git/pull-request-threads/list?view=azure-devops-rest-7.1&tabs=HTTP
func (p *Provider) ListPullRequestComments(ctx context.Context, repositoryID, pullRequestID string) ([]*vcs.PullRequestComment, error) {
	apiURL, err := p.getRepositoryAPIURL(repositoryID)
	if err != nil {
		return nil, err
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/pullRequests/%s/threads?%s", apiURL, pullRequestID, values.Encode())
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("failed to list thread, code: %v, body: %s", code, string(body))
	}

	var resp struct {
		Value []*PullRequestThread `json:"value"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, err
	}

	var res []*vcs.PullRequestComment
	for _, thread := range resp.Value {
		if len(thread.Comments) != 1 {
			continue
		}
		res = append(res, &vcs.PullRequestComment{
			ID:      fmt.Sprintf("%d", thread.ID),
			Content: thread.Comments[0].Content,
		})
	}
	return res, nil
}

// UpdatePullRequestComment updates a comment in a pull request.
//
// https://learn.microsoft.com/en-us/rest/api/azure/devops/git/pull-request-threads/update?view=azure-devops-rest-7.1
func (p *Provider) UpdatePullRequestComment(ctx context.Context, repositoryID, pullRequestID string, comment *vcs.PullRequestComment) error {
	thread := &PullRequestThread{
		Status: "active",
		Comments: []*Comment{
			{
				Content:     comment.Content,
				CommentType: "text",
			},
		},
	}
	commentCreatePayload, err := json.Marshal(thread)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body for creating pull request comment")
	}

	apiURL, err := p.getRepositoryAPIURL(repositoryID)
	if err != nil {
		return err
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/pullRequests/%s/threads/%s?%s", apiURL, pullRequestID, comment.ID, values.Encode())
	code, body, err := internal.Patch(ctx, url, p.getAuthorization(), commentCreatePayload)
	if err != nil {
		return errors.Wrapf(err, "PATCH %s", url)
	}
	if code != http.StatusOK {
		return errors.Errorf("failed to update thread, code: %v, body: %s", code, string(body))
	}

	return nil
}

// CreateWebhook creates a webhook in the organization, and returns the webhook ID which can be used in PatchWebhook.
// API Version 7.0 do not specify the OAuth scope for creating webhook explicitly, but it works.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/hooks/subscriptions/create?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) CreateWebhook(ctx context.Context, externalRepositoryID string, payload []byte) (string, error) {
	parts := strings.Split(externalRepositoryID, "/")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid repository ID %q", externalRepositoryID)
	}
	organizationName, _, _ := parts[0], parts[1], parts[2]
	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/%s/_apis/hooks/subscriptions?%s", p.APIURL(p.instanceURL), url.PathEscape(organizationName), urlParams.Encode())
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), payload)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create webhook")
	}
	if code >= 300 {
		return "", errors.Errorf("failed to create webhook, code: %v, body: %s", code, string(body))
	}

	type createServiceResponse struct {
		ID string `json:"id"`
	}
	c := new(createServiceResponse)
	if err := json.Unmarshal([]byte(body), c); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal create webhook response body, code %v", code)
	}
	return fmt.Sprintf("%s/%s", organizationName, c.ID), nil
}

// DeleteWebhook deletes the webhook in the repository.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/hooks/subscriptions/delete?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) DeleteWebhook(ctx context.Context, _, webhookID string) error {
	// By design, we encode the webhook ID as <organization>/<webhookID> for Azure DevOps.
	parts := strings.Split(webhookID, "/")
	if len(parts) != 2 {
		return errors.Errorf("invalid webhook ID %q", webhookID)
	}
	organizationName, webhookID := parts[0], parts[1]

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/%s/_apis/hooks/subscriptions/%s?%s", p.APIURL(p.instanceURL), url.PathEscape(organizationName), url.PathEscape(webhookID), values.Encode())

	code, body, err := internal.Delete(ctx, url, p.getAuthorization())
	if err != nil {
		return errors.Wrapf(err, "failed to send delete webhook request")
	}
	if code != http.StatusNoContent {
		return errors.Errorf("failed to delete webhook, code: %v, body: %s", code, string(body))
	}

	return nil
}

func getAzureRepositoryIDs(repositoryID string) (string, string, string, error) {
	// By design, we encode the repository ID as <organization>/<projectID>/<repositoryID> for Azure DevOps.
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return "", "", "", errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, projectName, repositoryID := parts[0], parts[1], parts[2]
	return organizationName, projectName, repositoryID, nil
}

func (p *Provider) getRepositoryAPIURL(repositoryID string) (string, error) {
	organizationName, projectName, repositoryID, err := getAzureRepositoryIDs(repositoryID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s", p.APIURL(p.instanceURL), url.PathEscape(organizationName), url.PathEscape(projectName), url.PathEscape(repositoryID)), nil
}

// getAuthorization returns the encoded azure token for authorization.
// Docs: https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops&tabs=Windows
func (p *Provider) getAuthorization() string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(":%s", p.authToken)))
	return fmt.Sprintf("Basic %s", encoded)
}
