// Package github is the plugin for GitHub.
package github

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/box"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/internal/oauth"
)

const (
	// githubComURL is URL for the GitHub.com.
	githubComURL = "https://github.com"

	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

func init() {
	vcs.Register(vcs.GitHubCom, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a GitHub VCS provider.
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

// APIURL returns the API URL path of GitHub.
func (*Provider) APIURL(instanceURL string) string {
	if instanceURL == githubComURL {
		return "https://api.github.com"
	}

	// If it's not the GitHub.com, we use the API URL for the GitHub Enterprise Server.
	return fmt.Sprintf("%s/api/v3", instanceURL)
}

// RepositoryRole is the role of the repository collaborator.
type RepositoryRole string

// The list of GitHub roles.
const (
	RepositoryRoleAdmin    RepositoryRole = "admin"
	RepositoryRoleMaintain RepositoryRole = "maintain"
	RepositoryRoleWrite    RepositoryRole = "write"
	RepositoryRoleTriage   RepositoryRole = "triage"
	RepositoryRoleRead     RepositoryRole = "read"
)

// RepositoryCollaborator represents a GitHub API response for a repository
// collaborator.
type RepositoryCollaborator struct {
	Login    string `json:"login"`
	RoleName string `json:"role_name"`
}

// User represents a GitHub API response for a user.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Repository represents a GitHub API response for a repository.
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Permissions struct {
		Admin bool `json:"admin"`
	} `json:"permissions"`
}

// RepositoryTree represents a GitHub API response for a repository tree.
type RepositoryTree struct {
	Tree []RepositoryTreeNode `json:"tree"`
}

// RepositoryTreeNode represents a GitHub API response for a repository tree
// node.
type RepositoryTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// File represents a GitHub API response for a repository file.
type File struct {
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	SHA      string `json:"sha"`
}

// WebhookType is the GitHub webhook type.
type WebhookType string

const (
	// WebhookPush is the webhook type for push.
	WebhookPush WebhookType = "push"
	// WebhookPing is the webhook type for ping.
	WebhookPing WebhookType = "ping"
)

// WebhookInfo represents a GitHub API response for the webhook information.
type WebhookInfo struct {
	ID int `json:"id"`
}

// WebhookConfig represents the GitHub API message for webhook configuration.
type WebhookConfig struct {
	// URL is the URL to which the payloads will be delivered.
	URL string `json:"url"`
	// ContentType is the media type used to serialize the payloads. Supported
	// values include "json" and "form". The default is "form".
	ContentType string `json:"content_type"`
	// Secret is the secret will be used as the key to generate the HMAC hex digest
	// value for delivery signature headers.
	Secret string `json:"secret"`
	// InsecureSSL determines whether the SSL certificate of the host for url will
	// be verified when delivering payloads. Supported values include 0
	// (verification is performed) and 1 (verification is not performed). The
	// default is 0.
	InsecureSSL int `json:"insecure_ssl"`
}

// WebhookCreateOrUpdate represents a GitHub API request for creating or
// updating a webhook.
//
// NOTE: GitHub uses different API payloads for creating and updating webhooks
// (the latter has more options), but we are not using any differentiated parts
// so it makes sense to have a combined struct until we needed.
type WebhookCreateOrUpdate struct {
	// Config contains settings for the webhook.
	Config WebhookConfig `json:"config"`
	// Events determines what events the hook is triggered for. The default is
	// ["push"]. The full list of events can be viewed at
	// https://docs.github.com/webhooks/event-payloads.
	Events []string `json:"events"`
}

// WebhookRepository is the API message for webhook repository.
type WebhookRepository struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// WebhookCommitAuthor is the API message for webhook commit author.
type WebhookCommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// WebhookSender is the API message for webhook sender.
type WebhookSender struct {
	Login string `json:"login"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	ID        string              `json:"id"`
	Distinct  bool                `json:"distinct"`
	Message   string              `json:"message"`
	Timestamp time.Time           `json:"timestamp"`
	URL       string              `json:"url"`
	Author    WebhookCommitAuthor `json:"author"`
	Added     []string            `json:"added"`
	Modified  []string            `json:"modified"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	Ref        string            `json:"ref"`
	Repository WebhookRepository `json:"repository"`
	Sender     WebhookSender     `json:"sender"`
	Commits    []WebhookCommit   `json:"commits"`
}

// fetchUserInfoImpl fetches user information from the given resourceURI, which
// should be either "user" or "users/{username}".
func (p *Provider) fetchUserInfoImpl(ctx context.Context, oauthCtx common.OauthContext, instanceURL, resourceURI string) (*vcs.UserInfo, error) {
	url := fmt.Sprintf("%s/%s", p.APIURL(instanceURL), resourceURI)
	code, _, body, err := oauth.Get(
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
		return nil, common.Errorf(common.NotFound, "failed to read user info from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to read user info from URL %s, status code: %d, body: %s", url, code, body)
	}

	var user User
	if err = json.Unmarshal([]byte(body), &user); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &vcs.UserInfo{
		PublicEmail: user.Email,
		Name:        user.Name,
		State:       vcs.StateActive,
	}, err
}

// TryLogin tries to fetch the user info from the current OAuth context.
func (p *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) (*vcs.UserInfo, error) {
	return p.fetchUserInfoImpl(ctx, oauthCtx, instanceURL, "user")
}

// CommitAuthor represents a GitHub API response for a commit author.
type CommitAuthor struct {
	// Date expects corresponding JSON value is a string in RFC 3339 format,
	// see https://pkg.go.dev/time#Time.MarshalJSON.
	Date time.Time `json:"date"`
	Name string    `json:"name"`
}

// Commit represents a GitHub API response for a commit.
type Commit struct {
	SHA    string       `json:"sha"`
	Author CommitAuthor `json:"author"`
}

// FileCommit represents a GitHub API request for committing a file.
type FileCommit struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

// FetchCommitByID fetches the commit data by its ID from the repository.
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("%s/repos/%s/git/commits/%s", p.APIURL(instanceURL), repositoryID, commitID)
	code, _, body, err := oauth.Get(
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
		return nil, common.Errorf(common.NotFound, "failed to fetch commit data from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to fetch commit data from URL %s, status code: %d, body: %s", url, code, body)
	}

	commit := &Commit{}
	if err := json.Unmarshal([]byte(body), commit); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	return &vcs.Commit{
		ID:         commit.SHA,
		AuthorName: commit.Author.Name,
		CreatedTs:  commit.Author.Date.Unix(),
	}, nil
}

// FetchUserInfo fetches user info of given user ID.
func (p *Provider) FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, instanceURL, username string) (*vcs.UserInfo, error) {
	return p.fetchUserInfoImpl(ctx, oauthCtx, instanceURL, fmt.Sprintf("users/%s", username))
}

