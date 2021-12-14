package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// APIPath is the API path.
	APIPath = "api/v4"
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	maxRetries = 3
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

// WebhookInfo is the API message for webhook info.
type WebhookInfo struct {
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

// FileCommit is the API message for file commit.
type FileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
}

// File is the API message for file.
type File struct {
	LastCommitID string `json:"last_commit_id"`
}

// POST sends a POST request.
func POST(instanceURL string, resourcePath string, token *string, body io.Reader, oauthContext OauthContext, refresher TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
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

// GET sends a GET request.
func GET(instanceURL string, resourcePath string, token *string, oauthContext OauthContext, refresher TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
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

// PUT sends a PUT request.
func PUT(instanceURL string, resourcePath string, token *string, body io.Reader, oauthContext OauthContext, refresher TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
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

// DELETE sends a DELETE request.
func DELETE(instanceURL string, resourcePath string, token *string, oauthContext OauthContext, refresher TokenRefresher) (*http.Response, error) {
	return retry(instanceURL, token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
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

func retry(instanceURL string, token *string, oauthContext OauthContext, refresher TokenRefresher, f func() (*http.Response, error)) (*http.Response, error) {
	retries := 0
RETRY:
	retries++

	resp, err := f()
	if err != nil {
		return nil, err
	}
	err = getErrorDetails(resp)
	if expiredTokenError(err) && retries < maxRetries {
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

func getErrorDetails(resp *http.Response) error {
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
	if oe.Err != "" || oe.ErrorDescription != "" {
		return &oe
	}
	return fmt.Errorf("gitlab response code %v error %q", resp.StatusCode, body)
}

func expiredTokenError(e error) bool {
	if e == nil {
		return false
	}
	oe, ok := e.(*oauthError)
	if !ok {
		return false
	}
	// https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/
	// {"error":"invalid_token","error_description":"Token is expired. You can either do re-authorization or token refresh."}
	return oe.Err == "invalid_token" && strings.Contains(oe.ErrorDescription, "expired")
}

// TokenRefresher is a function refreshes the oauth token and updates the repository.
type TokenRefresher func(token, refreshToken string, expiresTs int64) error

// OauthContext is the request context for refreshing oauth token.
type OauthContext struct {
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

func refreshToken(instanceURL string, oldToken *string, oauthContext OauthContext, refresher TokenRefresher) error {
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
	if err := getErrorDetails(resp); err != nil {
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
