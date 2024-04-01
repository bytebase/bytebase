// Package bitbucket is the plugin for Bitbucket Cloud.
package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal"
)

const (
	// bitbucketCloudURL is URL for the Bitbucket Cloud.
	bitbucketCloudURL = "https://bitbucket.org"

	// apiPageSize is the default page size when making API requests.
	apiPageSize = 100
)

func init() {
	vcs.Register(vcs.Bitbucket, newProvider)
}

var _ vcs.Provider = (*Provider)(nil)

// Provider is a Bitbucket Cloud VCS provider.
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

// APIURL returns the API URL path of Bitbucket Cloud.
func (*Provider) APIURL(instanceURL string) string {
	if instanceURL == bitbucketCloudURL {
		return "https://api.bitbucket.org/2.0"
	}
	return fmt.Sprintf("%s/2.0", instanceURL)
}

// User represents a Bitbucket Cloud API response for a user.
type User struct {
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
}

// CommitAuthor represents a Bitbucket Cloud API response for a commit author.
type CommitAuthor struct {
	User User `json:"user"`
}

// Commit represents a Bitbucket Cloud API response for a commit.
type Commit struct {
	Hash   string       `json:"hash"`
	Author CommitAuthor `json:"author"`
	// Date expects corresponding JSON value is a string in RFC 3339 format,
	// see https://pkg.go.dev/time#Time.MarshalJSON.
	Date time.Time `json:"date"`
}

// CommitFile represents a Bitbucket Cloud API response for a file at a commit.
type CommitFile struct {
	Path  string `json:"path"`
	Links Links  `json:"links"`
}

// CommitDiffStat represents a Bitbucket Cloud API response for commit diff stat.
type CommitDiffStat struct {
	// The status of the diff stat object, possible values are "added", "removed",
	// "modified", "renamed".
	Status string     `json:"status"`
	New    CommitFile `json:"new"`
}

type PullRequestResponse struct {
	Values []*CommitDiffStat `json:"values"`
	Next   string            `json:"next"`
}

func (p *Provider) fetchPaginatedDiffFileList(ctx context.Context, url string) (diffs []*CommitDiffStat, next string, err error) {
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, "", errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, "", common.Errorf(common.NotFound, "failed to get file diff list from URL %s", url)
	} else if code >= 300 {
		return nil, "", errors.Errorf("failed to get file diff list from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var resp PullRequestResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, "", errors.Wrapf(err, "failed to unmarshal file diff data from Bitbucket Cloud instance %s", url)
	}
	return resp.Values, resp.Next, nil
}

// Repository represents a Bitbucket Cloud API response for a repository.
type Repository struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Links    Links  `json:"links"`
}

// RepositoryPermission represents a Bitbucket Cloud API response for a
// repository permission.
type RepositoryPermission struct {
	Repository Repository `json:"repository"`
}

// FetchAllRepositoryList fetches all repositories where the authenticated user
// has admin permissions, which is required to create webhook in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-user-permissions-repositories-get
func (p *Provider) FetchAllRepositoryList(ctx context.Context) ([]*vcs.Repository, error) {
	var bbcRepos []*Repository
	params := url.Values{}
	params.Add("q", `permission="admin"`)
	params.Add("pagelen", strconv.Itoa(apiPageSize))
	next := fmt.Sprintf(`%s/user/permissions/repositories?%s`, p.APIURL(p.instanceURL), params.Encode())
	for next != "" {
		var err error
		var repos []*Repository
		repos, next, err = p.fetchPaginatedRepositoryList(ctx, next)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcRepos = append(bbcRepos, repos...)
	}

	var repos []*vcs.Repository
	for _, r := range bbcRepos {
		repos = append(repos,
			&vcs.Repository{
				ID:       r.UUID,
				Name:     r.Name,
				FullPath: r.FullName,
				WebURL:   fmt.Sprintf("%s/%s", p.instanceURL, r.FullName),
			},
		)
	}
	return repos, nil
}

