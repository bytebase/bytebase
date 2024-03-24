// Package bitbucket is the plugin for Bitbucket Cloud.
package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/plugin/vcs/internal/oauth"
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

// FetchCommitByID fetches the commit data by its ID from the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#api-repositories-workspace-repo-slug-commit-commit-get
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("%s/repositories/%s/commit/%s", p.APIURL(instanceURL), url.PathEscape(repositoryID), commitID)
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
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
		ID:         commit.Hash,
		AuthorName: commit.Author.User.DisplayName,
		CreatedTs:  commit.Date.Unix(),
	}, nil
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

// GetDiffFileList gets the diff files list between two commits.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#api-repositories-workspace-repo-slug-diffstat-spec-get
func (p *Provider) GetDiffFileList(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, beforeCommit, afterCommit string) ([]vcs.FileDiff, error) {
	var bbcDiffs []*CommitDiffStat
	next := fmt.Sprintf("%s/repositories/%s/diffstat/%s..%s?pagelen=%d", p.APIURL(instanceURL), repositoryID, afterCommit, beforeCommit, apiPageSize)
	for next != "" {
		var err error
		var diffs []*CommitDiffStat
		diffs, next, err = p.fetchPaginatedDiffFileList(ctx, oauthCtx, instanceURL, next)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcDiffs = append(bbcDiffs, diffs...)
	}

	var diffs []vcs.FileDiff
	for _, d := range bbcDiffs {
		diff := vcs.FileDiff{
			Path: d.New.Path,
		}
		switch d.Status {
		case "added":
			diff.Type = vcs.FileDiffTypeAdded
		case "modified":
			diff.Type = vcs.FileDiffTypeModified
		case "removed":
			diff.Type = vcs.FileDiffTypeRemoved
		// To be consistent with GitLab, we treat renamed as added.
		case "renamed":
			diff.Type = vcs.FileDiffTypeAdded
		default:
			// Skip because we don't care about file diff in other status
			continue
		}
		diffs = append(diffs, diff)
	}
	return diffs, nil
}

func (p *Provider) fetchPaginatedDiffFileList(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, url string) (diffs []*CommitDiffStat, next string, err error) {
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
	)
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

	var resp struct {
		Values []*CommitDiffStat `json:"values"`
		Next   string            `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, "", errors.Wrapf(err, "failed to unmarshal file diff data from Bitbucket Cloud instance %s", instanceURL)
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
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx *common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	var bbcRepos []*Repository
	params := url.Values{}
	params.Add("q", `permission="admin"`)
	params.Add("pagelen", strconv.Itoa(apiPageSize))
	next := fmt.Sprintf(`%s/user/permissions/repositories?%s`, p.APIURL(instanceURL), params.Encode())
	for next != "" {
		var err error
		var repos []*Repository
		repos, next, err = p.fetchPaginatedRepositoryList(ctx, oauthCtx, next)
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
				WebURL:   fmt.Sprintf("%s/%s", instanceURL, r.FullName),
			},
		)
	}
	return repos, nil
}

// fetchPaginatedRepositoryList fetches repositories in given page. It returns
// the paginated results along with a string indicating the URL of the next page
// (if exists).
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, oauthCtx *common.OauthContext, url string) (repos []*Repository, next string, err error) {
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
	)
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

// TreeEntry represents a Bitbucket Cloud API response for a repository tree
// entry.
type TreeEntry struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Commit Commit `json:"commit"`
}

// FetchRepositoryFileList fetches the all files from the given repository tree
// recursively.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#directory-listings
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	var bbcTreeEntries []*TreeEntry
	params := url.Values{}
	// NOTE: There is no way to ask the Bitbucket Cloud API to return all
	// subdirectories recursively, 10 levels down is just a good guess.
	params.Add("max_depth", "10")
	params.Add("q", `type="commit_file"`)
	params.Add("pagelen", strconv.Itoa(apiPageSize))
	next := fmt.Sprintf("%s/repositories/%s/src/%s/%s?%s", p.APIURL(instanceURL), repositoryID, url.PathEscape(ref), url.PathEscape(filePath), params.Encode())
	for next != "" {
		var err error
		var treeEntries []*TreeEntry
		treeEntries, next, err = p.fetchPaginatedRepositoryFileList(ctx, oauthCtx, next)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcTreeEntries = append(bbcTreeEntries, treeEntries...)
	}

	var treeNodes []*vcs.RepositoryTreeNode
	for _, n := range bbcTreeEntries {
		treeNodes = append(treeNodes,
			&vcs.RepositoryTreeNode{
				Path: n.Path,
				Type: n.Type,
			},
		)
	}
	return treeNodes, nil
}

// fetchPaginatedRepositoryFileList fetches files under a repository tree
// recursively in given page. It returns the paginated results along with a
// string indicating URL of the next page (if exists).
func (p *Provider) fetchPaginatedRepositoryFileList(ctx context.Context, oauthCtx *common.OauthContext, url string) (_ []*TreeEntry, next string, err error) {
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
	)
	if err != nil {
		return nil, "", errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, "", common.Errorf(common.NotFound, "failed to fetch repository file list from URL %s", url)
	} else if code >= 300 {
		return nil, "",
			errors.Errorf("failed to fetch repository file list from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	var resp struct {
		Values []*TreeEntry `json:"values"`
		Next   string       `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, "", errors.Wrap(err, "unmarshal body")
	}
	return resp.Values, resp.Next, nil
}