func getRoleAndMappedRole(roleName string) (githubRole RepositoryRole, bytebaseRole common.ProjectRole) {
	// Please refer to https://docs.github.com/en/organizations/managing-access-to-your-organizations-repositories/repository-roles-for-an-organization#repository-roles-for-organizations
	// for the detailed role descriptions of GitHub.
	switch roleName {
	case "admin":
		return RepositoryRoleAdmin, common.ProjectOwner
	case "maintain":
		return RepositoryRoleMaintain, common.ProjectOwner
	case "write":
		return RepositoryRoleWrite, common.ProjectOwner
	case "triage":
		return RepositoryRoleTriage, common.ProjectDeveloper
	case "read":
		return RepositoryRoleRead, common.ProjectDeveloper
	}
	return "", ""
}

// FetchRepositoryActiveMemberList fetch all active members of a repository
//
// Docs: https://docs.github.com/en/rest/collaborators/collaborators#list-repository-collaborators
func (p *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string) ([]*vcs.RepositoryMember, error) {
	var allCollaborators []RepositoryCollaborator
	page := 1
	for {
		collaborators, hasNextPage, err := p.fetchPaginatedRepositoryCollaborators(ctx, oauthCtx, instanceURL, repositoryID, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		allCollaborators = append(allCollaborators, collaborators...)

		if !hasNextPage {
			break
		}
		page++
	}

	var emptyEmailUserList []string
	var allMembers []*vcs.RepositoryMember
	for _, c := range allCollaborators {
		userInfo, err := p.FetchUserInfo(ctx, oauthCtx, githubComURL, c.Login)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch user info, login: %s", c.Login)
		}

		if userInfo.PublicEmail == "" {
			emptyEmailUserList = append(emptyEmailUserList, userInfo.Name)
			continue
		}

		githubRole, bytebaseRole := getRoleAndMappedRole(c.RoleName)
		allMembers = append(allMembers,
			&vcs.RepositoryMember{
				Name:         userInfo.Name,
				Email:        userInfo.PublicEmail,
				Role:         bytebaseRole,
				VCSRole:      string(githubRole),
				State:        vcs.StateActive,
				RoleProvider: vcs.GitHubCom,
			},
		)
	}

	if len(emptyEmailUserList) != 0 {
		return nil, errors.Errorf("[ %v ] did not configure their public email in GitHub, please make sure every members' public email is configured before syncing, see https://docs.github.com/en/account-and-profile", strings.Join(emptyEmailUserList, ", "))
	}

	return allMembers, nil
}

