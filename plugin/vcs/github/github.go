package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

const (
	maxRetries = 3

	apiURL = "https://api.github.com"
)

var _ vcs.Provider = (*Provider)(nil)

func init() {
	vcs.Register(vcs.GitHubCom, newProvider)
}

// Provider is the GitLab self host provider.
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

func (p *Provider) APIURL(string) string {
	return apiURL
}

// fetchUserInfo fetches user information from the given resourceURI, which
// should be either "user" or "users/{username}".
func (p *Provider) fetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, resourceURI string) (*vcs.UserInfo, error) {
	code, body, err := httpGet(
		ctx,
		p.client,
		resourceURI,
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return nil, errors.Wrap(err, "GET")
	}

	if code == http.StatusNotFound {
		errInfo := []string{"failed to fetch user info from GitHub.com"}
		resourceURISplit := strings.Split(resourceURI, "/")
		if len(resourceURI) > 1 {
			errInfo = append(errInfo, fmt.Sprintf("UserID: %v", resourceURISplit[1]))
		}
		return nil, common.Errorf(common.NotFound, fmt.Errorf(strings.Join(errInfo, ", ")))
	} else if code >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to read user info from GitHub.com, status code: %d",
			code,
		)
	}

	userInfo := &vcs.UserInfo{}
	if err = json.Unmarshal([]byte(body), userInfo); err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return userInfo, err
}

func (p *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, _ string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, "user")
}

func (p *Provider) FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, _, username string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, fmt.Sprintf("users/%s", username))
}

func (p *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string) ([]*vcs.RepositoryMember, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit vcs.FileCommitCreate) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit vcs.FileCommitCreate) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) ReadFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, commitID string) (string, error) {
	return "", errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, branch string) (*vcs.FileMeta, error) {
	return nil, errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	return "", errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	return errors.New("not implemented yet") // TODO: https://github.com/bytebase/bytebase/issues/928
}

func httpGet(ctx context.Context, client *http.Client, resourcePath string, token *string, oauthCtx oauthContext, refresher common.TokenRefresher) (code int, respBody string, err error) {
	return retry(ctx, token, oauthCtx, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "construct GET %q", url)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "GET %q", url)
		}
		return resp, nil
	})
}

func retry(ctx context.Context, token *string, oauthCtx oauthContext, refresher common.TokenRefresher, f func() (*http.Response, error)) (code int, respBody string, err error) {
	retries := 0
retry:
	retries++

	resp, err := f()
	if err != nil {
		return 0, "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("failed to read gitlab response body, code %v, error: %v", resp.StatusCode, err)
	}

	if err := getOAuthErrorDetails(resp.StatusCode, string(body)); err != nil {
		if _, ok := err.(oauthError); ok && retries < maxRetries {
			// Refresh and store the token.
			if err := refreshToken(token, oauthCtx, refresher); err != nil {
				return 0, "", err
			}
			goto retry // todo ???

		}
		// err must be oauthError. So this happens only when the number of retries has exceeded.
		return 0, "", fmt.Errorf("retries exceeded for oauth refresher; original code %v body %s; oauth error: %v", resp.StatusCode, string(body), err)
	}

	return resp.StatusCode, string(body), nil
}

type oauthError struct {
	Err              string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e oauthError) Error() string {
	return fmt.Sprintf("GitHub oauth response error %q description %q", e.Err, e.ErrorDescription)
}

// Only returns error if it's an oauth error. For other errors like 404 we don't return error.
// We do this because this method is only intended to be used by oauth to refresh access token
// on expiration. When it's error like 404, GitLab api doesn't return it as error so we keep the
// similar behavior and let caller check the response status code.
func getOAuthErrorDetails(code int, body string) error {
	if 200 <= code && code < 300 {
		return nil
	}

	var oe oauthError
	if err := json.Unmarshal([]byte(body), &oe); err != nil {
		// If we failed to unmarshal body with oauth error, it's not oauthError and we should return nil.
		return nil
	}
	// https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/
	// {"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
	if oe.Err == "invalid_token" && strings.Contains(oe.ErrorDescription, "expired") {
		return &oe
	}
	return nil
}

// oauthContext is the request context for refreshing oauth token.
type oauthContext struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"`
}

type refreshOauthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
	// token_type, scope are not used.
}

func refreshToken(oldToken *string, oauthContext oauthContext, refresher common.TokenRefresher) error {
	url := fmt.Sprintf("%s/oauth/token", apiURL)
	oauthContext.GrantType = "refresh_token"
	body, err := json.Marshal(oauthContext)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to construct refresh token POST %v (%w)", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send refresh token POST %v (%w)", url, err)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from refresh token POST %v (%w)", url, err)
	}

	// We should not call getOAuthErrorDetails.
	// In the sequence of 1) get file content with oauth error, 2) refresh token.
	// If step 2) failed still with oauth error, we should stop retries because we should always expect refreshing token request to succeed unless we're holding any invalid refresh token already.

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch refresh token, response code %v body %s", resp.StatusCode, body)
	}

	var r refreshOauthResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		return fmt.Errorf("failed to unmarshal body from refresh token POST %v (%w)", url, err)
	}

	// Update the old token to new value for retries.
	*oldToken = r.AccessToken

	// For GitLab, as of 13.12, the default config won't expire the access token, thus this field is 0.
	// see https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
	var expireAt int64
	if r.ExpiresIn != 0 {
		expireAt = r.CreatedAt + r.ExpiresIn
	}
	if err := refresher(r.AccessToken, r.RefreshToken, expireAt); err != nil {
		return err
	}

	return nil
}
