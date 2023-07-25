// Package gitlab is the plugin for GitLab.
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
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
)

const (
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	// apiPath is the API path.
	apiPath = "api/v4"
	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

var _ vcs.Provider = (*Provider)(nil)

// WebhookType is the GitLab webhook type.
type WebhookType string

const (
	// WebhookPush is the webhook type for push.
	WebhookPush WebhookType = "push"
)

// WebhookInfo represents a GitLab API response for the webhook information.
type WebhookInfo struct {
	ID int `json:"id"`
}

// WebhookCreate represents a GitLab API request for creating a new webhook.
type WebhookCreate struct {
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
	EnableSSLVerification bool `json:"enable_ssl_verification"`
}

// WebhookProject is the API message for webhook project.
type WebhookProject struct {
	ID       int    `json:"id"`
	WebURL   string `json:"web_url"`
	FullPath string `json:"path_with_namespace"`
}

// WebhookCommitAuthor is the API message for webhook commit author.
type WebhookCommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	Message      string              `json:"message"`
	Timestamp    string              `json:"timestamp"`
	URL          string              `json:"url"`
	Author       WebhookCommitAuthor `json:"author"`
	AddedList    []string            `json:"added"`
	ModifiedList []string            `json:"modified"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	ObjectKind WebhookType     `json:"object_kind"`
	Ref        string          `json:"ref"`
	Before     string          `json:"before"`
	After      string          `json:"after"`
	AuthorName string          `json:"user_name"`
	Project    WebhookProject  `json:"project"`
	CommitList []WebhookCommit `json:"commits"`
}

// Commit is the API message for commit.
type Commit struct {
	ID         string `json:"id"`
	AuthorName string `json:"author_name"`
	// CreatedAt expects corresponding JSON value is a string in RFC 3339 format,
	// see https://pkg.go.dev/time#Time.MarshalJSON.
	CreatedAt time.Time `json:"created_at"`
}

// FileCommit represents a GitLab API request for committing a file.
type FileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	AuthorEmail   string `json:"author_email,omitempty"`
}

// RepositoryTreeNode represents a GitLab API response for a repository tree
// node.
type RepositoryTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// File represents a GitLab API response for a repository file.
type File struct {
	Content      string
	LastCommitID string
}

// CommitsDiff represents a GitLab API response for comparing two commits.
type CommitsDiff struct {
	FileDiffList []MergeRequestFile `json:"diffs"`
}

// gitLabRepository represents a GitLab API response for a repository.
type gitLabRepository struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

func init() {
	vcs.Register(vcs.GitLab, newProvider)
}

// Provider is a GitLab self host VCS provider.
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

// APIURL returns the API URL path of a GitLab instance.
func (*Provider) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/%s", instanceURL, apiPath)
}

// oauthResponse is a GitLab OAuth response.
type oauthResponse struct {
	AccessToken      string `json:"access_token" `
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"`
	CreatedAt        int64  `json:"created_at"`
	ExpiresTs        int64  `json:"expires_ts"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// toVCSOAuthToken converts the response to *vcs.OAuthToken.
func (o oauthResponse) toVCSOAuthToken() *vcs.OAuthToken {
	oauthToken := &vcs.OAuthToken{
		AccessToken:  o.AccessToken,
		RefreshToken: o.RefreshToken,
		ExpiresIn:    o.ExpiresIn,
		CreatedAt:    o.CreatedAt,
		ExpiresTs:    o.ExpiresTs,
	}
	// For GitLab, as of 13.12, the default config won't expire the access token,
	// thus this field is 0. See https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
	if oauthToken.ExpiresIn != 0 {
		oauthToken.ExpiresTs = oauthToken.CreatedAt + oauthToken.ExpiresIn
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
	urlParams.Set("grant_type", "authorization_code")
	url := fmt.Sprintf("%s/oauth/token?%s", instanceURL, urlParams.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		urlParams.Set("client_secret", "**redacted**")
		redactedURL := fmt.Sprintf("%s/oauth/token?%s", instanceURL, urlParams.Encode())
		return nil, errors.Wrapf(err, "construct POST %s", redactedURL)
	}

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
// has a maintainer role, which is required to create webhook in the project.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#list-all-projects
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	var gitlabRepos []gitLabRepository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, oauthCtx, instanceURL, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		gitlabRepos = append(gitlabRepos, repos...)

		if !hasNextPage {
			break
		}
		page++
	}

	var allRepos []*vcs.Repository
	for _, r := range gitlabRepos {
		allRepos = append(allRepos,
			&vcs.Repository{
				ID:       strconv.FormatInt(r.ID, 10),
				Name:     r.Name,
				FullPath: r.PathWithNamespace,
				WebURL:   r.WebURL,
			},
		)
	}
	return allRepos, nil
}

// fetchPaginatedRepositoryList fetches repositories where the authenticated
// user has a maintainer role in given page. It return the paginated results
// along with a boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, page int) (repos []gitLabRepository, hasNextPage bool, err error) {
	// We will use user's token to create webhook in the project, which requires the
	// token owner to be at least the project maintainer(40).
	url := fmt.Sprintf("%s/projects?membership=true&simple=true&min_access_level=40&page=%d&per_page=%d", p.APIURL(instanceURL), page, apiPageSize)
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

// fetchUserInfoImpl fetches user information from the given resourceURI, which
// should be either "user" or "users/{userID}".
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
		return nil, errors.Errorf("failed to read user info from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var userInfo vcs.UserInfo
	if err := json.Unmarshal([]byte(body), &userInfo); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &userInfo, err
}

// FetchCommitByID fetches the commit data by its ID from the repository.
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("%s/projects/%s/repository/commits/%s", p.APIURL(instanceURL), repositoryID, commitID)
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
		return nil, errors.Errorf("failed to fetch commit data from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	commit := &Commit{}
	if err := json.Unmarshal([]byte(body), commit); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal commit data from GitLab instance %s", instanceURL)
	}

	return &vcs.Commit{
		ID:         commit.ID,
		AuthorName: commit.AuthorName,
		CreatedTs:  commit.CreatedAt.Unix(),
	}, nil
}

// GetDiffFileList gets the diff files list between two commits.
func (p *Provider) GetDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, beforeCommit, afterCommit string) ([]vcs.FileDiff, error) {
	url := fmt.Sprintf("%s/projects/%s/repository/compare?from=%s&to=%s", p.APIURL(instanceURL), repositoryID, beforeCommit, afterCommit)
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
		return nil, common.Errorf(common.NotFound, "failed to get diff file list from URL %s", url)
	} else if code >= 300 {
		return nil,
			errors.Errorf("failed to get diff file list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	diffs := &CommitsDiff{}
	if err := json.Unmarshal([]byte(body), diffs); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file diff data from GitLab instance %s", instanceURL)
	}

	var ret []vcs.FileDiff
	for _, diff := range diffs.FileDiffList {
		item := vcs.FileDiff{
			Path: diff.NewPath,
		}
		if diff.NewFile {
			item.Type = vcs.FileDiffTypeAdded
		} else if diff.DeletedFile {
			item.Type = vcs.FileDiffTypeRemoved
		} else {
			item.Type = vcs.FileDiffTypeModified
		}
		ret = append(ret, item)
	}
	return ret, nil
}

// FetchRepositoryFileList fetches the all files from the given repository tree
// recursively.
//
// Docs: https://docs.gitlab.com/ee/api/repositories.html#list-repository-tree
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	var gitlabTreeNodes []RepositoryTreeNode
	page := 1
	for {
		treeNodes, hasNextPage, err := p.fetchPaginatedRepositoryFileList(ctx, oauthCtx, instanceURL, repositoryID, ref, filePath, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		gitlabTreeNodes = append(gitlabTreeNodes, treeNodes...)

		if !hasNextPage {
			break
		}
		page++
	}

	var allTreeNodes []*vcs.RepositoryTreeNode
	for _, n := range gitlabTreeNodes {
		if n.Type == "blob" {
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

// fetchPaginatedRepositoryFileList fetches files under a repository tree
// recursively in given page. It return the paginated results along with a
// boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string, page int) (treeNodes []RepositoryTreeNode, hasNextPage bool, err error) {
	url := fmt.Sprintf("%s/projects/%s/repository/tree?recursive=true&ref=%s&path=%s&page=%d&per_page=%d", p.APIURL(instanceURL), repositoryID, ref, filePath, page, apiPageSize)
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
		return nil, false, common.Errorf(common.NotFound, "failed to fetch repository file list from URL %s", url)
	} else if code >= 300 {
		return nil, false,
			errors.Errorf("failed to fetch repository file list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	if err := json.Unmarshal([]byte(body), &treeNodes); err != nil {
		return nil, false, errors.Wrap(err, "unmarshal body")
	}

	// NOTE: We deliberately choose to not use the Link header for checking the next
	// page to avoid introducing a new dependency, see
	// https://github.com/bytebase/bytebase/pull/1423#discussion_r884278534 for the
	// discussion.
	return treeNodes, len(treeNodes) >= apiPageSize, nil
}

// CreateFile creates a file at given path in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/repository_files.html#create-new-file-in-repository
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(
		FileCommit{
			Branch:        fileCommitCreate.Branch,
			CommitMessage: fileCommitCreate.CommitMessage,
			Content:       fileCommitCreate.Content,
			AuthorName:    fileCommitCreate.AuthorName,
			AuthorEmail:   fileCommitCreate.AuthorEmail,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal file commit")
	}

	url := fmt.Sprintf("%s/projects/%s/repository/files/%s", p.APIURL(instanceURL), repositoryID, url.QueryEscape(filePath))
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
		return errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create file through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to create file through URL %s, status code: %d, response body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// OverwriteFile overwrites an existing file at given path in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/repository_files.html#update-existing-file-in-repository
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	body, err := json.Marshal(
		FileCommit{
			Branch:        fileCommitCreate.Branch,
			Content:       fileCommitCreate.Content,
			CommitMessage: fileCommitCreate.CommitMessage,
			LastCommitID:  fileCommitCreate.LastCommitID,
			AuthorName:    fileCommitCreate.AuthorName,
			AuthorEmail:   fileCommitCreate.AuthorEmail,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal file commit")
	}

	url := fmt.Sprintf("%s/projects/%s/repository/files/%s", p.APIURL(instanceURL), repositoryID, url.QueryEscape(filePath))
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
		return common.Errorf(common.NotFound, "failed to overwrite file through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to overwrite file through URL %s, status code: %d, response body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// ReadFileMeta reads the metadata of the given file in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*vcs.FileMeta, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	return &vcs.FileMeta{
		LastCommitID: file.LastCommitID,
	}, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (string, error) {
	file, err := p.readFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, ref)
	if err != nil {
		return "", errors.Wrap(err, "read file")
	}
	return file.Content, nil
}

// MergeRequestChange is the API message for GitLab merge request changes.
type MergeRequestChange struct {
	SHA     string             `json:"sha"`
	Changes []MergeRequestFile `json:"changes"`
}

// MergeRequestFile is the API message for files in GitLab merge request.
type MergeRequestFile struct {
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// ListPullRequestFile lists the changed files in the pull request.
//
// Docs: https://docs.gitlab.com/ee/api/merge_requests.html#get-single-mr-changes
func (p *Provider) ListPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%s/changes", p.APIURL(instanceURL), repositoryID, pullRequestID)
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
		return nil, common.Errorf(common.NotFound, "failed to list merge request file from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to list merge request file from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	pr := new(MergeRequestChange)
	if err := json.Unmarshal([]byte(body), pr); err != nil {
		return nil, err
	}

	var res []*vcs.PullRequestFile
	for _, file := range pr.Changes {
		res = append(res, &vcs.PullRequestFile{
			Path:         file.NewPath,
			LastCommitID: pr.SHA,
			IsDeleted:    file.DeletedFile,
		})
	}

	return res, nil
}

// Branch is the API message for GitLab branch.
type Branch struct {
	Name   string `json:"name"`
	Commit Commit `json:"commit"`
}

// BranchCreate is the API message to create the branch.
type BranchCreate struct {
	Branch string `json:"branch"`
	Ref    string `json:"ref"`
}

// MergeRequestCreate is the API message to create the merge request.
type MergeRequestCreate struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	RemoveSourceBranch bool   `json:"remove_source_branch"`
}

// GetBranch gets the given branch in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/branches.html#get-single-repository-branch
func (p *Provider) GetBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s", p.APIURL(instanceURL), repositoryID, branchName)
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
		return nil, common.Errorf(common.NotFound, "failed to get branch from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to get branch from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	branch := new(Branch)
	if err := json.Unmarshal([]byte(body), branch); err != nil {
		return nil, err
	}

	return &vcs.BranchInfo{
		Name:         branch.Name,
		LastCommitID: branch.Commit.ID,
	}, nil
}

// CreateBranch creates the branch in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/branches.html#create-repository-branch
func (p *Provider) CreateBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, branch *vcs.BranchInfo) error {
	body, err := json.Marshal(
		BranchCreate{
			Branch: branch.Name,
			Ref:    branch.LastCommitID,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal branch create")
	}

	url := fmt.Sprintf("%s/projects/%s/repository/branches", p.APIURL(instanceURL), repositoryID)
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

// MergeRequest is the API message for GitLab merge request.
type MergeRequest struct {
	WebURL string `json:"web_url"`
}

// CreatePullRequest creates the pull request in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/merge_requests.html#create-mr
func (p *Provider) CreatePullRequest(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, pullRequestCreate *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
	body, err := json.Marshal(
		MergeRequestCreate{
			Title:              pullRequestCreate.Title,
			Description:        pullRequestCreate.Body,
			SourceBranch:       pullRequestCreate.Head,
			TargetBranch:       pullRequestCreate.Base,
			RemoveSourceBranch: pullRequestCreate.RemoveHeadAfterMerged,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "marshal pull request create")
	}

	url := fmt.Sprintf("%s/projects/%s/merge_requests", p.APIURL(instanceURL), repositoryID)
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
		return nil, common.Errorf(common.NotFound, "failed to create merge request from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to create merge request from URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}

	var res MergeRequest
	if err := json.Unmarshal([]byte(resp), &res); err != nil {
		return nil, err
	}

	return &vcs.PullRequest{
		URL: res.WebURL,
	}, nil
}

// EnvironmentVariable is the API message for environment variable in GitLab project.
type EnvironmentVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UpsertEnvironmentVariable creates or updates the environment variable in the repository.
func (p *Provider) UpsertEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, key, value string) error {
	_, err := p.getEnvironmentVariable(ctx, oauthCtx, instanceURL, repositoryID, key)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return p.createEnvironmentVariable(ctx, oauthCtx, instanceURL, repositoryID, key, value)
		}

		return err
	}

	return p.updateEnvironmentVariable(ctx, oauthCtx, instanceURL, repositoryID, key, value)
}

// getEnvironmentVariable gets the environment variable in the repository.
//
// https://docs.gitlab.com/ee/api/project_level_variables.html#get-a-single-variable
func (p *Provider) getEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, key string) (*EnvironmentVariable, error) {
	url := fmt.Sprintf("%s/projects/%s/variables/%s", p.APIURL(instanceURL), repositoryID, key)
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

	variable := new(EnvironmentVariable)
	if err := json.Unmarshal([]byte(body), variable); err != nil {
		return nil, err
	}

	return variable, nil
}

// createEnvironmentVariable creates the environment variable in the repository.
//
// https://docs.gitlab.com/ee/api/project_level_variables.html#create-a-variable
func (p *Provider) createEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, key, value string) error {
	url := fmt.Sprintf("%s/projects/%s/variables", p.APIURL(instanceURL), repositoryID)
	body, err := json.Marshal(
		EnvironmentVariable{
			Key:   key,
			Value: value,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal environment create")
	}
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
		return errors.Wrapf(err, "POST %s", url)
	}
	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create environment variable through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to create environment variable through URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// updateEnvironmentVariable updates the environment variable in the repository.
//
// https://docs.gitlab.com/ee/api/project_level_variables.html#update-a-variable
func (p *Provider) updateEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, key, value string) error {
	url := fmt.Sprintf("%s/projects/%s/variables/%s", p.APIURL(instanceURL), repositoryID, key)
	body, err := json.Marshal(
		EnvironmentVariable{
			Key:   key,
			Value: value,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal environment create")
	}
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
		return common.Errorf(common.NotFound, "failed to update environment variable through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to update environment variable through URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#add-project-hook
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/hooks", p.APIURL(instanceURL), repositoryID)
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
	// GitLab returns 201 HTTP status codes upon successful webhook creation,
	// see https://docs.gitlab.com/ee/api/#status-codes for details.
	if code != http.StatusCreated {
		reason := fmt.Sprintf("failed to create webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
		// Add helper tips if the status code is 422, refer to https://github.com/bytebase/bytebase/issues/101 for more context.
		if code == http.StatusUnprocessableEntity {
			reason += ".\n\nIf GitLab and Bytebase are in the same private network, " +
				"please follow the instructions in https://docs.gitlab.com/ee/security/webhooks.html"
		}
		return "", errors.New(reason)
	}

	var webhookInfo WebhookInfo
	if err = json.Unmarshal([]byte(body), &webhookInfo); err != nil {
		return "", errors.Wrap(err, "unmarshal body")
	}
	return strconv.Itoa(webhookInfo.ID), nil
}

// PatchWebhook patches the webhook in the repository with given payload.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#edit-project-hook
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	url := fmt.Sprintf("%s/projects/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
	code, _, body, err := oauth.Put(
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
		return errors.Wrapf(err, "PUT %s", url)
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
// Docs: https://docs.gitlab.com/ee/api/projects.html#delete-project-hook
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/projects/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
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

// readFile reads the given file in the repository.
//
// TODO: The same GitLab API endpoint supports using the HEAD request to only
// get the file metadata.
func (p *Provider) readFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*File, error) {
	// GitLab is often deployed behind a reverse proxy, which may have compression enabled that is transparent to the GitLab instance.
	// In such cases, the HTTP header "Content-Encoding" will, for example, be changed to "gzip" and makes the value of "Content-Length" untrustworthy.
	// We can avoid dealing with this type of problem by using the raw API instead of the typical JSON API.
	url := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s", p.APIURL(instanceURL), repositoryID, url.QueryEscape(filePath), url.QueryEscape(ref))
	code, header, body, err := oauth.Get(
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

	return &File{
		Content:      body,
		LastCommitID: header.Get("x-gitlab-last-commit-id"),
	}, nil
}

// oauthContext is the request context for refreshing oauth token.
type oauthContext struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	GrantType    string
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
		params := &url.Values{}
		params.Set("client_id", oauthCtx.ClientID)
		params.Set("client_secret", oauthCtx.ClientSecret)
		params.Set("refresh_token", oauthCtx.RefreshToken)
		params.Set("grant_type", "refresh_token")

		url := fmt.Sprintf("%s/oauth/token", instanceURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(params.Encode()))
		if err != nil {
			return errors.Wrapf(err, "construct POST %s", url)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "POST %s", url)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "read body of POST %s", url)
		}

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("non-200 POST %s status code %d with body %q", url, resp.StatusCode, body)
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
		return refresher(r.AccessToken, r.RefreshToken, expireAt)
	}
}

// ToVCS returns the push event in VCS format.
func (p WebhookPushEvent) ToVCS() (vcs.PushEvent, error) {
	var commitList []vcs.Commit
	for _, commit := range p.CommitList {
		createdTime, err := time.Parse(time.RFC3339, commit.Timestamp)
		if err != nil {
			return vcs.PushEvent{}, errors.Wrapf(err, "failed to parse commit %q's timestamp %q", commit.ID, commit.Timestamp)
		}
		commitList = append(commitList, vcs.Commit{
			ID:           commit.ID,
			Title:        commit.Title,
			Message:      commit.Message,
			CreatedTs:    createdTime.Unix(),
			URL:          commit.URL,
			AuthorName:   commit.Author.Name,
			AuthorEmail:  commit.Author.Email,
			AddedList:    commit.AddedList,
			ModifiedList: commit.ModifiedList,
		})
	}
	return vcs.PushEvent{
		VCSType:            vcs.GitLab,
		Ref:                p.Ref,
		Before:             p.Before,
		After:              p.After,
		RepositoryID:       fmt.Sprintf("%v", p.Project.ID),
		RepositoryURL:      p.Project.WebURL,
		RepositoryFullPath: p.Project.FullPath,
		AuthorName:         p.AuthorName,
		CommitList:         commitList,
	}, nil
}