// fetchPaginatedRepositoryCollaborators fetches collaborators of a repository
// in given page. It return the paginated results along with a boolean
// indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryCollaborators(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, page int) (collaborators []RepositoryCollaborator, hasNextPage bool, err error) {
	url := fmt.Sprintf("%s/repos/%s/collaborators?page=%d&per_page=%d", p.APIURL(instanceURL), repositoryID, page, apiPageSize)
	code, _, body, err := oauth.Get(
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
		return nil, false, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, false, common.Errorf(common.NotFound, "failed to fetch repository collaborators from URL %s", url)
	} else if code >= 300 {
		return nil, false,
			errors.Errorf("failed to read repository collaborators from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	if err := json.Unmarshal([]byte(body), &collaborators); err != nil {
		return nil, false, errors.Wrap(err, "unmarshal body")
	}

	// NOTE: We deliberately choose to not use the Link header for checking the next
	// page to avoid introducing a new dependency, see
	// https://github.com/bytebase/bytebase/pull/1423#discussion_r884278534 for the
	// discussion.
	return collaborators, len(collaborators) >= apiPageSize, nil
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
func (p *Provider) ExchangeOAuthToken(ctx context.Context, instanceURL string, oauthExchange *common.OAuthExchange) (*vcs.OAuthToken, error) {
	urlParams := &url.Values{}
	urlParams.Set("client_id", oauthExchange.ClientID)
	urlParams.Set("client_secret", oauthExchange.ClientSecret)
	urlParams.Set("code", oauthExchange.Code)
	urlParams.Set("redirect_uri", oauthExchange.RedirectURL)
	url := fmt.Sprintf("%s/login/oauth/access_token?%s", instanceURL, urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		urlParams.Set("client_secret", "**redacted**")
		redactedURL := fmt.Sprintf("%s/login/oauth/access_token?%s", instanceURL, urlParams.Encode())
		return nil, errors.Wrapf(err, "construct POST %s", redactedURL)
	}

	// GitHub returns URL-encoded parameters as the response format by default,
	// we need to ask for a JSON response explicitly.
	req.Header.Set("Accept", "application/json")

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
	return oauthResp.toVCSOAuthToken(), nil
}

// FetchAllRepositoryList fetches all repositories where the authenticated user
// has admin permissions, which is required to create webhook in the repository.
//
// NOTE: GitHub API does not provide a native filter for admin permissions, thus
// we need to first fetch all repositories and then filter down the list using
// the `permissions.admin` field.
//
// Docs: https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	var githubRepos []Repository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, oauthCtx, instanceURL, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		githubRepos = append(githubRepos, repos...)

		if !hasNextPage {
			break
		}
		page++
	}

	var allRepos []*vcs.Repository
	for _, r := range githubRepos {
		if !r.Permissions.Admin {
			continue
		}
		allRepos = append(allRepos,
			&vcs.Repository{
				ID:       r.ID,
				Name:     r.Name,
				FullPath: r.FullName,
				WebURL:   r.HTMLURL,
			},
		)
	}
	return allRepos, nil
}

