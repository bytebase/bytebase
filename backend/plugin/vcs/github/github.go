// Package github is the plugin for GitHub.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// githubComURL is URL for the GitHub.com.
	githubComURL = "https://github.com"

	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

func init() {
	vcs.Register(storepb.VCSType_GITHUB, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a GitHub VCS provider.
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

// APIURL returns the API URL path of GitHub.
func (*Provider) APIURL(instanceURL string) string {
	if instanceURL == githubComURL {
		return "https://api.github.com"
	}

	// If it's not the GitHub.com, we use the API URL for the GitHub Enterprise Server.
	return fmt.Sprintf("%s/api/v3", instanceURL)
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

// CommitAuthor represents a GitHub API response for a commit author.
type CommitAuthor struct {
	// Date expects corresponding JSON value is a string in RFC 3339 format,
	// see https://pkg.go.dev/time#Time.MarshalJSON.
	Date  time.Time `json:"date"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// Commit represents a GitHub API response for a commit.
type Commit struct {
	SHA    string       `json:"sha"`
	Author CommitAuthor `json:"author"`
}

// FileCommit represents a GitHub API request for committing a file.
type FileCommit struct {
	Message string        `json:"message"`
	Content string        `json:"content"`
	SHA     string        `json:"sha,omitempty"`
	Branch  string        `json:"branch,omitempty"`
	Author  *CommitAuthor `json:"author,omitempty"`
}

// CommitsDiff represents a GitHub API response for comparing two commits.
type CommitsDiff struct {
	Files []PullRequestFile `json:"files"`
}

// FetchAllRepositoryList fetches all repositories where the authenticated user
// has admin permissions, which is required to create webhook in the repository.
//
// NOTE: GitHub API does not provide a native filter for admin permissions, thus
// we need to first fetch all repositories and then filter down the list using
// the `permissions.admin` field.
//
// Docs: https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
func (p *Provider) FetchAllRepositoryList(ctx context.Context) ([]*vcs.Repository, error) {
	var githubRepos []Repository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, page)
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
				ID:       strconv.FormatInt(r.ID, 10),
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
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, page int) (repos []Repository, hasNextPage bool, err error) {
	url := fmt.Sprintf("%s/user/repos?page=%d&per_page=%d", p.APIURL(p.instanceURL), page, apiPageSize)
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
// Docs: https://docs.github.com/en/rest/repos/contents#get-repository-content
func (p *Provider) ReadFileContent(ctx context.Context, repositoryID, filePath string, refInfo vcs.RefInfo) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", p.APIURL(p.instanceURL), repositoryID, url.QueryEscape(filePath), refInfo.RefName)
	code, body, err := internal.GetWithHeader(ctx, url, p.getAuthorization(),
		map[string]string{
			"Accept": "application/vnd.github.raw",
		},
	)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return "", common.Errorf(common.NotFound, "failed to read file content from URL %s", url)
	} else if code >= 300 {
		return "",
			errors.Errorf("failed to read file content from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}
	return body, nil
}

// PullRequestFile is the API message for files in GitHub pull request.
type PullRequestFile struct {
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
func (p *Provider) ListPullRequestFile(ctx context.Context, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	var allPRFiles []PullRequestFile
	page := 1
	for {
		fileList, err := p.listPaginatedPullRequestFile(ctx, repositoryID, pullRequestID, page)
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
			slog.Debug("Failed to parse content url for file",
				slog.String("content_url", file.ContentsURL),
				slog.String("file", file.FileName),
				log.BBError(err),
			)
			continue
		}

		m, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			slog.Debug("Failed to parse query for file",
				slog.String("content_url", file.ContentsURL),
				slog.String("file", file.FileName),
				log.BBError(err),
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
func (p *Provider) listPaginatedPullRequestFile(ctx context.Context, repositoryID, pullRequestID string, page int) ([]PullRequestFile, error) {
	requestURL := fmt.Sprintf("%s/repos/%s/pulls/%s/files?per_page=%d&page=%d", p.APIURL(p.instanceURL), repositoryID, pullRequestID, apiPageSize, page)
	code, body, err := internal.Get(ctx, requestURL, p.getAuthorization())
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

	var prFiles []PullRequestFile
	if err := json.Unmarshal([]byte(body), &prFiles); err != nil {
		return nil, err
	}
	return prFiles, nil
}

type Comment struct {
	Body string `json:"body"`
}

// CreatePullRequestComment creates a comment on the pull request.
//
// Issue comment makes comment on the pull request (Yes, you read it right).
// Pull request comment makes a pull request comment on the line.
// Pull request review makes a pull request review such as approval.
// Docs: https://docs.github.com/en/rest/issues/comments?apiVersion=2022-11-28#create-an-issue-comment
func (p *Provider) CreatePullRequestComment(ctx context.Context, repositoryID, pullRequestID, comment string) error {
	commentMessage := Comment{Body: comment}
	commentCreatePayload, err := json.Marshal(commentMessage)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body for creating pull request comment")
	}
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", p.APIURL(p.instanceURL), repositoryID, pullRequestID)
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), commentCreatePayload)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create pull request comment through URL %s", url)
	}

	// GitHub returns 201 HTTP status codes upon successful issue comment creation,
	if code != http.StatusCreated {
		return errors.Errorf("failed to create pull request comment through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}
	return nil
}

// Branch is the API message for GitHub branch.
type Branch struct {
	Ref    string          `json:"ref"`
	Object ReferenceObject `json:"object"`
}

// ReferenceObject is the reference for the GitHub branch.
type ReferenceObject struct {
	SHA string `json:"sha"`
}

// GetBranch gets the given branch in the repository.
//
// Docs: https://docs.github.com/en/rest/git/refs#get-a-reference
func (p *Provider) GetBranch(ctx context.Context, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/git/ref/heads/%s", p.APIURL(p.instanceURL), repositoryID, branchName)
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

	res := new(Branch)
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

// PullRequest is the API message for GitHub pull request.
type PullRequest struct {
	HTMLURL string `json:"html_url"`
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://docs.github.com/en/rest/webhooks/repos#create-a-repository-webhook
func (p *Provider) CreateWebhook(ctx context.Context, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/hooks", p.APIURL(p.instanceURL), repositoryID)
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), payload)
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

// DeleteWebhook deletes the webhook from the repository.
//
// Docs: https://docs.github.com/en/rest/webhooks/repos#delete-a-repository-webhook
func (p *Provider) DeleteWebhook(ctx context.Context, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/repos/%s/hooks/%s", p.APIURL(p.instanceURL), repositoryID, webhookID)
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

func (p *Provider) getAuthorization() string {
	return fmt.Sprintf("Bearer %s", p.authToken)
}
