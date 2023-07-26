// Package azure is the plugin for Azure DevOps.
package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

func init() {
	vcs.Register(vcs.AzureDevOps, newProvider)
}

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

// project is Azure DevOps project.
type repository struct {
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

// FetchCommitByID fetches the commit data by its ID from the repository.
func (*Provider) FetchCommitByID(_ context.Context, _ common.OauthContext, _, _, _ string) (*vcs.Commit, error) {
	return nil, errors.New("not implemented")
}

// GetDiffFileList gets the diff files list between two commits.
func (*Provider) GetDiffFileList(_ context.Context, _ common.OauthContext, _, _, _, _ string) ([]vcs.FileDiff, error) {
	return nil, errors.New("not implemented")
}

// FetchAllRepositoryList fetches all projects where the authenticated use has permissions, which is required
// to create webhook in the repository.
//
// NOTE: Azure DevOps does not support listing all projects cross all organizations API yet, thus we need
// to follow the https://stackoverflow.com/questions/53608013/get-all-organizations-via-rest-api-for-azure-devops
// to get all projects.
// The request included in this function requires the following scopes:
// vso.profile, vso.project.
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
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

	type project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Url  string `json:"url"`
	}
	type listProjectResponse struct {
		Count int       `json:"count"`
		Value []project `json:"value"`
	}

	urlParams := &url.Values{}
	urlParams.Set("api-version", "7.0")
	urlParams.Set("stateFilter", "wellFormed")
	for _, organization := range organizations {
		url := fmt.Sprintf("https://dev.azure.com/%s/_apis/projects?%s", url.PathEscape(organization), urlParams.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "construct GET %s", url)
		}
		req.Header.Set("Authorization", "Bearer "+oauthCtx.AccessToken)

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list repositories")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read list organizations for member response body, code %v", resp.StatusCode)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		l := new(listProjectResponse)
		if err := json.Unmarshal(body, l); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal list organizations for member response body, code %v", resp.StatusCode)
		}
		for _, p := range l.Value {
			result = append(result, &vcs.Repository{
				ID:       p.ID,
				Name:     p.Name,
				FullPath: fmt.Sprintf("%s/%s", organization, p.Name),
				WebURL:   p.Url,
			})
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
	url := "https://app.vssps.visualstudio.com/_apis/profile/profiles/me?api-version=7.0"
	type profileAlias struct {
		PublicAlias string `json:"publicAlias"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "construct GET %s", url)
	}
	req.Header.Set("Authorization", "Bearer "+oauthCtx.AccessToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get authenticated profile")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read profile response body, code %v", resp.StatusCode)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	r := new(profileAlias)
	if err := json.Unmarshal(body, r); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal profile response body, code %v", resp.StatusCode)
	}

	return r.PublicAlias, nil
}

// listOrganizationsForMember lists all organization for a given member.
//
// Docs: https://learn.microsoft.com/en-us/rest/api/azure/devops/account/accounts/list?view=azure-devops-rest-7.0&tabs=HTTP
func (p *Provider) listOrganizationsForMember(ctx context.Context, oauthCtx common.OauthContext, memberId string) ([]string, error) {
	log.Info("Token: ", zap.String("token", oauthCtx.AccessToken))
	urlParams := &url.Values{}
	urlParams.Set("memberId", memberId)
	urlParams.Set("api-version", "7.0")
	url := fmt.Sprintf("https://app.vssps.visualstudio.com/_apis/accounts?%s", urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "construct GET %s", url)
	}
	req.Header.Set("Authorization", "Bearer "+oauthCtx.AccessToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list organizations for member")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read list organizations for member response body, code %v", resp.StatusCode)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	log.Info("List organizations for member response", zap.String("body", string(body)))

	type accountsValue struct {
		AccountName string `json:"accountName"`
	}
	type accountsResponse struct {
		Count int             `json:"count"`
		Value []accountsValue `json:"value"`
	}

	r := new(accountsResponse)
	if err := json.Unmarshal(body, r); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal list organizations for member response body, code %v", resp.StatusCode)
	}

	result := make([]string, 0, len(r.Value))
	for _, v := range r.Value {
		result = append(result, v.AccountName)
	}

	return result, nil
}

// FetchRepositoryFileList fetches the all files from the given repository tree recursively.
func (*Provider) FetchRepositoryFileList(_ context.Context, _ common.OauthContext, _, _, _, _ string) ([]*vcs.RepositoryTreeNode, error) {
	return nil, errors.New("not implemented")
}

// CreateFile creates a file at given path in the repository.
func (*Provider) CreateFile(_ context.Context, _ common.OauthContext, _, _, _ string, _ vcs.FileCommitCreate) error {
	return errors.New("not implemented")
}

// OverwriteFile overwrites an existing file at given path in the repository.
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.CreateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate)
}

// ReadFileMeta reads the metadata of the given file in the repository.
func (*Provider) ReadFileMeta(_ context.Context, _ common.OauthContext, _, _, _, _ string) (*vcs.FileMeta, error) {
	return nil, errors.New("not implemented")
}

// ReadFileContent reads the content of the given file in the repository.
func (*Provider) ReadFileContent(_ context.Context, _ common.OauthContext, _, _, _, _ string) (string, error) {
	return "", errors.New("not implemented")
}

// GetBranch gets the given branch in the repository.
func (*Provider) GetBranch(_ context.Context, _ common.OauthContext, _, _, _ string) (*vcs.BranchInfo, error) {
	return nil, errors.New("not implemented")
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

// CreateWebhook creates a webhook in the repository with given payload.
func (*Provider) CreateWebhook(_ context.Context, _ common.OauthContext, _, _ string, _ []byte) (string, error) {
	return "", errors.New("not implemented")
}

// PatchWebhook patches the webhook in the repository with given payload.
func (*Provider) PatchWebhook(_ context.Context, _ common.OauthContext, _, _, _ string, _ []byte) error {
	return errors.New("not implemented")
}

// DeleteWebhook deletes the webhook from the repository.
func (*Provider) DeleteWebhook(_ context.Context, _ common.OauthContext, _, _, _ string) error {
	return errors.New("not implemented")
}
