package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	return retry(ctx, client, token, oauthCtx, refresher,
		func() (*http.Response, error) {
			url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "construct GET %s", url)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
			resp, err := client.Do(req)
			if err != nil {
				return nil, errors.Wrapf(err, "GET %s", url)
			}
			return resp, nil
		},
	)
}

func retry(ctx context.Context, client *http.Client, token *string, oauthCtx oauthContext, refresher common.TokenRefresher, f func() (*http.Response, error)) (code int, respBody string, err error) {
	var resp *http.Response
	var body []byte
	for retries := 0; retries < maxRetries; retries++ {
		select {
		case <-ctx.Done():
			return 0, "", ctx.Err()
		default:
		}

		resp, err = f()
		if err != nil {
			return 0, "", err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, "", errors.Wrapf(err, "read response body with status code %d", resp.StatusCode)
		}

		if err = getOAuthErrorDetails(resp.StatusCode, body); err != nil {
			if _, ok := err.(*oauthError); ok {
				// Refresh and store the token.
				if err := refreshToken(ctx, client, token, oauthCtx, refresher); err != nil {
					return 0, "", err
				}
				continue
			}
			return 0, "", errors.Errorf("want *oauthError but got %T", err)
		}
		return resp.StatusCode, string(body), nil
	}
	return 0, "", errors.Errorf("retries exceeded for OAuth refresher with status code %d and body %q", resp.StatusCode, string(body))
}

type oauthError struct {
	Err              string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e oauthError) Error() string {
	return fmt.Sprintf("GitHub OAuth response error %q description %q", e.Err, e.ErrorDescription)
}

// getOAuthErrorDetails only returns error if it's an OAuth error. For other
// errors like 404 we don't return error. We do this because this method is only
// intended to be used by oauth to refresh access token on expiration.
func getOAuthErrorDetails(code int, body []byte) error {
	if 200 <= code && code < 300 {
		return nil
	}

	var oe oauthError
	if err := json.Unmarshal(body, &oe); err != nil {
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

type refreshOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
	// token_type, scope are not used.
}

func refreshToken(ctx context.Context, client *http.Client, oldToken *string, oauthContext oauthContext, refresher common.TokenRefresher) error {
	url := fmt.Sprintf("%s/login/oauth/access_token", apiURL)
	oauthContext.GrantType = "refresh_token"
	body, err := json.Marshal(oauthContext)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
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
