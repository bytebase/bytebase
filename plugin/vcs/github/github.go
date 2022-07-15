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

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/plugin/vcs/internal/oauth"
)

const (
	// apiURL is the API URL.
	apiURL = "https://api.github.com"
	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

func init() {
	vcs.Register(vcs.GitHubCom, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a GitHub.com VCS provider.
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
func (p *Provider) APIURL(string) string {
	return apiURL
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
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
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
		// Date expects corresponding JSON value is a string in RFC 3339 format,
		// see https://pkg.go.dev/time#Time.MarshalJSON.
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
func (p *Provider) FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID string) ([]*vcs.RepositoryMember, error) {
	var allCollaborators []RepositoryCollaborator
	page := 1
	for {
		collaborators, hasNextPage, err := p.fetchPaginatedRepositoryCollaborators(ctx, oauthCtx, repositoryID, page)
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
		userInfo, err := p.FetchUserInfo(ctx, oauthCtx, "", c.Login)
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
		return nil, fmt.Errorf("[ %v ] did not configure their public email in GitHub, please make sure every members' public email is configured before syncing, see https://docs.github.com/en/account-and-profile", strings.Join(emptyEmailUserList, ", "))
	}

	return allMembers, nil
}

// fetchPaginatedRepositoryCollaborators fetches collaborators of a repository
// in given page. It return the paginated results along with a boolean
// indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryCollaborators(ctx context.Context, oauthCtx common.OauthContext, repositoryID string, page int) (collaborators []RepositoryCollaborator, hasNextPage bool, err error) {
	url := fmt.Sprintf("%s/repos/%s/collaborators?page=%d&per_page=%d", apiURL, repositoryID, page, apiPageSize)
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
		return nil, false, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, false, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch repository collaborators from URL %s", url))
	} else if code >= 300 {
		return nil, false,
			fmt.Errorf("failed to read repository collaborators from URL %s, status code: %d, body: %s",
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
	return collaborators, len(collaborators) >= 100, nil
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

// FetchAllRepositoryList fetches all repositories where the authenticated user
// has a owner role, which is required to create webhook in the repository.
//
// Docs: https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, _ string) ([]*vcs.Repository, error) {
	var githubRepos []Repository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, oauthCtx, page)
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
// user has a owner role in given page. It return the paginated results along
// with a boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, oauthCtx common.OauthContext, page int) (repos []Repository, hasNextPage bool, err error) {
	// We will use user's token to create webhook in the project, which requires the
	// token owner to be at least the project maintainer(40).
	url := fmt.Sprintf("%s/user/repos?affiliation=owner&page=%d&per_page=%d", apiURL, page, apiPageSize)
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
		return nil, false, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, false, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch repository list from URL %s", url))
	} else if code >= 300 {
		return nil, false,
			fmt.Errorf("failed to fetch repository list from URL %s, status code: %d, body: %s",
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
	return repos, len(repos) >= 100, nil
}

// FetchRepositoryFileList fetches the all files from the given repository tree
// recursively.
//
// Docs: https://docs.github.com/en/rest/git/trees#get-a-tree
//
// TODO: GitHub returns truncated response if the number of items in the tree
// array exceeded their maximum limit. It is not noted what exactly is the
// maximum limit and requires making non-recursive request to each sub-tree.
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, _, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	url := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=true", apiURL, repositoryID, ref)
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
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("failed to fetch repository file list from URL %s", url))
	} else if code >= 300 {
		return nil,
			fmt.Errorf("failed to fetch repository file list from URL %s, status code: %d, body: %s",
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
		if err = refresher(r.AccessToken, r.RefreshToken, expireAt); err != nil {
			return err
		}
		return nil
	}
}
