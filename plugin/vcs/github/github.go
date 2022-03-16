package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

const (
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	maxRetries = 3

	apiURL = "https://api.github.com"
)

var (
	_ vcs.Provider = (*Provider)(nil)
)

// WebhookType is the GitHub webhook type.
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

type WebhookPostConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
	InsecureSSL bool   `json:"insecure_ssl"`
}

// WebhookPost is the API message for webhook POST.
type WebhookPost struct {
	Config WebhookPostConfig `json:"config"`
	Events []string          `json:"events"`
	Active bool              `json:"active"`
}

// WebhookPut is the API message for webhook PUT.
type WebhookPut struct {
	URL                    string `json:"url"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
}

// WebhookRepository is the API message for webhook project.
type WebhookRepository struct {
	ID       int    `json:"id"`
	HTMLURL  string `json:"html_url"`
	FUllName string `json:"full_name"`
}

// WebhookCommitAuthor is the API message for webhook commit author.
type WebhookCommitAuthor struct {
	Name string `json:"name"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	ID        string              `json:"id"`
	Message   string              `json:"message"`
	Timestamp string              `json:"timestamp"`
	URL       string              `json:"url"`
	Author    WebhookCommitAuthor `json:"author"`
	AddedList []string            `json:"added"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	ObjectKind WebhookType       `json:"object_kind"`
	Ref        string            `json:"ref"`
	AuthorName string            `json:"user_name"`
	Repository WebhookRepository `json:"repository"`
	CommitList []WebhookCommit   `json:"commits"`
}

// FileCommit is the API message for file commit.
type FileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
}

// FileMeta is the API message for file metadata.
type FileMeta struct {
	LastCommitID string `json:"last_commit_id"`
}

// ProjectRole is the role of the project member
type ProjectRole string

// Gitlab Role type
const (
	ProjectRoleOwner         ProjectRole = "Owner"
	ProjectRoleMaintainer    ProjectRole = "Maintainer"
	ProjectRoleDeveloper     ProjectRole = "Developer"
	ProjectRoleReporter      ProjectRole = "Reporter"
	ProjectRoleGuest         ProjectRole = "Guest"
	ProjectRoleMinimalAccess ProjectRole = "MinimalAccess"
	ProjectRoleNoAccess      ProjectRole = "NoAccess"
)

func (e ProjectRole) String() string {
	switch e {
	case ProjectRoleOwner,
		ProjectRoleMaintainer,
		ProjectRoleDeveloper,
		ProjectRoleReporter,
		ProjectRoleGuest,
		ProjectRoleMinimalAccess,
		ProjectRoleNoAccess:
		return string(e)
	}
	return ""
}

// gitLabRepositoryMember is the API message for repository member
type gitLabRepositoryMember struct {
	ID          int       `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	State       vcs.State `json:"state"`
	AccessLevel int32     `json:"access_level"`
}

func init() {
	vcs.Register(vcs.GitHubDotCom, newProvider)
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

// APIURL returns the API URL path of a GitLab instance.
func (provider *Provider) APIURL(_ string) string {
	return apiURL
}

// fetchUserInfo will fetch user info from the given resourceURI, resourceURI should be either 'user' or 'users/:userID'
func fetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, resourceURI string) (*vcs.UserInfo, error) {
	code, body, err := httpGet(
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
		return nil, err
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
	if err := json.Unmarshal([]byte(body), userInfo); err != nil {
		return nil, err
	}

	return userInfo, err
}

// TryLogin will try to fetch the user info from the current OAuth content of GitHub.
func (provider *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, _ string) (*vcs.UserInfo, error) {
	return fetchUserInfo(ctx, oauthCtx, "user")
}

// FetchUserInfo will fetch user info from GitHub. If userID is set to nil, the user info of the current oauth would be returned.
func (provider *Provider) FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, _ string, userID int) (*vcs.UserInfo, error) {
	return fetchUserInfo(ctx, oauthCtx, fmt.Sprintf("users/%d", userID))
}

