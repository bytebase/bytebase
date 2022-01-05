package gitlab

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

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"go.uber.org/zap"
)

const (
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	maxRetries = 3

	// apiPath is the API path.
	apiPath = "api/v4"
)

var (
	_ vcs.Provider = (*Provider)(nil)
)

// WebhookType is the gitlab webhook type.
type WebhookType string

const (
	// WebhookPush is the webhook type for push.
	WebhookPush WebhookType = "push"
)

func (e WebhookType) String() string {
	switch e {
	case WebhookPush:
		return "push"
	}
	return "UNKNOWN"
}

// webhookInfo is the API message for webhook info.
type webhookInfo struct {
	ID int `json:"id"`
}

// WebhookPost is the API message for webhook POST.
type WebhookPost struct {
	URL         string `json:"url"`
	SecretToken string `json:"token"`
	// This is set to true
	PushEvents bool `json:"push_events"`
	// For now, there is no native dry run DDL support in mysql/postgres. One may wonder if we could wrap the DDL
	// in a transaction and just not commit at the end, unfortunately there are side effects which are hard to control.
	// See https://www.postgresql.org/message-id/CAMsr%2BYGiYQ7PYvYR2Voio37YdCpp79j5S%2BcmgVJMOLM2LnRQcA%40mail.gmail.com
	// So we can't possibly display useful info when reviewing a MR, thus we don't enable this event.
	// Saying that, delivering a souding dry run solution would be great and hopefully we can achieve that one day.
	// MergeRequestsEvents  bool   `json:"merge_requests_events"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
	// TODO(tianzhou): This is set to false, be lax to not enable_ssl_verification
	EnableSSLVerification bool `json:"enable_ssl_verification"`
}

