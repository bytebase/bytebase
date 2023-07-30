// Package azure is the plugin for Azure DevOps.
package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
)

func init() {
	vcs.Register(vcs.AzureDevOps, newProvider)
}

const (
	apiPageSize = 100
)

var _ vcs.Provider = (*Provider)(nil)

// Provider is a Azure DevOps VCS provider.
type Provider struct {
	client *http.Client
}

func newProvider(config vcs.ProviderConfig) vcs.Provider {
	if config.Client == nil {
		config.Client = &http.Client{}
	}
	return &Provider{
		client: config.Client,
	}
}

// APIURL returns the API URL path of Azure DevOps.
func (*Provider) APIURL(instanceURL string) string {
	return instanceURL
}

// oauthResponse is a Azure DevOps OAuth response.
type oauthResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        string `json:"expires_in"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
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

// ServiceHookCodePushEvent represents a Azure DevOps service hook code push event.
//
// Docs: https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops#git.push
type ServiceHookCodePushEvent struct {
	EventType string                           `json:"eventType"`
	Message   ServiceHookCodePushEventMessage  `json:"message"`
	Resource  ServiceHookCodePushEventResource `json:"resource"`
}

// toVCSOAuthToken converts the response to *vcs.OAuthToken.
func (o oauthResponse) toVCSOAuthToken() (*vcs.OAuthToken, error) {
	expiresIn, err := strconv.ParseInt(o.ExpiresIn, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to parse expires_in "%s" with error: %v`, o.ExpiresIn, err.Error())
	}
	oauthToken := &vcs.OAuthToken{
		AccessToken:  o.AccessToken,
		RefreshToken: o.RefreshToken,
		ExpiresIn:    expiresIn,
		CreatedAt:    time.Now().Unix(),
		ExpiresTs:    time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
	}
	log.Debug("OAuth Token", zap.Any("oauthToken", oauthToken))
	return oauthToken, nil
}

// ExchangeOAuthToken exchanges OAuth content with the provided authorization code.
func (p *Provider) ExchangeOAuthToken(ctx context.Context, _ string, oauthExchange *common.OAuthExchange) (*vcs.OAuthToken, error) {
	params := &url.Values{}
	params.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	params.Set("client_assertion", oauthExchange.ClientSecret)
	params.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	params.Set("assertion", oauthExchange.Code)
	params.Set("redirect_uri", oauthExchange.RedirectURL)
	url := "https://app.vssps.visualstudio.com/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, errors.Wrapf(err, "construct POST %s", url)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to exchange OAuth token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read OAuth response body, code %v", resp.StatusCode)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	oauthResp := new(oauthResponse)
	if err := json.Unmarshal(body, oauthResp); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal OAuth response body, code %v", resp.StatusCode)
	}
	if oauthResp.Error != "" {
		return nil, errors.Errorf("failed to exchange OAuth token, error: %v, error_description: %v", oauthResp.Error, oauthResp.ErrorDescription)
	}
	return oauthResp.toVCSOAuthToken()
}