func getRoleAndMappedRole(accessLevel int32) (gitLabRole ProjectRole, bytebaseRole common.ProjectRole) {
	// see https://docs.gitlab.com/ee/api/members.html for the detailed role type at GitLab
	switch accessLevel {
	case 50 /* Owner */ :
		return ProjectRoleOwner, common.ProjectOwner
	case 40 /* Maintainer */ :
		return ProjectRoleMaintainer, common.ProjectOwner
	case 30 /* Developer */ :
		return ProjectRoleDeveloper, common.ProjectDeveloper
	case 20 /* Reporter */ :
		return ProjectRoleReporter, common.ProjectDeveloper
	case 10 /* Guest */ :
		return ProjectRoleGuest, common.ProjectDeveloper
	case 5 /* Minimal access */ :
		return ProjectRoleMinimalAccess, common.ProjectDeveloper
	case 0 /* No access */ :
		return ProjectRoleNoAccess, common.ProjectDeveloper
	}

	return "", ""
}

// FetchRepositoryActiveMemberList fetch all active members of a repository
func (provider *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string) ([]*vcs.RepositoryMember, error) {
	code, body, err := httpGet(
		// official API doc: https://docs.gitlab.com/14.6/ee/api/members.html
		fmt.Sprintf("projects/%s/members/all", repositoryID),
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)
	if err != nil {
		return nil, err
	}

	if code == 404 {
		return nil, common.Errorf(common.NotFound, errors.New("failed to fetch repository members from GitHub.com"))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read repository members from GitHub.com, status code: %d",
			code,
		)
	}

	var gitLabrepositoryMember []gitLabRepositoryMember
	if err := json.Unmarshal([]byte(body), &gitLabrepositoryMember); err != nil {
		return nil, err
	}

	// we only return active member (both state and membership_state is active)
	var emptyEmailUserIDList []string
	var activeRepositoryMemberList []*vcs.RepositoryMember
	for _, gitLabMember := range gitLabrepositoryMember {
		if gitLabMember.State == vcs.StateActive {
			// The email field will only be returned if the caller credential is associated with a GitLab admin account.
			// And since most callers are not GitLab admins, thus we fetch public email
			// TODO: need to work around this if the user does not set public email. For now, we just return an error listing users not having public emails.
			// TODO: if the number of the member is too large, fetching sequentially may cause performance issue
			userInfo, err := provider.FetchUserInfo(ctx, oauthCtx, "", gitLabMember.ID)
			if err != nil {
				return nil, err
			}
			if userInfo.PublicEmail == "" {
				emptyEmailUserIDList = append(emptyEmailUserIDList, gitLabMember.Name)
			}

			gitLabRole, bytebaseRole := getRoleAndMappedRole(gitLabMember.AccessLevel)
			repositoryMember := &vcs.RepositoryMember{
				Name:         gitLabMember.Name,
				Email:        userInfo.PublicEmail,
				Role:         bytebaseRole,
				VCSRole:      gitLabRole.String(),
				State:        vcs.StateActive,
				RoleProvider: vcs.GitLabSelfHost,
			}
			activeRepositoryMemberList = append(activeRepositoryMemberList, repositoryMember)
		}
	}

	if len(emptyEmailUserIDList) != 0 {
		return nil, fmt.Errorf("[ %v ] did not configure their public email in GitLab, please make sure every members' public email is configured before syncing, see https://docs.gitlab.com/ee/user/profile", strings.Join(emptyEmailUserIDList, ", "))
	}

	return activeRepositoryMemberList, nil
}

// CreateFile creates a file.
func (provider *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(FileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	code, _, err := httpPost(
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
		return fmt.Errorf("failed to create file %s on GitHub.com, err: %w", filePath, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to create file %s on GitHub.com, status code: %d",
			filePath,
			code,
		)
	}
	return nil
}

// OverwriteFile overwrite the content of a file.
func (provider *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(FileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
		LastCommitID:  fileCommitCreate.LastCommitID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	code, _, err := httpPut(
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
		return fmt.Errorf("failed to create file %s on GitHub.com, error: %w", filePath, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to create file %s on GitHub.com, status code: %d",
			filePath,
			code,
		)
	}
	return nil
}

// ReadFile reads the content of a file.
func (provider *Provider) ReadFile(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, filePath string, commitID string) (string, error) {
	code, body, err := httpGet(
		fmt.Sprintf("repos/%s/contents/%s?ref=%s", repositoryID, url.QueryEscape(filePath), commitID),
		&oauthCtx.AccessToken,
		oauthContext{
			ClientID:     oauthCtx.ClientID,
			ClientSecret: oauthCtx.ClientSecret,
			RefreshToken: oauthCtx.RefreshToken,
		},
		oauthCtx.Refresher,
	)

	if err != nil {
		return "", fmt.Errorf("failed to read file %s from GitHub.com: %w", filePath, err)
	}

	if code == 404 {
		return "", common.Errorf(common.NotFound, fmt.Errorf("failed to read file %s from GitHub.com, file not found", filePath))
	} else if code >= 300 {
		return "", fmt.Errorf("failed to read file %s from GitHub.com, status code: %d",
			filePath,
			code,
		)
	}

	return body, nil
}

// ReadFileMeta reads the metadata of a file.
func (provider *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, filePath string, branch string) (*vcs.FileMeta, error) {
	code, body, err := httpGet(
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
		return nil, fmt.Errorf("failed to read file meta %s from GitLab.com: %w", filePath, err)
	}

	if code == 404 {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to read file meta %s from GitHub.com, file not found", filePath))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read file meta %s from GitHub.com, status code: %d",
			filePath,
			code,
		)
	}

	file := &FileMeta{}
	if err := json.Unmarshal([]byte(body), file); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file meta from GitHub.com: %w", err)
	}

	return &vcs.FileMeta{
		LastCommitID: file.LastCommitID,
	}, nil
}