// CreateFile creates a file at given path in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#api-repositories-workspace-repo-slug-src-post
func (p *Provider) CreateFile(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormField(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}
	_, err = part.Write([]byte(fileCommitCreate.Content))
	if err != nil {
		return errors.Wrap(err, "failed to write file to form")
	}
	_ = w.WriteField("message", fileCommitCreate.CommitMessage)
	_ = w.WriteField("parents", fileCommitCreate.LastCommitID)
	_ = w.WriteField("branch", fileCommitCreate.Branch)
	_ = w.Close()

	url := fmt.Sprintf("%s/repositories/%s/src", p.APIURL(instanceURL), repositoryID)
	code, _, resp, err := oauth.PostWithHeader(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		&body,
		map[string]string{
			"Content-Type": w.FormDataContentType(),
		},
	)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
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
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#api-repositories-workspace-repo-slug-src-post
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.CreateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate)
}

// ReadFileMeta reads the metadata of the given file in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#file-meta-data
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, filePath string, refInfo vcs.RefInfo) (*vcs.FileMeta, error) {
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s?format=meta", p.APIURL(instanceURL), repositoryID, url.PathEscape(refInfo.RefName), url.PathEscape(filePath))
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
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

	var treeEntry TreeEntry
	if err = json.Unmarshal([]byte(body), &treeEntry); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	if treeEntry.Type != "commit_file" {
		return nil, errors.Errorf("%q is not a file", filePath)
	}

	return &vcs.FileMeta{
		Name:         path.Base(treeEntry.Path),
		Path:         treeEntry.Path,
		Size:         treeEntry.Size,
		LastCommitID: treeEntry.Commit.Hash,
	}, nil
}

// ReadFileContent reads the content of the given file in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#raw-file-contents
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, filePath string, refInfo vcs.RefInfo) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s", p.APIURL(instanceURL), repositoryID, url.PathEscape(refInfo.RefName), url.PathEscape(filePath))
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
	)
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
func (p *Provider) GetBranch(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/repositories/%s/refs/branches/%s", p.APIURL(instanceURL), repositoryID, branchName)
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
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

	var branch Branch
	if err := json.Unmarshal([]byte(body), &branch); err != nil {
		return nil, errors.Wrap(err, "unmarshal body")
	}

	return &vcs.BranchInfo{
		Name:         branch.Name,
		LastCommitID: branch.Target.Hash,
	}, nil
}

type branchCreateTarget struct {
	Hash string `json:"hash"`
}

type branchCreate struct {
	Name   string             `json:"name"`
	Target branchCreateTarget `json:"target"`
}

