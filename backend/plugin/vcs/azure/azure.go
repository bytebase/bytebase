// Package azure is the plugin for Azure DevOps.
package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal"
)

func init() {
	vcs.Register(vcs.AzureDevOps, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a Azure DevOps VCS provider.
type Provider struct {
	instanceURL string
	authToken   string
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

// WebhookCreateConsumerInputs represents the consumer inputs for creating a webhook.
type WebhookCreateConsumerInputs struct {
	URL                  string `json:"url"`
	AcceptUntrustedCerts bool   `json:"acceptUntrustedCerts"`
}

// WebhookCreatePublisherInputs represents the publisher inputs for creating a webhook.
type WebhookCreatePublisherInputs struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	PushedBy   string `json:"pushedBy"`
	ProjectID  string `json:"projectId"`
}

// WebhookCreateOrUpdate represents a Bitbucket API request for creating or
// updating a webhook.
type WebhookCreateOrUpdate struct {
	ConsumerActionID string                       `json:"consumerActionId"`
	ConsumerID       string                       `json:"consumerId"`
	ConsumerInputs   WebhookCreateConsumerInputs  `json:"consumerInputs"`
	EventType        string                       `json:"eventType"`
	PublisherID      string                       `json:"publisherId"`
	PublisherInputs  WebhookCreatePublisherInputs `json:"publisherInputs"`
}

// CommitAuthor represents a Azure DevOps commit author.
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// ServiceHookCodePushEventMessage represents a Azure DevOps service hook code push event message.
type ServiceHookCodePushEventMessage struct {
	Text string `json:"text"`
}

// ServiceHookCodePushEventResourceCommit represents a Azure DevOps service hook code push event resource commit.
type ServiceHookCodePushEventResourceCommit struct {
	CommitID string       `json:"commitId"`
	Author   CommitAuthor `json:"author"`
	Comment  string       `json:"comment"`
	URL      string       `json:"url"`
}

// ServiceHookCodePushEventRefUpdates represents a Azure DevOps service hook code push event ref updates.
type ServiceHookCodePushEventRefUpdates struct {
	Name        string `json:"name"`
	OldObjectID string `json:"oldObjectId"`
	NewObjectID string `json:"newObjectId"`
}

// ServiceHookCodePushEventResourcePushedBy represents a Azure DevOps service hook code push event resource pushed by.
type ServiceHookCodePushEventResourcePushedBy struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

// ServiceHookCodePushEventResourceRepository represents a Azure DevOps service hook code push event resource repository.
type ServiceHookCodePushEventResourceRepository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ServiceHookCodePushEventResource represents a Azure DevOps service hook code push event resource.
type ServiceHookCodePushEventResource struct {
	Commits    []ServiceHookCodePushEventResourceCommit   `json:"commits"`
	Repository ServiceHookCodePushEventResourceRepository `json:"repository"`
	RefUpdates []ServiceHookCodePushEventRefUpdates       `json:"refUpdates"`
	PushedBy   ServiceHookCodePushEventResourcePushedBy   `json:"pushedBy"`
	PushID     uint64                                     `json:"pushId"`
}

type project struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type repository struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// RemoteURL is the repo url in https://{org name}@dev.azure.com/{org name}/{project name}/_git/{repo name}
	// The pipeline ci will use this url, so we need this url
	RemoteURL string `json:"remoteUrl"`
	// WebURL is the repo url in https://dev.azure.com/{org name}/{project name}/_git/{repo name}
	WebURL  string  `json:"webUrl"`
	Project project `json:"project"`
}

// ServiceHookCodePushEvent represents a Azure DevOps service hook code push event.
//
// Docs: https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops#git.push
type ServiceHookCodePushEvent struct {
	EventType string                           `json:"eventType"`
	Message   ServiceHookCodePushEventMessage  `json:"message"`
	Resource  ServiceHookCodePushEventResource `json:"resource"`
}

// ChangesResponseChangeItem represents a Azure DevOps changes response change item.
type ChangesResponseChangeItem struct {
	GitObjectType string `json:"gitObjectType"`
	Path          string `json:"path"`
	CommitID      string `json:"commitId"`
}

// ChangesResponseChange represents a Azure DevOps changes response change.
type ChangesResponseChange struct {
	Item       ChangesResponseChangeItem `json:"item"`
	ChangeType string                    `json:"changeType"`
}

// ChangesResponse represents a Azure DevOps changes response.
type ChangesResponse struct {
	Changes []ChangesResponseChange `json:"changes"`
}

// getChangesByCommit gets the changes by commit ID, and returns the list of blob files changed in the specify commit.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/commits/get-changes?view=azure-devops-rest-7.0&tabs=HTTP
// TODO(zp): We should GET the changes pagenated, otherwise it may hit the Azure DevOps API limit.
func (p *Provider) getChangesByCommit(ctx context.Context, externalRepositoryID, commitID string) (*ChangesResponse, error) {
	apiURL, err := getRepositoryAPIURL(externalRepositoryID)
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
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("commit %q does not exist in the repository %s", commitID, externalRepositoryID))
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	changes := new(ChangesResponse)
	if err := json.Unmarshal([]byte(body), changes); err != nil {
		return nil, errors.Wrapf(err, "unmarshal body")
	}

	var result ChangesResponse
	for _, change := range changes.Changes {
		if change.Item.GitObjectType == "blob" {
			result.Changes = append(result.Changes, change)
		}
	}

	return &result, nil
}

