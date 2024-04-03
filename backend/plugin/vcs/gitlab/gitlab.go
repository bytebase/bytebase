// Package gitlab is the plugin for GitLab.
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var _ vcs.Provider = (*Provider)(nil)

func init() {
	vcs.Register(storepb.VCSType_GITLAB, newProvider)
}

const (
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16

	// apiPath is the API path.
	apiPath = "api/v4"
	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

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
	PushEvents          bool `json:"push_events"`
	NoteEvents          bool `json:"note_events"`
	MergeRequestsEvents bool `json:"merge_requests_events"`
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
	Content string
}

// Repository represents a GitLab API response for a repository.
type Repository struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

// Provider is a GitLab self host VCS provider.
type Provider struct {
	client      *http.Client
	instanceURL string
	authToken   string
}

func newProvider(config vcs.ProviderConfig) vcs.Provider {
	return &Provider{
		client:      &http.Client{},
		instanceURL: config.InstanceURL,
		authToken:   config.AuthToken,
	}
}

// APIURL returns the API URL path of a GitLab instance.
func (*Provider) APIURL(instanceURL string) string {
	return fmt.Sprintf("%s/%s", instanceURL, apiPath)
}

// FetchRepositoryList fetches all repositories where the authenticated user
// has a maintainer role, which is required to create webhook in the project.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#list-all-projects
func (p *Provider) FetchRepositoryList(ctx context.Context, listAll bool) ([]*vcs.Repository, error) {
	var gitlabRepos []Repository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		gitlabRepos = append(gitlabRepos, repos...)

		if !hasNextPage || !listAll {
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
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, page int) (repos []Repository, hasNextPage bool, err error) {
	// We will use user's token to create webhook in the project, which requires the
	// token owner to be at least the project maintainer(40).
	url := fmt.Sprintf("%s/projects?membership=true&simple=true&min_access_level=40&page=%d&per_page=%d", p.APIURL(p.instanceURL), page, apiPageSize)
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
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

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://docs.gitlab.com/ee/api/repository_files.html#get-file-from-repository
func (p *Provider) ReadFileContent(ctx context.Context, repositoryID, filePath string, refInfo vcs.RefInfo) (string, error) {
	file, err := p.readFile(ctx, repositoryID, filePath, refInfo)
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
// TODO(d): migrate to diff API.
// Docs: https://docs.gitlab.com/ee/api/merge_requests.html#get-single-mr-changes
func (p *Provider) ListPullRequestFile(ctx context.Context, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%s/changes", p.APIURL(p.instanceURL), repositoryID, pullRequestID)
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
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

// CreatePullRequestComment creates a pull request comment.
func (p *Provider) CreatePullRequestComment(ctx context.Context, repositoryID, pullRequestID, comment string) error {
	url := fmt.Sprintf("%s/projects/%s/merge_requests/%s/notes?body=%s", p.APIURL(p.instanceURL), repositoryID, pullRequestID, url.QueryEscape(comment))
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), nil)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create pull request comment through URL %s", url)
	}

	// GitLab returns 201 HTTP status codes upon successful issue comment creation,
	if code != http.StatusCreated {
		return errors.Errorf("failed to create pull request comment through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}
	return nil
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
func (p *Provider) GetBranch(ctx context.Context, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/projects/%s/repository/branches/%s", p.APIURL(p.instanceURL), repositoryID, branchName)
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
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

// MergeRequest is the API message for GitLab merge request.
type MergeRequest struct {
	WebURL string `json:"web_url"`
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#add-project-hook
func (p *Provider) CreateWebhook(ctx context.Context, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/hooks", p.APIURL(p.instanceURL), repositoryID)
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), payload)
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

// DeleteWebhook deletes the webhook from the repository.
//
// Docs: https://docs.gitlab.com/ee/api/projects.html#delete-project-hook
func (p *Provider) DeleteWebhook(ctx context.Context, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/projects/%s/hooks/%s", p.APIURL(p.instanceURL), repositoryID, webhookID)
	code, body, err := internal.Delete(ctx, url, p.getAuthorization())
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
func (p *Provider) readFile(ctx context.Context, repositoryID, filePath string, refInfo vcs.RefInfo) (*File, error) {
	// GitLab is often deployed behind a reverse proxy, which may have compression enabled that is transparent to the GitLab instance.
	// In such cases, the HTTP header "Content-Encoding" will, for example, be changed to "gzip" and makes the value of "Content-Length" untrustworthy.
	// We can avoid dealing with this type of problem by using the raw API instead of the typical JSON API.
	url := fmt.Sprintf("%s/projects/%s/repository/files/%s/raw?ref=%s", p.APIURL(p.instanceURL), repositoryID, url.QueryEscape(filePath), url.QueryEscape(refInfo.RefName))
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
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
		Content: body,
	}, nil
}

func (p *Provider) getAuthorization() string {
	return fmt.Sprintf("Bearer %s", p.authToken)
}