// CreateBranch creates the branch in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-refs/#api-repositories-workspace-repo-slug-refs-branches-post
func (p *Provider) CreateBranch(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID string, branch *vcs.BranchInfo) error {
	body, err := json.Marshal(
		branchCreate{
			Name:   branch.Name,
			Target: branchCreateTarget{Hash: branch.LastCommitID},
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal branch create")
	}

	url := fmt.Sprintf("%s/repositories/%s/refs/branches", p.APIURL(instanceURL), repositoryID)
	code, _, resp, err := oauth.Post(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(body),
	)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
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

// ListPullRequestFile lists the changed files in the pull request.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/#api-repositories-workspace-repo-slug-pullrequests-pull-request-id-diffstat-get
func (p *Provider) ListPullRequestFile(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	var bbcDiffs []*CommitDiffStat
	next := fmt.Sprintf("%s/repositories/%s/pullrequests/%s/diffstat?pagelen=%d", p.APIURL(instanceURL), url.PathEscape(repositoryID), pullRequestID, apiPageSize)
	for next != "" {
		var err error
		var diffs []*CommitDiffStat
		diffs, next, err = p.fetchPaginatedDiffFileList(ctx, oauthCtx, instanceURL, next)
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

type pullRequestCreateBranch struct {
	Name string `json:"name"`
}
type pullRequestCreateTarget struct {
	Branch pullRequestCreateBranch `json:"branch"`
}

type pullRequestCreate struct {
	Title             string                  `json:"title"`
	Description       string                  `json:"description"`
	CloseSourceBranch bool                    `json:"close_source_branch"`
	Source            pullRequestCreateTarget `json:"source"`
	Destination       pullRequestCreateTarget `json:"destination"`
}

// CreatePullRequest creates the pull request in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/#api-repositories-workspace-repo-slug-pullrequests-post
func (p *Provider) CreatePullRequest(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID string, create *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
	payload, err := json.Marshal(
		pullRequestCreate{
			Title:             create.Title,
			Description:       create.Body,
			CloseSourceBranch: create.RemoveHeadAfterMerged,
			Source: pullRequestCreateTarget{
				Branch: pullRequestCreateBranch{Name: create.Head},
			},
			Destination: pullRequestCreateTarget{
				Branch: pullRequestCreateBranch{Name: create.Base},
			},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "marshal pull request create")
	}

	url := fmt.Sprintf("%s/repositories/%s/pullrequests", p.APIURL(instanceURL), repositoryID)
	code, _, body, err := oauth.Post(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "POST %s", url)
	}

	if code == http.StatusNotFound {
		return nil, common.Errorf(common.NotFound, "failed to create pull request from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to create pull request from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var resp struct {
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, err
	}

	return &vcs.PullRequest{
		URL: resp.Links.HTML.Href,
	}, nil
}

type pipelineVariable struct {
	// Type should always be "pipeline_variable"
	Type    string `json:"type,omitempty"`
	UUID    string `json:"uuid,omitempty"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Secured bool   `json:"secured,omitempty"`
}

type listPipelineVariable struct {
	Values []*pipelineVariable `json:"values"`
}

// UpsertEnvironmentVariable creates or updates the pipeline variable in the repository.
func (p *Provider) UpsertEnvironmentVariable(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, key, value string) error {
	existed, err := p.getPipelineVariable(ctx, oauthCtx, instanceURL, repositoryID, key)
	if err != nil {
		return err
	}

	if existed == nil {
		return p.createPipelineVariable(ctx, oauthCtx, instanceURL, repositoryID, key, value)
	}
	return p.updatePipelineVariable(ctx, oauthCtx, instanceURL, repositoryID, existed.UUID, value)
}

// getPipelineVariable gets the pipeline variable in the repository.
//
// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-config-variables-get.
func (p *Provider) getPipelineVariable(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, key string) (*pipelineVariable, error) {
	url := fmt.Sprintf("%s/repositories/%s/pipelines_config/variables", p.APIURL(instanceURL), repositoryID)
	code, _, body, err := oauth.Get(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", url)
	}

	if code == http.StatusNotFound {
		return nil, nil
	} else if code >= 300 {
		return nil,
			errors.Errorf("failed to found variable from URL %s, status code: %d, body: %s",
				url,
				code,
				body,
			)
	}

	variables := new(listPipelineVariable)
	if err := json.Unmarshal([]byte(body), variables); err != nil {
		return nil, err
	}

	for _, variable := range variables.Values {
		if variable.Key == key {
			return variable, nil
		}
	}

	return nil, nil
}

// createPipelineVariable creates the pipeline variable in the repository.
//
// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-config-variables-post.
func (p *Provider) createPipelineVariable(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, key, value string) error {
	url := fmt.Sprintf("%s/repositories/%s/pipelines_config/variables", p.APIURL(instanceURL), repositoryID)
	body, err := json.Marshal(
		pipelineVariable{
			Type:    "pipeline_variable",
			Key:     key,
			Value:   value,
			Secured: true,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal variable create")
	}
	code, _, resp, err := oauth.Post(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(body),
	)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}
	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to create pipeline variable through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to create pipeline variable through URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
	return nil
}

// updatePipelineVariable updates the environment variable in the repository.
//
// https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pipelines/#api-repositories-workspace-repo-slug-pipelines-config-variables-variable-uuid-put.
func (p *Provider) updatePipelineVariable(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, variableID, value string) error {
	url := fmt.Sprintf("%s/repositories/%s/pipelines_config/variables/%s", p.APIURL(instanceURL), repositoryID, variableID)
	body, err := json.Marshal(
		pipelineVariable{
			Value: value,
		},
	)
	if err != nil {
		return errors.Wrap(err, "marshal variable update")
	}
	code, _, resp, err := oauth.Put(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(body),
	)
	if err != nil {
		return errors.Wrapf(err, "PUT %s", url)
	}
	if code == http.StatusNotFound {
		return common.Errorf(common.NotFound, "failed to update pipeline variable through URL %s", url)
	} else if code >= 300 {
		return errors.Errorf("failed to update pipeline variable through URL %s, status code: %d, body: %s",
			url,
			code,
			resp,
		)
	}
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
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/hooks", p.APIURL(instanceURL), repositoryID)
	code, _, body, err := oauth.Post(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(payload),
	)
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

// PatchWebhook patches the webhook in the repository with given payload.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-uid-put
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	url := fmt.Sprintf("%s/repositories/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
	code, _, body, err := oauth.Put(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
		bytes.NewReader(payload),
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
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-uid-delete
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx *common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/repositories/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
	code, _, body, err := oauth.Delete(
		ctx,
		p.client,
		url,
		oauthCtx.AccessToken,
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
