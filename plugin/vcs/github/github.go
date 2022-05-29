package github

import (
	"bytes"
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

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/internal/oauth"
)

const apiURL = "https://api.github.com"

func init() {
	vcs.Register(vcs.GitHubCom, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a GitHub.com VCS provider.
type Provider struct {
	l      *zap.Logger
	client *http.Client
}

func newProvider(config vcs.ProviderConfig) vcs.Provider {
	if config.Client == nil {
		config.Client = &http.Client{}
	}
	return &Provider{
		l:      config.Logger,
		client: config.Client,
	}
}

// APIURL returns the API URL path of GitHub.
func (p *Provider) APIURL(string) string {
	return apiURL
}

// User represents a GitHub API response for a user.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// fetchUserInfo fetches user information from the given resourceURI, which
// should be either "user" or "users/{username}".
func (p *Provider) fetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, resourceURI string) (*vcs.UserInfo, error) {
	url := fmt.Sprintf("%s/%s", apiURL, resourceURI)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "GET")
	}

	if code == http.StatusNotFound {
		errInfo := []string{"failed to fetch user info from GitHub.com"}
		resourceURISplit := strings.Split(resourceURI, "/")
		if len(resourceURI) > 1 {
			errInfo = append(errInfo, fmt.Sprintf("Username: %s", resourceURISplit[1]))
		}
		return nil, common.Errorf(common.NotFound, fmt.Errorf(strings.Join(errInfo, ", ")))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read user info from GitHub.com, status code: %d", code)
	}

	var user User
	if err = json.Unmarshal([]byte(body), &user); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &vcs.UserInfo{
		PublicEmail: user.Email,
		Name:        user.Name,
	}, err
}

// TryLogin tries to fetch the user info from the current OAuth context.
func (p *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, _ string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, "user")
}

// Commit represents a GitHub API response for a commit.
type Commit struct {
	SHA    string `json:"sha"`
	Author struct {
		Date time.Time `json:"date"`
		Name string    `json:"name"`
	} `json:"author"`
}

// FetchCommitByID fetches the commit data by its ID from the repository.
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("%s/repos/%s/git/commits/%s", apiURL, repositoryID, commitID)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "GET")
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, errors.New("failed to fetch commit data from GitHub.com, not found"))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to fetch commit data from GitHub.com, status code: %d, body: %s", code, body)
	}

	commit := &Commit{}
	if err := json.Unmarshal([]byte(body), commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit data from GitHub.com, err: %w", err)
	}

	return &vcs.Commit{
		ID:         commit.SHA,
		AuthorName: commit.Author.Name,
		CreatedTs:  commit.Author.Date.Unix(),
	}, nil
}

// FetchUserInfo fetches user info of given user ID.
func (p *Provider) FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, _, username string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, fmt.Sprintf("users/%s", username))
}

// FetchRepositoryActiveMemberList fetch all active members of a repository
func (p *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string) ([]*vcs.RepositoryMember, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// oauthResponse is a GitHub OAuth response.
type oauthResponse struct {
	AccessToken      string `json:"access_token" `
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// toVCSOAuthToken converts the response to *vcs.OAuthToken.
func (o oauthResponse) toVCSOAuthToken() *vcs.OAuthToken {
	oauthToken := &vcs.OAuthToken{
		AccessToken: o.AccessToken,
		// GitHub OAuth token never expires
	}
	return oauthToken
}

// ExchangeOAuthToken exchanges OAuth content with the provided authorization code.
func (p *Provider) ExchangeOAuthToken(ctx context.Context, _ string, oauthExchange *common.OAuthExchange) (*vcs.OAuthToken, error) {
	urlParams := &url.Values{}
	urlParams.Set("client_id", oauthExchange.ClientID)
	urlParams.Set("client_secret", oauthExchange.ClientSecret)
	urlParams.Set("code", oauthExchange.Code)
	urlParams.Set("redirect_uri", oauthExchange.RedirectURL)
	url := fmt.Sprintf("https://github.com/login/oauth/access_token?%s", urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		urlParams.Set("client_secret", "**redacted**")
		redactedURL := fmt.Sprintf("https://github.com/login/oauth/access_token?%s", urlParams.Encode())
		return nil, errors.Wrapf(err, "construct POST %s", redactedURL)
	}

	// GitHub returns URL-encoded parameters as the response format by default,
	// we need to ask for a JSON response explicitly.
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth token, error: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth response body, code %v, error: %v", resp.StatusCode, err)
	}
	defer func() { _ = resp.Body.Close() }()

	oauthResp := new(oauthResponse)
	if err := json.Unmarshal(body, oauthResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OAuth response body, code %v, error: %v", resp.StatusCode, err)
	}
	if oauthResp.Error != "" {
		return nil, fmt.Errorf("failed to exchange OAuth token, error: %v, error_description: %v", oauthResp.Error, oauthResp.ErrorDescription)
	}
	return oauthResp.toVCSOAuthToken(), nil
}

// FetchAllRepositoryList fetches all repositories where the authenticated user has a maintainer role.
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	return nil, errors.New("not implemented yet")
}

// FetchRepositoryFileList fetch the files from repository tree
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// CreateFile creates a file.
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit vcs.FileCommitCreate) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// OverwriteFile overwrite the content of a file.
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit vcs.FileCommitCreate) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// ReadFileMeta reads the file metadata.
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*vcs.FileMeta, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// ReadFileContent reads the file content.
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (string, error) {
	return "", errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// CreateWebhook creates a webhook in a GitLab project.
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	return "", errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// PatchWebhook patches a webhook in a GitLab project.
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// DeleteWebhook deletes a webhook in a GitLab project.
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

// oauthContext is the request context for refreshing oauth token.
type oauthContext struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"`
}

type refreshOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
	// token_type, scope are not used.
}

func tokenRefresher(oauthCtx oauthContext, refresher common.TokenRefresher) oauth.TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		url := fmt.Sprintf("%s/login/oauth/access_token", apiURL)
		oauthCtx.GrantType = "refresh_token"
		body, err := json.Marshal(oauthCtx)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return errors.Wrapf(err, "construct POST %s", url)
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "POST %s", url)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "read body of POST %s", url)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("non-200 status code %d with body %q", resp.StatusCode, body)
		}

		var r refreshOAuthResponse
		if err = json.Unmarshal(body, &r); err != nil {
			return errors.Wrapf(err, "unmarshal body from POST %s", url)
		}

		// Update the old token to new value for retries.
		*oldToken = r.AccessToken

		// OAuth token never expires for traditional GitHub OAuth (i.e. not a GitHub App)
		var expireAt int64
		if r.ExpiresIn != "" {
			expiresIn, _ := strconv.ParseInt(r.ExpiresIn, 10, 64)
			expireAt = r.CreatedAt + expiresIn
		}
		if err = refresher(r.AccessToken, r.RefreshToken, expireAt); err != nil {
			return err
		}
		return nil
	}
}