// ChangesResponseChangeItem represents a Azure DevOps changes response change item.
type ChangesResponseChangeItem struct {
	GitObjectType string `json:"gitObjectType"`
	Path          string `json:"path"`
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

// GetChangesByCommit gets the changes by commit ID.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/commits/get-changes?view=azure-devops-rest-7.0&tabs=HTTP
// TODO(zp): We should GET the changes pagenated, otherwise it may hit the Azure DevOps API limit.
func GetChangesByCommit(ctx context.Context, oauthCtx common.OauthContext, externalRepositoryID, commitID string) (*ChangesResponse, error) {
	client := &http.Client{}
	// By design, we encode the repository ID as <organization>/<projectID>/<repositoryID> for Azure DevOps.
	parts := strings.Split(externalRepositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", externalRepositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/commits/%s/changes?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), url.PathEscape(commitID), values.Encode())
	code, _, body, err := oauth.Get(ctx, client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("commit %q does not exist in the repository %s under the organization %s", commitID, repositoryID, organizationName))
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

// FetchCommitByID fetches the commit data by its ID from the repository.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/commits/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, _, externalRepositoryID, commitID string) (*vcs.Commit, error) {
	// By design, we encode the repository ID as <organization>/<projectID>/<repositoryID> for Azure DevOps.
	parts := strings.Split(externalRepositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", externalRepositoryID)
	}
	organizationName, projectID, repositoryID := parts[0], parts[1], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/commits/%s?%s", url.PathEscape(organizationName), url.PathEscape(projectID), url.PathEscape(repositoryID), url.PathEscape(commitID), values.Encode())
	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("commit %q does not exist in the repository %s under the organization %s", commitID, repositoryID, organizationName))
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type fetchCommitByIDResponse struct {
		CommitID  string       `json:"commitId"`
		Author    CommitAuthor `json:"author"`
		RemoteURL string       `json:"remoteUrl"`
	}

	commit := new(fetchCommitByIDResponse)
	if err := json.Unmarshal([]byte(body), commit); err != nil {
		return nil, errors.Wrapf(err, "unmarshal body")
	}

	return &vcs.Commit{
		ID:         commit.CommitID,
		AuthorName: commit.Author.Name,
		CreatedTs:  commit.Author.Date.Unix(),
		URL:        commit.RemoteURL,
	}, nil
}

// GetDiffFileList gets the diff files list between two commits.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/diffs/get?view=azure-devops-rest-7.0&tabs=HTTP#between-commit-ids
func (p *Provider) GetDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, beforeCommit string, afterCommit string) ([]vcs.FileDiff, error) {
	var result []vcs.FileDiff
	page := 0
	for {
		files, hasMore, err := p.getPaginatedDiffFileList(ctx, oauthCtx, instanceURL, repositoryID, beforeCommit, afterCommit, page)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get paginated diff file list")
		}
		result = append(result, files...)
		if !hasMore {
			break
		}
		page++
	}
	return result, nil
}

// getPaginatedDiffFileList gets the diff file list between two commits with pagination.
func (p *Provider) getPaginatedDiffFileList(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, beforeCommit string, afterCommit string, page int) ([]vcs.FileDiff, bool, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, false, errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("$top", fmt.Sprintf("%d", apiPageSize))
	values.Set("$skip", fmt.Sprintf("%d", page*apiPageSize))
	if beforeCommit != "" {
		values.Set("baseVersion", beforeCommit)
		values.Set("baseVersionType", "commit")
	}
	values.Set("targetVersion", afterCommit)
	values.Set("targetVersionType", "commit")
	values.Set("diffCommonCommit", "false")

	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/diffs/commits?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), values.Encode())
	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, false, errors.Wrapf(err, "GET %s", url)
	}

	if code != http.StatusOK {
		return nil, false, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type diffFileResponseChangeItem struct {
		Path string `json:"path"`
	}
	type diffFileResponseChange struct {
		Item       diffFileResponseChangeItem `json:"item"`
		ChangeType string                     `json:"changeType"`
	}
	type diffFileResponse struct {
		Changes []diffFileResponseChange `json:"changes"`
	}

	r := new(diffFileResponse)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, false, errors.Wrapf(err, "failed to unmarshal get diff file list response body, code %v", code)
	}

	result := make([]vcs.FileDiff, 0, len(r.Changes))
	for _, c := range r.Changes {
		var changeType vcs.FileDiffType
		switch c.ChangeType {
		case "add":
			changeType = vcs.FileDiffTypeAdded
		case "delete":
			changeType = vcs.FileDiffTypeRemoved
		case "edit":
			changeType = vcs.FileDiffTypeModified
		default:
			changeType = vcs.FileDiffTypeUnknown
		}
		result = append(result, vcs.FileDiff{
			Path: c.Item.Path,
			Type: changeType,
		})
	}

	return result, len(r.Changes) == apiPageSize, nil
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
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, _ string) ([]*vcs.Repository, error) {
	publicAlias, err := p.getAuthenticatedProfilePublicAlias(ctx, oauthCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get authenticated profile public alias")
	}
	log.Info("Authenticated user public alias", zap.String("publicAlias", publicAlias))
	organizations, err := p.listOrganizationsForMember(ctx, oauthCtx, publicAlias)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list organizations for member")
	}
	log.Info("Authenticated user organizations", zap.Strings("organizations", organizations))

	var result []*vcs.Repository

	type listRepositoriesResponseValueProject struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	}
	type listRepositoriesResponseValue struct {
		ID        string                               `json:"id"`
		Name      string                               `json:"name"`
		RemoteURL string                               `json:"remoteUrl"`
		Project   listRepositoriesResponseValueProject `json:"project"`
	}
	type listRepositoriesResponse struct {
		Count int                             `json:"count"`
		Value []listRepositoriesResponseValue `json:"value"`
	}

	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	for _, organization := range organizations {
		if err := func() error {
			url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories?%s", url.PathEscape(organization), urlParams.Encode())
			code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
				oauthContext{
					RefreshToken: oauthCtx.RefreshToken,
					ClientSecret: oauthCtx.ClientSecret,
					RedirectURL:  oauthCtx.RedirectURL,
				},
				oauthCtx.Refresher,
			))
			if err != nil {
				return errors.Wrapf(err, "GET %s", url)
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
					log.Debug("Skip the repository whose project is not wellFormed", zap.String("organization", organization), zap.String("project", r.Project.Name), zap.String("repository", r.Name))
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
	slices.SortFunc[*vcs.Repository](result, func(i, j *vcs.Repository) bool {
		return i.FullPath < j.FullPath
	})

	return result, nil
}

