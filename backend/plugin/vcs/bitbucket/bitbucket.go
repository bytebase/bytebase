// Package bitbucket is the plugin for Bitbucket Cloud.
package bitbucket

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
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
func (*Provider) APIURL(string) string {
	return "https://api.bitbucket.org/2.0"
}

// oauthResponse is a Bitbucket Cloud OAuth response.
type oauthResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// toVCSOAuthToken converts the response to *vcs.OAuthToken.
func (o oauthResponse) toVCSOAuthToken() *vcs.OAuthToken {
	oauthToken := &vcs.OAuthToken{
		AccessToken:  o.AccessToken,
		RefreshToken: o.RefreshToken,
		ExpiresIn:    o.ExpiresIn,
		CreatedAt:    time.Now().Unix(),
		ExpiresTs:    time.Now().Add(time.Duration(o.ExpiresIn) * time.Second).Unix(),
	}
	return oauthToken
}

// ExchangeOAuthToken exchanges OAuth content with the provided authorization code.
func (p *Provider) ExchangeOAuthToken(ctx context.Context, instanceURL string, oauthExchange *common.OAuthExchange) (*vcs.OAuthToken, error) {
	form := &url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", oauthExchange.Code)
	url := fmt.Sprintf("%s/site/oauth2/access_token", instanceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrapf(err, "construct POST %s", url)
	}

	digested := base64.StdEncoding.EncodeToString([]byte(oauthExchange.ClientID + ":" + oauthExchange.ClientSecret))
	req.Header.Set("Authorization", digested)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

func (p *Provider) TryLogin(context.Context, common.OauthContext, string) (*vcs.UserInfo, error) {
	// TODO: We will remove VCS login as part of https://linear.app/bytebase/issue/BYT-2615,
	// so leaving it as unimplemented here.
	return nil, errors.New("not implemented")
}

// User represents a Bitbucket Cloud API response for a user.
type User struct {
	DisplayName string `json:"display_name"`
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
func (p *Provider) FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, commitID string) (*vcs.Commit, error) {
	url := fmt.Sprintf("%s/repositories/%s/commit/%s", p.APIURL(instanceURL), url.PathEscape(repositoryID), commitID)
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
		ID:         commit.Hash,
		AuthorName: commit.Author.User.DisplayName,
		CreatedTs:  commit.Date.Unix(),
	}, nil
}

// CommitFile represents a Bitbucket Cloud API response for a file at a commit.
type CommitFile struct {
	Path  string `json:"path"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
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
func (p *Provider) GetDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, beforeCommit, afterCommit string) ([]vcs.FileDiff, error) {
	var bbcDiffs []*CommitDiffStat
	page := 1
	for {
		url := fmt.Sprintf("%s/repositories/%s/diffstat/%s..%s?page=%d&pagelen=%d", p.APIURL(instanceURL), url.PathEscape(repositoryID), afterCommit, beforeCommit, page, apiPageSize)
		diffs, hasNextPage, err := p.fetchPaginatedDiffFileList(ctx, oauthCtx, instanceURL, url)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcDiffs = append(bbcDiffs, diffs...)

		if !hasNextPage {
			break
		}
		page++
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
		default:
			// Skip because we don't care about file diff in other status
			continue
		}
		diffs = append(diffs, diff)
	}
	return diffs, nil
}

func (p *Provider) fetchPaginatedDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, url string) (diffs []*CommitDiffStat, hasNextPage bool, err error) {
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
		return nil, false, common.Errorf(common.NotFound, "failed to get file diff list from URL %s", url)
	} else if code >= 300 {
		return nil, false, errors.Errorf("failed to get file diff list from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var resp struct {
		Values []*CommitDiffStat `json:"values"`
		Next   bool              `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, false, errors.Wrapf(err, "failed to unmarshal file diff data from Bitbucket Cloud instance %s", instanceURL)
	}
	return resp.Values, resp.Next, nil
}