// FetchAllRepositoryList fetches all projects where the authenticated use has permissions, which is required
// to create webhook in the repository.
//
// NOTE: Azure DevOps does not support listing all projects cross all organizations API yet, thus we need
// to follow the https://stackoverflow.com/questions/53608013/get-all-organizations-via-rest-api-for-azure-devops
// to get all projects.
// The request included in this function requires the following scopes:
// vso.profile, vso.project.
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/repositories/list?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) FetchAllRepositoryList(ctx context.Context) ([]*vcs.Repository, error) {
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
		Value []repository `json:"value"`
	}

	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	for _, organization := range organizations {
		if err := func() error {
			url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories?%s", url.PathEscape(organization), urlParams.Encode())
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
					WebURL:   r.RemoteURL,
				})
			}
			return nil
		}(); err != nil {
			return nil, errors.Wrapf(err, "failed to list repositories under the organization %s", organization)
		}
	}

	// Sort result by FullPath.
	sort.Slice(result, func(i, j int) bool {
		return result[i].FullPath < result[j].FullPath
	})

	return result, nil
}

// getAuthenticatedProfilePublicAlias gets the authenticated user's profile, and returns the public alias in the
// profile response.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/profile/profiles/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) getAuthenticatedProfilePublicAlias(ctx context.Context) (string, error) {
	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/profile/profiles/me?%s", values.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type profileAlias struct {
		PublicAlias string `json:"publicAlias"`
	}

	r := new(profileAlias)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal profile response body, code %v", code)
	}

	return r.PublicAlias, nil
}

// listOrganizationsForMember lists all organization for a given member.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/account/accounts/list?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) listOrganizationsForMember(ctx context.Context, memberID string) ([]string, error) {
	urlParams := &url.Values{}
	urlParams.Set("memberId", memberID)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/accounts?%s", urlParams.Encode())

	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type accountsValue struct {
		AccountName string `json:"accountName"`
	}
	type accountsResponse struct {
		Count int             `json:"count"`
		Value []accountsValue `json:"value"`
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
	apiURL, err := getRepositoryAPIURL(repositoryID)
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

// GetBranch try to retrieve the branch from the repository, and returns the last commit ID of the branch, if the branch
// does not exist, it returns common.NotFound.
// Args:
// - repositoryID: The repository ID in the format of <organization>/<repository>.
// - branchName: The branch name.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/stats/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) GetBranch(ctx context.Context, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	if branchName == "" {
		return nil, errors.New("branch name is required")
	}

	apiURL, err := getRepositoryAPIURL(repositoryID)
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

	type branchStatResponseCommit struct {
		CommitID string `json:"commitId"`
	}
	type branchStatResponse struct {
		Name   string                   `json:"name"`
		Commit branchStatResponseCommit `json:"commit"`
	}

	r := new(branchStatResponse)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get the static of the branch %s of the repository %s with response body, body: %s", branchName, repositoryID, string(body))
	}
	return &vcs.BranchInfo{
		Name:         r.Name,
		LastCommitID: r.Commit.CommitID,
	}, nil
}

// ListPullRequestFile lists the changed files in the pull request.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/pull-requests/get-pull-request?view=azure-devops-rest-7.1
func (p *Provider) ListPullRequestFile(ctx context.Context, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	type mergeCommit struct {
		CommitID string `json:"commitId"`
	}
	type azurePullRequest struct {
		LastMergeCommit *mergeCommit `json:"lastMergeCommit"`
	}

	apiURL, err := getRepositoryAPIURL(repositoryID)
	if err != nil {
		return nil, err
	}

	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("%s/pullrequests/%s?%s", apiURL, pullRequestID, urlParams.Encode())

	code, resp, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code >= 300 {
		return nil, errors.Errorf("failed to create merge request from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}

	var res azurePullRequest
	if err := json.Unmarshal([]byte(resp), &res); err != nil {
		return nil, err
	}

	changeResponse, err := p.getChangesByCommit(ctx, repositoryID, res.LastMergeCommit.CommitID)
	if err != nil {
		return nil, err
	}
	files := []*vcs.PullRequestFile{}
	for _, change := range changeResponse.Changes {
		files = append(files, &vcs.PullRequestFile{
			Path:         change.Item.Path,
			LastCommitID: change.Item.CommitID,
			IsDeleted:    change.ChangeType == "delete",
		})
	}
	return files, nil
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
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/hooks/subscriptions?%s", url.PathEscape(organizationName), urlParams.Encode())
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), payload)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create webhook")
	}
	if code != http.StatusOK {
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
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/hooks/subscriptions/%s?%s", url.PathEscape(organizationName), url.PathEscape(webhookID), values.Encode())

	code, body, err := internal.Delete(ctx, url, p.getAuthorization())
	if err != nil {
		return errors.Wrapf(err, "failed to send delete webhook request")
	}
	if code != http.StatusNoContent {
		return errors.Errorf("failed to delete webhook, code: %v, body: %s", code, string(body))
	}

	return nil
}

func getRepositoryAPIURL(repositoryID string) (string, error) {
	// By design, we encode the repository ID as <organization>/<projectID>/<repositoryID> for Azure DevOps.
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, projectName, repositoryID := parts[0], parts[1], parts[2]

	return fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s", url.PathEscape(organizationName), url.PathEscape(projectName), url.PathEscape(repositoryID)), nil
}

func (p *Provider) getAuthorization() string {
	return fmt.Sprintf("Basic %s", p.authToken)
}
