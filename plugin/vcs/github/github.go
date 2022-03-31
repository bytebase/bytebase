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