// fetchPaginatedRepositoryList fetches repositories where the authenticated
// user has access to in given page. It returns the paginated results along
// with a boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, page int) (repos []Repository, hasNextPage bool, err error) {
	url := fmt.Sprintf("%s/user/repos?page=%d&per_page=%d", p.APIURL(instanceURL), page, apiPageSize)
	code, _, body, err := oauth.Get(
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
		return nil, false, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, false, common.Errorf(common.NotFound, "failed to fetch repository list from URL %s", url)
	} else if code >= 300 {
		return nil, false,
			errors.Errorf("failed to fetch repository list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	if err := json.Unmarshal([]byte(body), &repos); err != nil {
		return nil, false, errors.Wrap(err, "unmarshal")
	}

	// NOTE: We deliberately choose to not use the Link header for checking the next
	// page to avoid introducing a new dependency, see
	// https://github.com/bytebase/bytebase/pull/1423#discussion_r884278534 for the
	// discussion.
	return repos, len(repos) >= apiPageSize, nil
}

// FetchRepositoryFileList fetches the all files from the given repository tree
// recursively.
//
// Docs: https://docs.github.com/en/rest/git/trees#get-a-tree
//
// TODO: GitHub returns truncated response if the number of items in the tree
// array exceeded their maximum limit. It is not noted what exactly is the
// maximum limit and requires making non-recursive request to each sub-tree.
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	url := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=true", p.APIURL(instanceURL), repositoryID, ref)
	code, _, body, err := oauth.Get(
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to fetch repository file list from URL %s", url)
	} else if code >= 300 {
		return nil,
			errors.Errorf("failed to fetch repository file list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	var repoTree RepositoryTree
	if err := json.Unmarshal([]byte(body), &repoTree); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	if filePath != "" && !strings.HasSuffix(filePath, "/") {
		filePath += "/"
	}

	var allTreeNodes []*vcs.RepositoryTreeNode
	for _, n := range repoTree.Tree {
		// GitHub does not support filtering by path prefix, thus simulating the
		// behavior here.
		if n.Type == "blob" && strings.HasPrefix(n.Path, filePath) {
			allTreeNodes = append(allTreeNodes,
				&vcs.RepositoryTreeNode{
					Path: n.Path,
					Type: n.Type,
				},
			)
		}
	}
	return allTreeNodes, nil
}

// CreateFile creates a file at given path in the repository.
//
// Docs: https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(
		FileCommit{
			Message: fileCommitCreate.CommitMessage,
			Content: base64.StdEncoding.EncodeToString([]byte(fileCommitCreate.Content)),
			Branch:  fileCommitCreate.Branch,
			SHA:     fileCommitCreate.LastCommitID,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal file commit")
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.APIURL(instanceURL), repositoryID, url.QueryEscape(filePath))
	code, _, resp, err := oauth.Put(
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
		return errors.Wrapf(err, "PUT %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create/update file through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to create/update file through URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// OverwriteFile overwrites an existing file at given path in the repository.
//
// Docs: https://docs.github.com/en/rest/repos/contents#create-or-update-file-contents
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.CreateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate)
}

// ReadFileMeta reads the metadata of the given file in the repository.
//
// Docs: https://docs.github.com/en/rest/repos/contents#get-repository-content
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*vcs.FileMeta, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	return &vcs.FileMeta{
		Name:         file.Name,
		Path:         file.Path,
		Size:         file.Size,
		LastCommitID: file.SHA,
	}, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://docs.github.com/en/rest/repos/contents#get-repository-content
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (string, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return "", errors.Wrap(err, "read file")
	}
	return file.Content, nil
}