// getAuthenticatedProfilePublicAlias gets the authenticated user's profile, and returns the public alias in the
// profile response.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/profile/profiles/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) getAuthenticatedProfilePublicAlias(ctx context.Context, oauthCtx common.OauthContext) (string, error) {
	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/profile/profiles/me?%s", values.Encode())

	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
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
func (p *Provider) listOrganizationsForMember(ctx context.Context, oauthCtx common.OauthContext, memberID string) ([]string, error) {
	log.Info("Token: ", zap.String("token", oauthCtx.AccessToken))
	urlParams := &url.Values{}
	urlParams.Set("memberId", memberID)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/accounts?%s", urlParams.Encode())

	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	log.Info("List organizations for member response", zap.String("body", string(body)))

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

// FetchRepositoryFileList fetches the all files from the given repository tree recursively.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/trees/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, ref string, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("recursive", "true")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/trees/%s?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), url.PathEscape(ref), values.Encode())

	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type getTreesResponseTreeEntries struct {
		RelativePath  string `json:"relativePath"`
		GitObjectType string `json:"gitObjectType"`
	}
	type getTreesResponse struct {
		TreeEntries []getTreesResponseTreeEntries `json:"treeEntries"`
	}

	r := new(getTreesResponse)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get trees response body, code %v", code)
	}

	result := make([]*vcs.RepositoryTreeNode, 0, len(r.TreeEntries))
	for _, e := range r.TreeEntries {
		if e.GitObjectType != "blob" || !strings.HasPrefix(e.RelativePath, filePath) {
			continue
		}
		result = append(result, &vcs.RepositoryTreeNode{
			Path: e.RelativePath,
			Type: e.GitObjectType,
		})
	}
	return result, nil
}

// CreateFile creates a file at given path in the repository.
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.createOrUpdateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate, true)
}

// OverwriteFile overwrites an existing file at given path in the repository.
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.createOrUpdateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate, false)
}

func (p *Provider) getLatestCommitIDOnBranch(ctx context.Context, oauthCtx common.OauthContext, repositoryID, branchName string) (string, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]

	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("searchCriteria.itemVersion.version", branchName)

	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/commits?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), values.Encode())
	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET %s status code %d with body %q", url, code, string(body))
	}

	type getCommitsOnBranchResponseValue struct {
		CommitID string `json:"commitId"`
	}
	type getCommitsOnBranchResponse struct {
		Value []getCommitsOnBranchResponseValue `json:"value"`
	}

	g := new(getCommitsOnBranchResponse)
	if err := json.Unmarshal([]byte(body), g); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal get commits on branch response body, code %v", code)
	}

	return g.Value[0].CommitID, nil
}