// CreateWebhook creates a webhook in a GitLab project.
func (provider *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, payload []byte) (string, error) {
	fmt.Println("repositoryID", repositoryID, oauthCtx.AccessToken)
	resourcePath := fmt.Sprintf("repos/%s/hooks", repositoryID)
	code, body, err := httpPost(
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
		return "", fmt.Errorf("failed to create webhook for repository %s from GitHub.com: %w", repositoryID, err)
	}

	if code >= 300 {
		reason := fmt.Sprintf(
			"failed to create webhook for repository %s from GitHub.com, status code: %d",
			repositoryID,
			code,
		)
		// Add helper tips if the status code is 422, refer to bytebase#101 for more context.
		if code == http.StatusUnprocessableEntity {
			reason += ".\n\nIf GitLab and Bytebase are in the same private network, " +
				"please follow the instructions in https://docs.gitlab.com/ee/security/webhooks.html"
		}
		return "", fmt.Errorf(reason)
	}

	webhookInfo := &WebhookInfo{}
	if err := json.Unmarshal([]byte(body), webhookInfo); err != nil {
		return "", fmt.Errorf("failed to unmarshal create webhook response for repository %s from GitHub.com: %w", repositoryID, err)
	}
	return strconv.Itoa(webhookInfo.ID), nil
}

// PatchWebhook patches a webhook in a GitLab project.
func (provider *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, webhookID string, payload []byte) error {
	resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repositoryID, webhookID)
	code, _, err := httpPut(
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
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitHub.com: %w", webhookID, repositoryID, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitHub.com, status code: %d", webhookID, repositoryID, code)
	}
	return nil
}

// DeleteWebhook deletes a webhook in a GitLab project.
func (provider *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, _ string, repositoryID string, webhookID string) error {
	resourcePath := fmt.Sprintf("projects/%s/hooks/%s", repositoryID, webhookID)
	code, _, err := httpDelete(
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
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitHub.com: %w", webhookID, repositoryID, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitHub.com, status code: %d", webhookID, repositoryID, code)
	}
	return nil
}

// httpPost sends a POST request.
func httpPost(resourcePath string, token *string, body io.Reader, oauthContext oauthContext, refresher common.TokenRefresher) (code int, respBody string, err error) {
	return retry(token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
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
func httpGet(resourcePath string, token *string, oauthContext oauthContext, refresher common.TokenRefresher) (code int, respBody string, err error) {
	return retry(token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
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
func httpPut(resourcePath string, token *string, body io.Reader, oauthContext oauthContext, refresher common.TokenRefresher) (code int, respBody string, err error) {
	return retry(token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
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
func httpDelete(resourcePath string, token *string, oauthContext oauthContext, refresher common.TokenRefresher) (code int, respBody string, err error) {
	return retry(token, oauthContext, refresher, func() (*http.Response, error) {
		url := fmt.Sprintf("%s/%s", apiURL, resourcePath)
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

func retry(token *string, oauthContext oauthContext, refresher common.TokenRefresher, f func() (*http.Response, error)) (code int, respBody string, err error) {
	retries := 0
RETRY:
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
			if err := refreshToken(token, oauthContext, refresher); err != nil {
				return 0, "", err
			}
			goto RETRY

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