// readFile reads the given file in the repository.
func (p *Provider) readFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*File, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", p.APIURL(instanceURL), repositoryID, url.QueryEscape(filePath), ref)
	code, _, body, err := oauth.Get(
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to read file from URL %s", url)
	} else if code >= 300 {
		return nil,
			errors.Errorf("failed to read file from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	// This API endpoint returns a JSON array if the path is a directory, and we do
	// not want that.
	if body != "" && body[0] == '[' {
		return nil, errors.Errorf("%q is a directory not a file", filePath)
	}

	var file File
	if err = json.Unmarshal([]byte(body), &file); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	if file.Encoding == "base64" {
		decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, errors.Wrap(err, "decode file content")
		}
		file.Content = string(decodedContent)
	}
	return &file, nil
}

type githubPullRequestFile struct {
	FileName string `json:"filename"`
	SHA      string `json:"sha"`
	// The file status in GitHub PR.
	// Available values: "added", "removed", "modified", "renamed", "copied", "changed", "unchanged"
	Status string `json:"status"`
	// The file content API URL, which contains the ref value in the query.
	// Example: https://api.github.com/repos/octocat/Hello-World/contents/file1.txt?ref=6dcb09b5b57875f334f61aebed695e2e4193db5e
	ContentsURL string `json:"contents_url"`
}

// ListPullRequestFile lists the changed files in the pull request.
//
// Docs: https://docs.github.com/en/rest/pulls/pulls#list-pull-requests-files
func (p *Provider) ListPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	var allPRFiles []githubPullRequestFile
	page := 1
	for {
		fileList, err := p.listPaginatedPullRequestFile(ctx, oauthCtx, instanceURL, repositoryID, pullRequestID, page)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list pull request file")
		}

		if len(fileList) == 0 {
			break
		}
		allPRFiles = append(allPRFiles, fileList...)
		page++
	}

	var res []*vcs.PullRequestFile
	for _, file := range allPRFiles {
		u, err := url.Parse(file.ContentsURL)
		if err != nil {
			log.Debug("Failed to parse content url for file",
				zap.String("content_url", file.ContentsURL),
				zap.String("file", file.FileName),
				zap.Error(err),
			)
			continue
		}
		m, _ := url.ParseQuery(u.RawQuery)
		if err != nil {
			log.Debug("Failed to parse query for file",
				zap.String("content_url", file.ContentsURL),
				zap.String("file", file.FileName),
				zap.Error(err),
			)
			continue
		}
		refs, ok := m["ref"]
		if !ok || len(refs) != 1 {
			continue
		}

		res = append(res, &vcs.PullRequestFile{
			Path:         file.FileName,
			LastCommitID: refs[0],
			IsDeleted:    file.Status == "removed",
		})
	}

	return res, nil
}

// listPaginatedPullRequestFile lists the changed files in the pull request with pagination.
func (p *Provider) listPaginatedPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string, page int) ([]githubPullRequestFile, error) {
	requestURL := fmt.Sprintf("%s/repos/%s/pulls/%s/files?per_page=%d&page=%d", p.APIURL(instanceURL), repositoryID, pullRequestID, apiPageSize, page)
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		requestURL,
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
		return nil, errors.Wrapf(err, "GET %s", requestURL)
	}
	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to list pull request file from URL %s", requestURL)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to list pull request file from URL %s, status code: %d, body: %s",
			requestURL,
			code,
			body,
		)
	}

	var prFiles []githubPullRequestFile
	if err := json.Unmarshal([]byte(body), &prFiles); err != nil {
		return nil, err
	}
	return prFiles, nil
}

type githubBranchCreate struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type githubBranch struct {
	Ref    string          `json:"ref"`
	Object referenceObject `json:"object"`
}

type referenceObject struct {
	SHA string `json:"sha"`
}

// GetBranch gets the given branch in the repository.
//
// Docs: https://docs.github.com/en/rest/git/refs#get-a-reference
func (p *Provider) GetBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/git/ref/heads/%s", p.APIURL(instanceURL), repositoryID, branchName)
	code, _, body, err := oauth.Get(
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to create branch from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to create branch from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	res := new(githubBranch)
	if err := json.Unmarshal([]byte(body), res); err != nil {
		return nil, err
	}

	name, err := vcs.Branch(res.Ref)
	if err != nil {
		return nil, err
	}

	return &vcs.BranchInfo{
		Name:         name,
		LastCommitID: res.Object.SHA,
	}, nil
}