// WebhookPut is the API message for webhook PUT.
type WebhookPut struct {
	URL                    string `json:"url"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
}

// WebhookProject is the API message for webhook project.
type WebhookProject struct {
	ID       int    `json:"id"`
	WebURL   string `json:"web_url"`
	FullPath string `json:"path_with_namespace"`
}

// WebhookCommitAuthor is the API message for webhook commit author.
type WebhookCommitAuthor struct {
	Name string `json:"name"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	Message   string              `json:"message"`
	Timestamp string              `json:"timestamp"`
	URL       string              `json:"url"`
	Author    WebhookCommitAuthor `json:"author"`
	AddedList []string            `json:"added"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	ObjectKind WebhookType     `json:"object_kind"`
	Ref        string          `json:"ref"`
	AuthorName string          `json:"user_name"`
	Project    WebhookProject  `json:"project"`
	CommitList []WebhookCommit `json:"commits"`
}

// fileCommit is the API message for file commit.
type fileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
}

// fileMeta is the API message for file metadata.
type fileMeta struct {
	LastCommitID string `json:"last_commit_id"`
}

func init() {
	vcs.Register(vcs.GitLabSelfHost, newProvider)
}

// Provider is the GitLab self host provider.
type Provider struct {
	l *zap.Logger
}

func newProvider(config vcs.ProviderConfig) vcs.Provider {
	return &Provider{
		l: config.Logger,
	}
}

func (provider *Provider) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/%s", instanceURL, apiPath)
}

func (provider *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(fileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	resp, err := httpPost(
		instanceURL,
		fmt.Sprintf("projects/%s/repository/files/%s", repositoryID, url.QueryEscape(filePath)),
		&oauthCtx.AccessToken,
		bytes.NewBuffer(body),
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, err: %w", filePath, instanceURL, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			resp.StatusCode,
		)
	}
	return nil
}

func (provider *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(fileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
		LastCommitID:  fileCommitCreate.LastCommitID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	resp, err := httpPut(
		instanceURL,
		fmt.Sprintf("projects/%s/repository/files/%s", repositoryID, url.QueryEscape(filePath)),
		&oauthCtx.AccessToken,
		bytes.NewBuffer(body),
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, error: %w", filePath, instanceURL, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			resp.StatusCode,
		)
	}
	return nil
}

func (provider *Provider) ReadFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, commitID string) (io.ReadCloser, error) {
	resp, err := httpGet(
		instanceURL,
		fmt.Sprintf("projects/%s/repository/files/%s/raw?ref=%s", repositoryID, url.QueryEscape(filePath), commitID),
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to read file %s from GitLab instance %s: %w", filePath, instanceURL, err)
	}

	if resp.StatusCode == 404 {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to read file %s from GitLab instance %s, file not found", filePath, instanceURL))
	} else if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to read file %s from GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			resp.StatusCode,
		)
	}

	return resp.Body, nil
}

func (provider *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, branch string) (*vcs.FileMeta, error) {
	resp, err := httpGet(
		instanceURL,
		fmt.Sprintf("projects/%s/repository/files/%s?ref=%s", repositoryID, url.QueryEscape(filePath), url.QueryEscape(branch)),
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to read file meta %s from GitLab instance %s: %w", filePath, instanceURL, err)
	}

	if resp.StatusCode == 404 {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to read file meta %s from GitLab instance %s, file not found", filePath, instanceURL))
	} else if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to read file meta %s from GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			resp.StatusCode,
		)
	}
	defer resp.Body.Close()

	file := &fileMeta{}
	if err := json.NewDecoder(resp.Body).Decode(file); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file meta from GitLab instance %s: %w", instanceURL, err)
	}

	return &vcs.FileMeta{
		LastCommitID: file.LastCommitID,
	}, nil
}

func (provider *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, payload []byte) (string, error) {
	resourcePath := fmt.Sprintf("projects/%s/hooks", repositoryID)
	resp, err := httpPost(
		instanceURL,
		resourcePath,
		&oauthCtx.AccessToken,
		bytes.NewBuffer(payload),
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook for repository %s from GitLab instance %s: %w", repositoryID, instanceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		reason := fmt.Sprintf(
			"failed to create webhook for repository %s from GitLab instance %s, status code: %d",
			repositoryID,
			instanceURL,
			resp.StatusCode,
		)
		// Add helper tips if the status code is 422, refer to bytebase#101 for more context.
		if resp.StatusCode == http.StatusUnprocessableEntity {
			reason += ".\n\nIf GitLab and Bytebase are in the same private network, " +
				"please follow the instructions in https://docs.gitlab.com/ee/security/webhooks.html"
		}
		return "", fmt.Errorf(reason)
	}

	webhookInfo := &webhookInfo{}
	if err := json.NewDecoder(resp.Body).Decode(webhookInfo); err != nil {
		return "", fmt.Errorf("failed to unmarshal create webhook response for repository %s from GitLab instance %s: %w", repositoryID, instanceURL, err)
	}
	return strconv.Itoa(webhookInfo.ID), nil
}

func (provider *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, webhookId string, payload []byte) error {
	resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repositoryID, webhookId)
	resp, err := httpPut(
		instanceURL,
		resourcePath,
		&oauthCtx.AccessToken,
		bytes.NewBuffer(payload),
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitLab instance %s: %w", webhookId, repositoryID, instanceURL, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitLab instance %s, status code: %d", webhookId, repositoryID, instanceURL, resp.StatusCode)
	}
	return nil
}

func (provider *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, webhookId string) error {
	resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repositoryID, webhookId)
	resp, err := httpDelete(
		instanceURL,
		resourcePath,
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitLab instance %s: %w", webhookId, repositoryID, instanceURL, err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitLab instance %s, status code: %d", webhookId, repositoryID, instanceURL, resp.StatusCode)
	}
	return nil
}

// httpPost sends a POST request.
func httpPost(instanceURL string, resourcePath string, token *string, body io.Reader, oauthContext oauthContext, refresher common.TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, apiPath, resourcePath)
		req, err := http.NewRequest("POST",
			url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to construct POST %v (%w)", url, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed POST %v (%w)", url, err)
		}
		return resp, nil
	})
}

// httpGet sends a GET request.
func httpGet(instanceURL string, resourcePath string, token *string, oauthContext oauthContext, refresher common.TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, apiPath, resourcePath)
		req, err := http.NewRequest("GET",
			url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to construct GET %v (%w)", url, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed GET %v (%w)", url, err)
		}
		return resp, nil
	})
}

// httpPut sends a PUT request.
func httpPut(instanceURL string, resourcePath string, token *string, body io.Reader, oauthContext oauthContext, refresher common.TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, apiPath, resourcePath)
		req, err := http.NewRequest("PUT",
			url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to construct PUT %v (%w)", url, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed PUT %v (%w)", url, err)
		}
		return resp, nil
	})
}

// httpDelete sends a DELETE request.
func httpDelete(instanceURL string, resourcePath string, token *string, oauthContext oauthContext, refresher common.TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, apiPath, resourcePath)
		req, err := http.NewRequest("DELETE",
			url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to construct DELETE %v (%w)", url, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed DELETE %v (%w)", url, err)
		}
		return resp, nil
	})
}

func retry(instanceURL string, token *string, oauthContext oauthContext, refresher common.TokenRefresher, f func() (*http.Response, error)) (*http.Response, error) {
	retries := 0
RETRY:
	retries++

	resp, err := f()
	if err != nil {
		return nil, err
	}

	if err := getOAuthErrorDetails(resp); err != nil && retries < maxRetries {
		// Refresh and store the token.
		if err := refreshToken(instanceURL, token, oauthContext, refresher); err != nil {
			return nil, err
		}
		goto RETRY
	}
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type oauthError struct {
	Err              string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e *oauthError) Error() string {
	return fmt.Sprintf("gitlab oauth response error %q description %q", e.Err, e.ErrorDescription)
}

// Only returns error if it's an oauth error. For other errors like 404 we don't return error.
// We do this because this method is only intended to be used by oauth to refresh access token
// on expiration. When it's error like 404, GitLab api doesn't return it as error so we keep the
// similar behavior and let caller check the response status code.
func getOAuthErrorDetails(resp *http.Response) error {
	if 200 <= resp.StatusCode && resp.StatusCode < 300 {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var oe oauthError
	if err = json.Unmarshal([]byte(body), &oe); err != nil {
		return err
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

func refreshToken(instanceURL string, oldToken *string, oauthContext oauthContext, refresher common.TokenRefresher) error {
	url := fmt.Sprintf("%s/oauth/token", instanceURL)
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
	if err := getOAuthErrorDetails(resp); err != nil {
		return err
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from refresh token POST %v (%w)", url, err)
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