// fetchPaginatedRepositoryList fetches repositories in given page. It returns
// the paginated results along with a string indicating the URL of the next page
// (if exists).
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, url string) (repos []*Repository, next string, err error) {
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return nil, "", errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, "", common.Errorf(common.NotFound, "failed to fetch repository list from URL %s", url)
	} else if code >= 300 {
		return nil, "",
			errors.Errorf("failed to fetch repository list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	var resp struct {
		Values []*RepositoryPermission `json:"values"`
		Next   string                  `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, "", errors.Wrap(err, "unmarshal")
	}

	for _, v := range resp.Values {
		repos = append(repos,
			&Repository{
				UUID:     v.Repository.UUID,
				Name:     v.Repository.Name,
				FullName: v.Repository.FullName,
			},
		)
	}
	return repos, resp.Next, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#raw-file-contents
func (p *Provider) ReadFileContent(ctx context.Context, repositoryID, filePath string, refInfo vcs.RefInfo) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s", p.APIURL(p.instanceURL), repositoryID, url.PathEscape(refInfo.RefName), url.PathEscape(filePath))
	code, body, err := internal.Get(ctx, url, p.getAuthorization())
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return "", common.Errorf(common.NotFound, "failed to read file from URL %s", url)
	} else if code >= 300 {
		return "",
			errors.Errorf("failed to read file from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}
	return body, nil
}

// Target is the API message for Bitbucket Cloud target.
type Target struct {
	Hash string `json:"hash"`
}

// Branch is the API message for Bitbucket Cloud branch.
type Branch struct {
	Name   string `json:"name"`
	Target Target `json:"target"`
}

// GetBranch gets the given branch in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-refs/#api-repositories-workspace-repo-slug-refs-branches-name-get
func (p *Provider) GetBranch(ctx context.Context, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/repositories/%s/refs/branches/%s", p.APIURL(p.instanceURL), repositoryID, branchName)
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

	var branch Branch
	if err := json.Unmarshal([]byte(body), &branch); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	return &vcs.BranchInfo{
		Name:         branch.Name,
		LastCommitID: branch.Target.Hash,
	}, nil
}

// ListPullRequestFile lists the changed files in the pull request.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/#api-repositories-workspace-repo-slug-pullrequests-pull-request-id-diffstat-get
func (p *Provider) ListPullRequestFile(ctx context.Context, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	var bbcDiffs []*CommitDiffStat
	next := fmt.Sprintf("%s/repositories/%s/pullrequests/%s/diffstat?pagelen=%d", p.APIURL(p.instanceURL), repositoryID, pullRequestID, apiPageSize)
	for next != "" {
		var err error
		var diffs []*CommitDiffStat
		diffs, next, err = p.fetchPaginatedDiffFileList(ctx, next)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcDiffs = append(bbcDiffs, diffs...)
	}

	// NOTE: The API response does not guarantee to return the value of the commit
	// ID, so we need to extract it from the link instead.
	extractCommitIDFromLinkSelf := func(href string) string {
		const anchor = "/src/"
		i := strings.Index(href, anchor)
		if i < 0 {
			return "<no commit ID found>"
		}
		fields := strings.SplitN(href[i+len(anchor):], "/", 2)
		return fields[0]
	}

	var files []*vcs.PullRequestFile
	for _, d := range bbcDiffs {
		file := &vcs.PullRequestFile{
			Path:         d.New.Path,
			LastCommitID: extractCommitIDFromLinkSelf(d.New.Links.Self.Href),
			IsDeleted:    d.Status == "removed",
		}
		files = append(files, file)
	}
	return files, nil
}

// CreatePullRequestComment creates a pull request comment.
func (*Provider) CreatePullRequestComment(_ context.Context, _, _, _ string) error {
	return nil
}

// Link is the API message for link.
type Link struct {
	Href string `json:"href"`
}

// Links is the API message for links.
type Links struct {
	Self Link `json:"self"`
	HTML Link `json:"html"`
}

// Author is the API message for author.
type Author struct {
	Raw  string `json:"raw"`
	User User   `json:"user"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	Hash    string    `json:"hash"`
	Date    time.Time `json:"date"`
	Author  Author    `json:"author"`
	Message string    `json:"message"`
	Links   Links     `json:"links"`
	Parents []Target  `json:"parents"`
}

// WebhookPushChange is the API message for webhook push change.
type WebhookPushChange struct {
	Old     Branch          `json:"old"`
	New     Branch          `json:"new"`
	Commits []WebhookCommit `json:"commits"`
}

// WebhookPush is the API message for webhook push.
type WebhookPush struct {
	Changes []WebhookPushChange `json:"changes"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	Push       WebhookPush `json:"push"`
	Repository Repository  `json:"repository"`
	Actor      User        `json:"actor"`
}

// WebhookCreateOrUpdate represents a Bitbucket API request for creating or
// updating a webhook.
type WebhookCreateOrUpdate struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
}

// Webhook represents a Bitbucket Cloud API response for the webhook
// information.
type Webhook struct {
	UUID string `json:"uuid"`
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-post
func (p *Provider) CreateWebhook(ctx context.Context, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/hooks", p.APIURL(p.instanceURL), repositoryID)
	code, body, err := internal.Post(ctx, url, p.getAuthorization(), payload)
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return "", common.Errorf(common.NotFound, "failed to create webhook through URL %s", url)
	}

	// Bitbucket Cloud returns 201 HTTP status codes upon successful webhook creation,
	// see https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-post-responses for details.
	if code != http.StatusCreated {
		return "", errors.Errorf("failed to create webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var resp Webhook
	if err = json.Unmarshal([]byte(body), &resp); err != nil {
		return "", errors.Wrap(err, "unmarshal body")
	}
	return resp.UUID, nil
}

// DeleteWebhook deletes the webhook from the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-uid-delete
func (p *Provider) DeleteWebhook(ctx context.Context, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/repositories/%s/hooks/%s", p.APIURL(p.instanceURL), repositoryID, webhookID)
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
	return fmt.Sprintf("Basic %s", p.authToken)
}