// Repository represents a Bitbucket Cloud API response for a repository.
type Repository struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
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
func (p *Provider) FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*vcs.Repository, error) {
	var bbcRepos []*Repository
	page := 1
	for {
		repos, hasNextPage, err := p.fetchPaginatedRepositoryList(ctx, oauthCtx, instanceURL, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcRepos = append(bbcRepos, repos...)

		if !hasNextPage {
			break
		}
		page++
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

// fetchPaginatedRepositoryList fetches repositories where the authenticated
// user has admin access to in given page. It returns the paginated results
// along with a boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, page int) (repos []*Repository, hasNextPage bool, err error) {
	url := fmt.Sprintf(`%s/user/permissions/repositories?q=%s&page=%d&pagelen=%d`, p.APIURL(instanceURL), url.PathEscape(`permission="admin"`), page, apiPageSize)
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

	var resp struct {
		Values []*RepositoryPermission `json:"values"`
		Next   bool                    `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, false, errors.Wrap(err, "unmarshal")
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
func (p *Provider) FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*vcs.RepositoryTreeNode, error) {
	var bbcTreeEntries []*TreeEntry
	page := 1
	for {
		treeEntries, hasNextPage, err := p.fetchPaginatedRepositoryFileList(ctx, oauthCtx, instanceURL, repositoryID, ref, filePath, page)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcTreeEntries = append(bbcTreeEntries, treeEntries...)

		if !hasNextPage {
			break
		}
		page++
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
// boolean indicating whether the next page exists.
func (p *Provider) fetchPaginatedRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string, page int) (_ []*TreeEntry, hasNextPage bool, err error) {
	// NOTE: There is no way to ask the Bitbucket Cloud API to return all
	// subdirectories recursively, 10 levels down is just a good guess.
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s?max_depth=10&q=%s&page=%d&pagelen=%d", p.APIURL(instanceURL), repositoryID, ref, url.PathEscape(filePath), url.QueryEscape(`type="commit_file"`), page, apiPageSize)
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

	var resp struct {
		Values []*TreeEntry `json:"values"`
		Next   bool         `json:"next"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, false, errors.Wrap(err, "unmarshal body")
	}
	return resp.Values, resp.Next, nil
}

// CreateFile creates a file at given path in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#api-repositories-workspace-repo-slug-src-post
func (p *Provider) CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("filename", filePath)
	if err != nil {
		return errors.Wrap(err, "failed to create form file")
	}
	_, err = part.Write([]byte(fileCommitCreate.Content))
	if err != nil {
		return errors.Wrap(err, "failed to write file to form")
	}
	_ = w.Close()

	urlParams := &url.Values{}
	urlParams.Set("message", url.QueryEscape(fileCommitCreate.CommitMessage))
	urlParams.Set("parents", fileCommitCreate.LastCommitID)
	urlParams.Set("branch", fileCommitCreate.Branch)

	url := fmt.Sprintf("%s/repositories/%s/src?%s", p.APIURL(instanceURL), repositoryID, urlParams.Encode())
	code, _, resp, err := oauth.Post(
		ctx,
		p.client,
		url,
		&oauthCtx.AccessToken,
		&body,
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
func (p *Provider) OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommitCreate vcs.FileCommitCreate) error {
	return p.CreateFile(ctx, oauthCtx, instanceURL, repositoryID, filePath, fileCommitCreate)
}

// ReadFileMeta reads the metadata of the given file in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-source/#file-meta-data
func (p *Provider) ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*vcs.FileMeta, error) {
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s?format=meta", p.APIURL(instanceURL), repositoryID, ref, url.PathEscape(filePath))
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
func (p *Provider) ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/src/%s/%s", p.APIURL(instanceURL), repositoryID, ref, url.PathEscape(filePath))
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

// Branch is the API message for Bitbucket Cloud branch.
type Branch struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

// GetBranch gets the given branch in the repository.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-refs/#api-repositories-workspace-repo-slug-refs-branches-name-get
func (p *Provider) GetBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	url := fmt.Sprintf("%s/repositories/%s/refs/branches/%s", p.APIURL(instanceURL), repositoryID, branchName)
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
func (p *Provider) CreateBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, branch *vcs.BranchInfo) error {
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
func (p *Provider) ListPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	var bbcDiffs []*CommitDiffStat
	page := 1
	for {
		url := fmt.Sprintf("%s/repositories/%s/pullrequests/%s/diffstat?page=%d&pagelen=%d", p.APIURL(instanceURL), url.PathEscape(repositoryID), pullRequestID, page, apiPageSize)
		diffs, hasNextPage, err := p.fetchPaginatedDiffFileList(ctx, oauthCtx, instanceURL, url)
		if err != nil {
			return nil, errors.Wrap(err, "fetch paginated list")
		}
		bbcDiffs = append(bbcDiffs, diffs...)

		if !hasNextPage {
			break
		}
		page++
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
func (p *Provider) CreatePullRequest(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, create *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
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

// UpsertEnvironmentVariable creates or updates the environment variable in the repository.
//
// WARNING: This is not supported in Bitbucket Cloud.
func (p *Provider) UpsertEnvironmentVariable(context.Context, common.OauthContext, string, string, string, string) error {
	return errors.New("not supported")
}

// CreateWebhook creates a webhook in the repository with given payload.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-post
func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	url := fmt.Sprintf("%s/repositories/%s/hooks", p.APIURL(instanceURL), repositoryID)
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

	// Bitbucket Cloud returns 201 HTTP status codes upon successful webhook creation,
	// see https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-post-responses for details.
	if code != http.StatusCreated {
		return "", errors.Errorf("failed to create webhook through URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var resp struct {
		UUID string `json:"uuid"`
	}
	if err = json.Unmarshal([]byte(body), &resp); err != nil {
		return "", errors.Wrap(err, "unmarshal body")
	}
	return resp.UUID, nil
}

// PatchWebhook patches the webhook in the repository with given payload.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-uid-put
func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	url := fmt.Sprintf("%s/repositories/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
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
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-repositories/#api-repositories-workspace-repo-slug-hooks-uid-delete
func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	url := fmt.Sprintf("%s/repositories/%s/hooks/%s", p.APIURL(instanceURL), repositoryID, webhookID)
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

// oauthContext is the request context for refreshing OAuth token.
type oauthContext struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	GrantType    string `json:"grant_type"`
}

type refreshOAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	// token_type, scope are not used.
}

func tokenRefresher(instanceURL string, oauthCtx oauthContext, refresher common.TokenRefresher) oauth.TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		form := &url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", oauthCtx.RefreshToken)
		url := fmt.Sprintf("%s/site/oauth2/access_token", instanceURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
		if err != nil {
			return errors.Wrapf(err, "construct POST %s", url)
		}
		digested := base64.StdEncoding.EncodeToString([]byte(oauthCtx.ClientID + ":" + oauthCtx.ClientSecret))
		req.Header.Set("Authorization", digested)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "POST %s", url)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "read body of POST %s", url)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("non-200 POST %s status code %d with body %q", url, resp.StatusCode, body)
		}

		var r refreshOAuthResponse
		if err = json.Unmarshal(body, &r); err != nil {
			return errors.Wrapf(err, "unmarshal body from POST %s", url)
		}

		// Update the old token to new value for retries.
		*oldToken = r.AccessToken

		expireAt := time.Now().Add(time.Duration(r.ExpiresIn) * time.Second).Unix()
		return refresher(r.AccessToken, r.RefreshToken, expireAt)
	}
}