func (p *Provider) createOrUpdateFile(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate, create bool) error {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, exteralRepositoryID := parts[0], parts[2]

	changeType := "edit"
	if create {
		changeType = "add"
	}

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/pushes?%s", url.PathEscape(organizationName), url.PathEscape(exteralRepositoryID), values.Encode())

	type createFileReqeustCommitChangesItem struct {
		Path string `json:"path"`
	}
	type createFileReqeustCommitChangesNewContent struct {
		Content     string `json:"content"`
		ContentType string `json:"contentType"`
	}
	type createFileReqeustCommitChanges struct {
		ChangeType string                                   `json:"changeType"`
		Item       createFileReqeustCommitChangesItem       `json:"item"`
		NewContent createFileReqeustCommitChangesNewContent `json:"newContent"`
	}
	type createFileReqeustCommitAuthor struct {
		Date  string `json:"date"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	type createFileReqeustCommit struct {
		Comment   string                           `json:"comment"`
		Changes   []createFileReqeustCommitChanges `json:"changes"`
		Author    createFileReqeustCommitAuthor    `json:"author"`
		Committer createFileReqeustCommitAuthor    `json:"committer"`
	}
	type createFileRequestRefUpdates struct {
		Name        string `json:"name"`
		OldObjectID string `json:"oldObjectId"`
	}
	type createFileRequest struct {
		RefUpdates []createFileRequestRefUpdates `json:"refUpdates"`
		Commits    []createFileReqeustCommit     `json:"commits"`
	}

	for i := 0; i < 3; i++ {
		branchCommit, err := p.getLatestCommitIDOnBranch(ctx, oauthCtx, repositoryID, fileCommitCreate.Branch)
		if err != nil {
			return errors.Wrapf(err, "failed to get latest commit ID on branch %q", fileCommitCreate.Branch)
		}

		requestBody := &createFileRequest{
			RefUpdates: []createFileRequestRefUpdates{
				{
					Name:        fmt.Sprintf("refs/heads/%s", fileCommitCreate.Branch),
					OldObjectID: branchCommit,
				},
			},
			Commits: []createFileReqeustCommit{
				{
					Comment: fileCommitCreate.CommitMessage,
					Changes: []createFileReqeustCommitChanges{
						{
							ChangeType: changeType,
							Item: createFileReqeustCommitChangesItem{
								Path: filePath,
							},
							NewContent: createFileReqeustCommitChangesNewContent{
								Content:     fileCommitCreate.Content,
								ContentType: "rawtext",
							},
						},
					},
				},
			},
		}
		if fileCommitCreate.AuthorName != "" && fileCommitCreate.AuthorEmail != "" {
			requestBody.Commits[0].Author = createFileReqeustCommitAuthor{
				Date:  time.Now().Format(time.RFC3339),
				Name:  fileCommitCreate.AuthorName,
				Email: fileCommitCreate.AuthorEmail,
			}
			requestBody.Commits[0].Committer = createFileReqeustCommitAuthor{
				Date:  time.Now().Format(time.RFC3339),
				Name:  fileCommitCreate.AuthorName,
				Email: fileCommitCreate.AuthorEmail,
			}
		}

		marshalBody, err := json.Marshal(requestBody)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal create file request body, request body: %+v", requestBody)
		}
		code, _, body, err := oauth.Post(ctx, p.client, url, &oauthCtx.AccessToken, bytes.NewReader(marshalBody), tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		))
		if err != nil {
			return errors.Wrapf(err, "POST %s", url)
		}
		if code == http.StatusBadRequest {
			log.Info("Failed to create file due to commit conflict, retrying", zap.String("url", url), zap.String("body", string(body)))
			continue
		}
		if code != http.StatusCreated {
			return errors.Errorf("non-201 POST %s status code %d with body %q", url, code, string(body))
		}

		return nil
	}

	return errors.Errorf("failed to create file after 3 retries")
}

// ReadFileMeta reads the metadata of the given file in the repository.
//
// Docs:
// - https://learn.microsoft.com/en-us/rest/api/azure/devops/git/items/get?view=azure-devops-rest-7.0&tabs=HTTP
// - https://learn.microsoft.com/en-us/rest/api/azure/devops/git/blobs/get-blob?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID, filePath, ref string) (*vcs.FileMeta, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("scopePath", filePath)
	values.Set("$format", "json")
	values.Set("versionDescriptor.version", ref)
	itemsURL := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/items?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), values.Encode())

	type fileMetaResponseValue struct {
		ObjectID string `json:"objectId"`
		CommitID string `json:"commitId"`
		Path     string `json:"path"`
		URL      string `json:"url"`
	}
	type fileMetaResponse struct {
		Value []fileMetaResponseValue `json:"value"`
	}

	code, _, body, err := oauth.Get(ctx, p.client, itemsURL, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", itemsURL)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to read file meta from URL %s", itemsURL)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", itemsURL, code, string(body))
	}

	r := new(fileMetaResponse)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get file meta response body, code %v", code)
	}

	// Validate Presumption: The response should only contain one file meta.
	if len(r.Value) != 1 {
		return nil, errors.Wrapf(err, fmt.Sprintf("expect to get one file meta, but got %d, response: %+v", len(r.Value), r))
	}

	values = &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("$format", "json")
	blobURL := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/blobs/%s?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), url.PathEscape(r.Value[0].ObjectID), values.Encode())

	type blobsResponse struct {
		Size int64 `json:"size"`
	}

	code, _, body, err = oauth.Get(ctx, p.client, blobURL, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", blobURL)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to read file size from URL %s", blobURL)
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 GET %s status code %d with body %q", blobURL, code, string(body))
	}

	b := new(blobsResponse)
	if err := json.Unmarshal([]byte(body), b); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get file meta response body, code %v", code)
	}

	return &vcs.FileMeta{
		Name:         filepath.Base(r.Value[0].Path),
		Path:         r.Value[0].Path,
		Size:         b.Size,
		LastCommitID: r.Value[0].CommitID,
		SHA:          r.Value[0].ObjectID,
	}, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/items/get?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, filePath string, ref string) (string, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]
	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("download", "false")
	values.Set("resolveLfs", "true")
	values.Set("includeContent", "true")
	values.Set("path", filePath)
	values.Set("versionDescriptor.version", ref)
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/items?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), values.Encode())

	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
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
func (p *Provider) GetBranch(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	if branchName == "" {
		return nil, errors.New("branch name is required")
	}

	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]

	urlParams := &url.Values{}
	urlParams.Set("name", branchName)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/stats/branches?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), urlParams.Encode())

	code, _, body, err := oauth.Get(ctx, p.client, url, &oauthCtx.AccessToken, tokenRefresher(
		oauthContext{
			RefreshToken: oauthCtx.RefreshToken,
			ClientSecret: oauthCtx.ClientSecret,
			RedirectURL:  oauthCtx.RedirectURL,
		},
		oauthCtx.Refresher,
	))
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, fmt.Sprintf("branch %q does not exist in the repository %s under the organization %s", branchName, repositoryID, organizationName))
	}
	if code != http.StatusOK {
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
		return nil, errors.Wrapf(err, "failed to unmarshal get the static of the branch %s of the repository %s under the organization %s response body, body: %s", branchName, repositoryID, organizationName, string(body))
	}
	return &vcs.BranchInfo{
		Name:         r.Name,
		LastCommitID: r.Commit.CommitID,
	}, nil
}

// CreateBranch creates the branch in the repository.
func (*Provider) CreateBranch(_ context.Context, _ common.OauthContext, _, _ string, _ *vcs.BranchInfo) error {
	return errors.New("not implemented")
}

// ListPullRequestFile lists the changed files in the pull request.
func (*Provider) ListPullRequestFile(_ context.Context, _ common.OauthContext, _, _, _ string) ([]*vcs.PullRequestFile, error) {
	return nil, errors.New("not implemented")
}

// CreatePullRequest creates the pull request in the repository.
func (*Provider) CreatePullRequest(_ context.Context, _ common.OauthContext, _, _ string, _ *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
	return nil, errors.New("not implemented")
}

// UpsertEnvironmentVariable creates or updates the environment variable in the repository.
func (*Provider) UpsertEnvironmentVariable(context.Context, common.OauthContext, string, string, string, string) error {
	return errors.New("not supported")
}

// CreateWebhook creates a webhook in the organization, and returns the webhook ID which can be used in PatchWebhook.
// API Version 7.0 do not specify the OAuth scope for creating webhook explicitly, but it works.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/hooks/subscriptions/create?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, _, externalRepositoryID string, payload []byte) (string, error) {
	parts := strings.Split(externalRepositoryID, "/")
	if len(parts) != 3 {
		return "", errors.Errorf("invalid repository ID %q", externalRepositoryID)
	}
	organizationName, _, _ := parts[0], parts[1], parts[2]
	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/hooks/subscriptions?%s", url.PathEscape(organizationName), urlParams.Encode())
	code, _, body, err := oauth.Post(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(payload),
		tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		),
	)
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

// PatchWebhook patches the webhook in the repository.
// Due to the Azure DevOps API do not provide an endpoint the update the webhook,
// so we should set the webhook full configuration in the payload.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/hooks/subscriptions/replace-subscription?view=azure-devops-rest-7.0&tabs=HTTP
// (2023/07/07, zp): It seems that the PatchWebhook API of each provider is not used by Bytebase, should we remove it?
func (*Provider) PatchWebhook(_ context.Context, _ common.OauthContext, _, _, _ string, _ []byte) error {
	return errors.New("not implemented")
}

// DeleteWebhook deletes the webhook in the repository.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/hooks/subscriptions/delete?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, _, _, webhookID string) error {
	// By design, we encode the webhook ID as <organization>/<webhookID> for Azure DevOps.
	parts := strings.Split(webhookID, "/")
	if len(parts) != 2 {
		return errors.Errorf("invalid webhook ID %q", webhookID)
	}
	organizationName, webhookID := parts[0], parts[1]

	values := &url.Values{}
	values.Set("api-version", "7.0")
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/hooks/subscriptions/%s?%s", url.PathEscape(organizationName), url.PathEscape(webhookID), values.Encode())

	code, _, body, err := oauth.Delete(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to send delete webhook request")
	}
	if code != http.StatusNoContent {
		return errors.Errorf("failed to delete webhook, code: %v, body: %s", code, string(body))
	}

	return nil
}

// CommitsInPushValue is the commit in the push.
type CommitsInPushValue struct {
	CommitID  string `json:"commitId"`
	RemoteURL string `json:"remoteUrl"`
}

// CommitInPush is the commit in the push.
type CommitsInPush struct {
	Value []CommitsInPushValue `json:"value"`
}

// GetPushCommitsByPushID gets the commits in the push by batch, it is useful when the push contains a lot of commits.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/commits/get-push-commits?view=azure-devops-rest-7.0&tabs=HTTP
func GetPushCommitsByPushID(ctx context.Context, oauthCtx common.OauthContext, repositoryID string, pushID uint64) (*CommitsInPush, error) {
	parts := strings.Split(repositoryID, "/")
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid repository ID %q", repositoryID)
	}
	organizationName, repositoryID := parts[0], parts[2]

	values := &url.Values{}
	values.Set("api-version", "7.0")
	values.Set("pushId", fmt.Sprintf("%d", pushID))
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories/%s/commits?%s", url.PathEscape(organizationName), url.PathEscape(repositoryID), values.Encode())

	client := &http.Client{}

	code, _, body, err := oauth.Get(
		ctx,
		client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			oauthContext{
				RefreshToken: oauthCtx.RefreshToken,
				ClientSecret: oauthCtx.ClientSecret,
				RedirectURL:  oauthCtx.RedirectURL,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get push commits")
	}
	if code != http.StatusOK {
		return nil, errors.Errorf("failed to get push commits, code: %v, body: %s", code, string(body))
	}

	r := new(CommitsInPush)
	if err := json.Unmarshal([]byte(body), r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal get push commits response body, code %v", code)
	}

	return r, nil
}

// oauthContext is the request context for OAuth.
type oauthContext struct {
	ClientSecret string
	RefreshToken string
	RedirectURL  string
}

type refreshOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// https://learn.microsoft.com/en-us/azure/devops/integrate/get-started/authentication/oauth?view=azure-devops#refresh-an-expired-access-token
func tokenRefresher(oauthCtx oauthContext, refresher common.TokenRefresher) oauth.TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		values := url.Values{}
		values.Set("client_assertion_type", `urn:ietf:params:oauth:client-assertion-type:jwt-bearer`)
		values.Set("client_assertion", oauthCtx.ClientSecret)
		values.Set("grant_type", "refresh_token")
		values.Set("assertion", oauthCtx.RefreshToken)
		values.Set("redirect_uri", oauthCtx.RedirectURL)
		encodedValues := values.Encode()

		url := "https://app.vssps.visualstudio.com/oauth2/token"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(encodedValues))
		if err != nil {
			return errors.Wrapf(err, "construct POST %s", url)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Content-Length", strconv.Itoa(len(encodedValues)))
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "failed to refresh OAuth token")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "read body of POST %s", url)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("non-200 POST %s status code %d with body %q", url, resp.StatusCode, body)
		}

		r := new(refreshOAuthTokenResponse)
		if err := json.Unmarshal(body, r); err != nil {
			return errors.Wrapf(err, "failed to unmarshal refresh OAuth token response body, code %v", resp.StatusCode)
		}

		*oldToken = r.AccessToken

		var expiresIn int64
		if r.ExpiresIn != "" {
			expiresAt, _ := strconv.ParseInt(r.ExpiresIn, 10, 64)
			expiresIn = time.Now().Unix() + expiresAt
		}
		return refresher(r.AccessToken, r.RefreshToken, expiresIn)
	}
}
