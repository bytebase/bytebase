// Package azure is the plugin for Azure DevOps.
package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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

// oauthResponse is a Bitbucket Cloud OAuth response.
type oauthResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        string `json:"expires_in"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
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

// TryLogin tries to fetch the user info from the current OAuth context.
func (*Provider) TryLogin(context.Context, common.OauthContext, string) (*vcs.UserInfo, error) {
	return nil, errors.New("not implemented")
}

// FetchCommitByID fetches the commit data by its ID from the repository.
func (*Provider) FetchCommitByID(_ context.Context, _ common.OauthContext, _, _, _ string) (*vcs.Commit, error) {
	return nil, errors.New("not implemented")
}

// GetDiffFileList gets the diff files list between two commits.
func (*Provider) GetDiffFileList(_ context.Context, _ common.OauthContext, _, _, _, _ string) ([]vcs.FileDiff, error) {
	return nil, errors.New("not implemented")
}

// FetchAllRepositoryList fetches all repositories.
func (*Provider) FetchAllRepositoryList(_ context.Context, _ common.OauthContext, _ string) ([]*vcs.Repository, error) {
	return nil, errors.New("not implemented")
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