// CreateBranch creates the branch in the repository.
//
// Docs: https://docs.github.com/en/rest/git/refs#create-a-reference
func (p *Provider) CreateBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, branch *vcs.BranchInfo) error {
	body, err := json.Marshal(
		githubBranchCreate{
			Ref: fmt.Sprintf("refs/heads/%s", branch.Name),
			SHA: branch.LastCommitID,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal branch create")
	}

	url := fmt.Sprintf("%s/repos/%s/git/refs", p.APIURL(instanceURL), repositoryID)
	code, _, resp, err := oauth.Post(
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
		return errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create branch from URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to create branch from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}

	return nil
}

type githubPullRequest struct {
	HTMLURL string `json:"html_url"`
}

// CreatePullRequest creates the pull request in the repository.
//
// Docs: https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request
func (p *Provider) CreatePullRequest(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, pullRequestCreate *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
	body, err := json.Marshal(pullRequestCreate)
	if err != nil {
		return nil, errors.Wrap(err, "marshal pull request create")
	}

	url := fmt.Sprintf("%s/repos/%s/pulls", p.APIURL(instanceURL), repositoryID)
	code, _, resp, err := oauth.Post(
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to create pull request from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to create pull request from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}

	var res githubPullRequest
	if err := json.Unmarshal([]byte(resp), &res); err != nil {
		return nil, err
	}

	return &vcs.PullRequest{
		URL: res.HTMLURL,
	}, nil
}

type environmentVariable struct {
	EncryptedValue string `json:"encrypted_value"`
	KeyID          string `json:"key_id"`
}

// UpsertEnvironmentVariable creates or updates the environment variable in the repository.
//
// https://docs.github.com/en/rest/actions/secrets#create-or-update-a-repository-secret
func (p *Provider) UpsertEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, key, value string) error {
	// We have to encrypt the secret value using the public key in the repository.
	// Docs: https://docs.github.com/en/rest/actions/secrets#example-encrypting-a-secret-using-nodejs
	publicKey, err := p.getRepositoryPublicKey(ctx, oauthCtx, instanceURL, repositoryID)
	if err != nil {
		return errors.Wrapf(err, "Failed to get public key")
	}
	encryptValue, err := encryptEnvironmentVariable(publicKey.Key, value)
	if err != nil {
		return errors.Wrapf(err, "Failed to encrypt environment variable")
	}

	body, err := json.Marshal(
		environmentVariable{
			KeyID:          publicKey.KeyID,
			EncryptedValue: encryptValue,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal environment variable")
	}

	url := fmt.Sprintf("%s/repos/%s/actions/secrets/%s", p.APIURL(instanceURL), repositoryID, key)
	code, _, resp, err := oauth.Put(
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
		return errors.Wrapf(err, "PUT %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to upsert environment variable from URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to upsert environment variable from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}

	return nil
}

// encryptEnvironmentVariable encrypt the value with public key
//
// https://github.com/jefflinse/githubsecret
func encryptEnvironmentVariable(publicKey, value string) (string, error) {
	const keySize = 32
	const nonceSize = 24

	// decode the provided public key from base64
	recipientKey := new([keySize]byte)
	b, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", err
	} else if size := len(b); size != keySize {
		return "", errors.Errorf("Public key has invalid length, expect %d bytes but found %d", keySize, size)
	}

	copy(recipientKey[:], b)

	// create an ephemeral key pair
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	// create the nonce by hashing together the two public keys
	nonce := new([nonceSize]byte)
	nonceHash, err := blake2b.New(nonceSize, nil)
	if err != nil {
		return "", err
	}

	if _, err := nonceHash.Write(pubKey[:]); err != nil {
		return "", err
	}

	if _, err := nonceHash.Write(recipientKey[:]); err != nil {
		return "", err
	}

	copy(nonce[:], nonceHash.Sum(nil))

	// begin the output with the ephemeral public key and append the encrypted content
	out := box.Seal(pubKey[:], []byte(value), nonce, recipientKey, privKey)

	// base64-encode the final output
	return base64.StdEncoding.EncodeToString(out), nil
}

type repositoryPublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

// getRepositoryPublicKey returns the public key in the GitHub repository.
//
// https://docs.github.com/en/rest/actions/secrets#get-a-repository-public-key
func (p *Provider) getRepositoryPublicKey(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string) (*repositoryPublicKey, error) {
	url := fmt.Sprintf("%s/repos/%s/actions/secrets/public-key", p.APIURL(instanceURL), repositoryID)
	code, _, body, err := oauth.Get(
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to get repo public key from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to get repo public key from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	res := new(repositoryPublicKey)
	if err := json.Unmarshal([]byte(body), res); err != nil {
		return nil, err
	}

	return res, nil
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://docs.github.com/en/rest/webhooks/repos#create-a-repository-webhook
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/hooks", p.APIURL(instanceURL), repositoryID)
	code, _, body, err := oauth.Post(
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
		return "", errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return "", common.Errorf(common.NotFound, "failed to create webhook through URL %s", url)
	}

	// GitHub returns 201 HTTP status codes upon successful webhook creation,
	// see https://docs.github.com/en/rest/webhooks/repos#create-a-repository-webhook for details.
	if code != http.StatusCreated {
		return "", errors.Errorf("failed to create webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var webhookInfo WebhookInfo
	if err = json.Unmarshal([]byte(body), &webhookInfo); err != nil {
		return "", errors.Wrap(err, "unmarshal body")
	}
	return strconv.Itoa(webhookInfo.ID), nil
}

// PatchWebhook patches the webhook in the repository with given payload.
//
// Docs: https://docs.github.com/en/rest/webhooks/repos#update-a-repository-webhook
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	url := fmt.Sprintf("%s/repos/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
	code, _, body, err := oauth.Patch(
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
		return errors.Wrapf(err, "PATCH %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to patch webhook through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to patch webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}
	return nil
}

// DeleteWebhook deletes the webhook from the repository.
//
// Docs: https://docs.github.com/en/rest/webhooks/repos#delete-a-repository-webhook
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/repos/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
	code, _, body, err := oauth.Delete(
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
		return errors.Wrapf(err, "DELETE %s", url)
	}

	if code == http.StatusNotFound {
		return nil // It is OK if the webhook has already gone
	} else if code >= 300 {
		return errors.Errorf("failed to delete webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
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

func tokenRefresher(instanceURL string, oauthCtx oauthContext, refresher common.TokenRefresher) oauth.TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		url := fmt.Sprintf("%s/login/oauth/access_token", instanceURL)
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
			return errors.Errorf("non-200 POST %s status code %d with body %q", url, resp.StatusCode, body)
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
		return refresher(r.AccessToken, r.RefreshToken, expireAt)
	}
}

// ToVCS returns the push event in VCS format.
func (p WebhookPushEvent) ToVCS() vcs.PushEvent {
	var commitList []vcs.Commit
	for _, commit := range p.Commits {
		// The Distinct is false if the commit has not been pushed before.
		if !commit.Distinct {
			continue
		}
		// Per Git convention, the message title and body are separated by two new line characters.
		messages := strings.SplitN(commit.Message, "\n\n", 2)
		messageTitle := messages[0]

		commitList = append(commitList, vcs.Commit{
			ID:           commit.ID,
			Title:        messageTitle,
			Message:      commit.Message,
			CreatedTs:    commit.Timestamp.Unix(),
			URL:          commit.URL,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			AddedList:    commit.Added,
			ModifiedList: commit.Modified,
		})
	}
	return vcs.PushEvent{
		Ref:                p.Ref,
		RepositoryID:       p.Repository.FullName,
		RepositoryURL:      p.Repository.HTMLURL,
		RepositoryFullPath: p.Repository.FullName,
		AuthorName:         p.Sender.Login,
		CommitList:         commitList,
	}
}
