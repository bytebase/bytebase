package gitlab

import (
	"bytes"
	"context"
	"encoding/base64"
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

const (
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	// apiPath is the API path.
	apiPath = "api/v4"
)

var _ vcs.Provider = (*Provider)(nil)

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

// Commit is the API message for commit.
type Commit struct {
	ID         string `json:"id"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

// FileCommit is the API message for file commit.
type FileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
}

// RepositoryTreeNode is the API message for git tree node.
type RepositoryTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// File is the API message for file metadata.
type File struct {
	FileName     string `json:"file_name"`
	FilePath     string `json:"file_path"`
	Encoding     string `json:"encoding"`
	Content      string `json:"content"`
	Size         int64  `json:"size"`
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
	case ProjectRoleOwner:
		return "Owner"
	case ProjectRoleMaintainer:
		return "Maintainer"
	case ProjectRoleDeveloper:
		return "Developer"
	case ProjectRoleReporter:
		return "Reporter"
	case ProjectRoleGuest:
		return "Guest"
	case ProjectRoleMinimalAccess:
		return "MinimalAccess"
	case ProjectRoleNoAccess:
		return "NoAccess"
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

// gitLabRepository is the API message for repository in GitLab
type gitLabRepository struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

func init() {
	vcs.Register(vcs.GitLabSelfHost, newProvider)
}

// Provider is a GitLab self host VCS provider.
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

// APIURL returns the API URL path of a GitLab instance.
func (p *Provider) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/%s", instanceURL, apiPath)
}

// ExchangeOAuthToken exchange oauth content with the provided authentication code.
func (p *Provider) ExchangeOAuthToken(ctx context.Context, instanceURL string, oauthExchange *common.OAuthExchange) (*vcs.OAuthToken, error) {
	urlParams := &url.Values{}
	urlParams.Set("client_id", oauthExchange.ClientID)
	urlParams.Set("client_secret", oauthExchange.ClientSecret)
	urlParams.Set("code", oauthExchange.Code)
	urlParams.Set("redirect_uri", oauthExchange.RedirectURL)
	urlParams.Set("grant_type", "authorization_code")
	url := fmt.Sprintf("%s/oauth/token?%s", instanceURL, urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		urlParams.Set("client_secret", "**redacted**")
		redactedURL := fmt.Sprintf("%s/oauth/token?%s", instanceURL, urlParams.Encode())
		return nil, errors.Wrapf(err, "construct POST %s", redactedURL)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange Oauth Token, code %v, error: %v", resp.StatusCode, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read oauth response body, code %v, error: %v", resp.StatusCode, err)
	}

	oauthToken := &vcs.OAuthToken{}
	if err := json.Unmarshal(body, oauthToken); err != nil {
		return nil, fmt.Errorf("failed to unmarshal oauth response body, code %v, error: %v", resp.StatusCode, err)
	}

	// For GitLab, as of 13.12, the default config won't expire the access token,
	// thus this field is 0. See https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
	if oauthToken.ExpiresIn != 0 {
		oauthToken.ExpiresTs = oauthToken.CreatedAt + oauthToken.ExpiresIn
	}
	return oauthToken, nil
}

// FetchRepositoryList fetched all repositories in which the authenticated user
// has a maintainer role.
func (p *Provider) FetchRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	// We will use user's token to create webhook in the project, which requires the
	// token owner to be at least the project maintainer(40).
	url := fmt.Sprintf("%s/%s/projects?membership=true&simple=true&min_access_level=40", instanceURL, apiPath)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
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
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch repository list from GitLab instance %s", instanceURL))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read repository list from GitLab instance %s, status code: %d",
			instanceURL,
			code,
		)
	}

	var gitlabRepos []gitLabRepository
	if err := json.Unmarshal([]byte(body), &gitlabRepos); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	var repos []*vcs.Repository
	for _, r := range gitlabRepos {
		repo := &vcs.Repository{
			ID:       r.ID,
			Name:     r.Name,
			FullPath: r.PathWithNamespace,
			WebURL:   r.WebURL,
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// fetchUserInfo fetches user information from the given resourceURI, which
// should be either "user" or "users/{userID}".
func (p *Provider) fetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, instanceURL, resourceURI string) (*vcs.UserInfo, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, apiPath, resourceURI)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
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
		errInfo := []string{fmt.Sprintf("failed to fetch user info from GitLab instance %s", instanceURL)}
		resourceURISplit := strings.Split(resourceURI, "/")
		if len(resourceURI) > 1 {
			errInfo = append(errInfo, fmt.Sprintf("UserID: %s", resourceURISplit[1]))
		}
		return nil, common.Errorf(common.NotFound, fmt.Errorf(strings.Join(errInfo, ", ")))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read user info from GitLab instance %s, status code: %d",
			instanceURL,
			code,
		)
	}

	var userInfo vcs.UserInfo
	if err := json.Unmarshal([]byte(body), &userInfo); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &userInfo, err
}

// TryLogin tries to fetch the user info from the current OAuth context.
func (p *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, instanceURL, "user")
}

// FetchCommitByID fetch the commit data by id.
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("projects/%s/repository/commits/%s", repositoryID, commitID)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
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
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch commit data from GitLab instance %s, not found", instanceURL))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to fetch commit data from GitLab instance %s, status code: %d",
			instanceURL,
			code,
		)
	}

	commit := &Commit{}
	if err := json.Unmarshal([]byte(body), commit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commit data from GitLab instance %s, err: %w", instanceURL, err)
	}

	createdTime, err := time.Parse(time.RFC3339, commit.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit created_at field, err: %w", err)
	}

	return &vcs.Commit{
		ID:         commit.ID,
		AuthorName: commit.AuthorName,
		CreatedTs:  createdTime.Unix(),
	}, nil
}

// FetchUserInfo fetches user info of given user ID.
func (p *Provider) FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, instanceURL, userID string) (*vcs.UserInfo, error) {
	return p.fetchUserInfo(ctx, oauthCtx, instanceURL, fmt.Sprintf("users/%s", userID))
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
func (p *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string) ([]*vcs.RepositoryMember, error) {
	// Official API doc: https://docs.gitlab.com/14.6/ee/api/members.html
	url := fmt.Sprintf("%s/%s/projects/%s/members/all", instanceURL, apiPath, repositoryID)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
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

	if code == 404 {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch repository members from GitLab instance %s", instanceURL))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read repository members from GitLab instance %s, status code: %d",
			instanceURL,
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
			userInfo, err := p.FetchUserInfo(ctx, oauthCtx, instanceURL, strconv.Itoa(gitLabMember.ID))
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

// FetchRepositoryFileList fetch the files from repository tree
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	url := fmt.Sprintf("%s/%s/projects/%s/repository/tree?recursive=true&ref=%s&path=%s", instanceURL, apiPath, repositoryID, ref, filePath)
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository tree on GitLab instance %s, err: %w", instanceURL, err)
	}
	if code >= 300 {
		return nil, fmt.Errorf("failed to fetch repository tree on GitLab instance %s, status code: %d",
			instanceURL,
			code,
		)
	}

	var nodeList []*RepositoryTreeNode
	if err := json.Unmarshal([]byte(body), &nodeList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repository tree from GitLab instance %s, err: %w", instanceURL, err)
	}

	// Filter out folder nodes, we only need the file nodes.
	var fileList []*vcs.RepositoryTreeNode
	for _, node := range nodeList {
		if node.Type == "blob" {
			fileList = append(fileList, &vcs.RepositoryTreeNode{
				Path: node.Path,
				Type: node.Type,
			})
		}
	}

	return fileList, nil
}

// CreateFile creates a file.
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(FileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	url := fmt.Sprintf("%s/%s/projects/%s/repository/files/%s", instanceURL, apiPath, repositoryID, url.QueryEscape(filePath))
	code, _, err := oauth.Post(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(body),
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, err: %w", filePath, instanceURL, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			code,
		)
	}
	return nil
}

// OverwriteFile overwrite the content of a file.
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(FileCommit{
		Branch:        fileCommitCreate.Branch,
		CommitMessage: fileCommitCreate.CommitMessage,
		Content:       fileCommitCreate.Content,
		LastCommitID:  fileCommitCreate.LastCommitID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal file commit: %w", err)
	}

	url := fmt.Sprintf("%s/%s/projects/%s/repository/files/%s", instanceURL, apiPath, repositoryID, url.QueryEscape(filePath))
	code, _, err := oauth.Put(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(body),
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, error: %w", filePath, instanceURL, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to create file %s on GitLab instance %s, status code: %d",
			filePath,
			instanceURL,
			code,
		)
	}
	return nil
}

// ReadFileMeta reads the file metadata.
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, ref string) (*vcs.FileMeta, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to read file metadata %s from GitLab instance %s: %w", filePath, instanceURL, err)
	}

	return &vcs.FileMeta{
		Name:         file.FileName,
		Path:         file.FilePath,
		Size:         file.Size,
		LastCommitID: file.LastCommitID,
	}, nil
}

// ReadFileContent reads the file content.
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, ref string) (string, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return "", fmt.Errorf("failed to read file content %s from GitLab instance %s: %w", filePath, instanceURL, err)
	}

	return file.Content, nil
}

// CreateWebhook creates a webhook in a GitLab project.
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/%s/projects/%s/hooks", instanceURL, apiPath, repositoryID)
	code, body, err := oauth.Post(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(payload),
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create webhook for repository %s from GitLab instance %s: %w", repositoryID, instanceURL, err)
	}

	if code >= 300 {
		reason := fmt.Sprintf(
			"failed to create webhook for repository %s from GitLab instance %s, status code: %d",
			repositoryID,
			instanceURL,
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
		return "", fmt.Errorf("failed to unmarshal create webhook response for repository %s from GitLab instance %s: %w", repositoryID, instanceURL, err)
	}
	return strconv.Itoa(webhookInfo.ID), nil
}

// PatchWebhook patches a webhook in a GitLab project.
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	url := fmt.Sprintf("%s/%s/projects/%s/hooks/%s", instanceURL, apiPath, repositoryID, webhookID)
	code, _, err := oauth.Put(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		bytes.NewReader(payload),
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitLab instance %s: %w", webhookID, repositoryID, instanceURL, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to patch webhook ID %s for repository %s from GitLab instance %s, status code: %d", webhookID, repositoryID, instanceURL, code)
	}
	return nil
}

// DeleteWebhook deletes a webhook in a GitLab project.
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/%s/projects/%s/hooks/%s", instanceURL, apiPath, repositoryID, webhookID)
	code, _, err := oauth.Delete(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
			oauthContext{
				ClientID:     oauthCtx.ClientID,
				ClientSecret: oauthCtx.ClientSecret,
				RefreshToken: oauthCtx.RefreshToken,
			},
			oauthCtx.Refresher,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitLab instance %s: %w", webhookID, repositoryID, instanceURL, err)
	}

	if code >= 300 {
		return fmt.Errorf("failed to delete webhook ID %s for repository %s from GitLab instance %s, status code: %d", webhookID, repositoryID, instanceURL, code)
	}
	return nil
}

// readFile reads the file data including metadata and content.
func (p *Provider) readFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, ref string) (*File, error) {
	url := fmt.Sprintf("projects/%s/repository/files/%s?ref=%s", repositoryID, url.QueryEscape(filePath), url.QueryEscape(ref))
	code, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		tokenRefresher(
			instanceURL,
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
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to read file data from GitLab instance %s", instanceURL))
	} else if code >= 300 {
		return nil, fmt.Errorf("failed to read file data from GitLab instance %s, status code: %d",
			instanceURL,
			code,
		)
	}

	file := &File{}
	if err := json.Unmarshal([]byte(body), file); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file from GitLab instance %s: %w", instanceURL, err)
	}

	content := file.Content
	if file.Encoding == "base64" {
		decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file content, err %w", err)
		}
		content = string(decodedContent)
	}

	return &File{
		FileName:     file.FileName,
		FilePath:     file.FilePath,
		Size:         file.Size,
		Encoding:     file.Encoding,
		Content:      content,
		LastCommitID: file.LastCommitID,
	}, nil
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

func tokenRefresher(instanceURL string, oauthCtx oauthContext, refresher common.TokenRefresher) oauth.TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		url := fmt.Sprintf("%s/oauth/token", instanceURL)
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

		var r refreshOauthResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return errors.Wrapf(err, "unmarshal body from POST %s", url)
		}

		// Update the old token to new value for retries.
		*oldToken = r.AccessToken

		// For GitLab, as of 13.12, the default config won't expire the access token,
		// thus this field is 0. See https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
		var expireAt int64
		if r.ExpiresIn != 0 {
			expireAt = r.CreatedAt + r.ExpiresIn
		}
		if err := refresher(r.AccessToken, r.RefreshToken, expireAt); err != nil {
			return err
		}
		return nil
	}
}
