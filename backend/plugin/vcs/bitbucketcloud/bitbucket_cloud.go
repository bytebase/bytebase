package bitbucketcloud

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
	vcs.Register(vcs.BitbucketCloud, newProvider)
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
func (p *Provider) APIURL(string) string {
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

func (p *Provider) TryLogin(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) (*vcs.UserInfo, error) {
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
	Path string `json:"path"`
}

// CommitDiffStat represents a Bitbucket Cloud API response for commit diff stat.
type CommitDiffStat struct {
	// The status of the diff stat object, possible values are "added", "removed",
	// "modified", "renamed".
	Status string     `json:"status"`
	New    CommitFile `json:"new"`
}

// CommitsDiff represents a Bitbucket Cloud API response for comparing two
// commits.
type CommitsDiff struct {
	Values []CommitDiffStat `json:"values"`
}

// GetDiffFileList gets the diff files list between two commits.
//
// Docs: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-commits/#api-repositories-workspace-repo-slug-diffstat-spec-get
func (p *Provider) GetDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, beforeCommit, afterCommit string) ([]vcs.FileDiff, error) {
	url := fmt.Sprintf("%s/repositories/%s/diffstat/%s..%s", p.APIURL(instanceURL), url.PathEscape(repositoryID), afterCommit, beforeCommit)
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
		return nil, common.Errorf(common.NotFound, "failed to get file diff list from URL %s", url)
	} else if code >= 300 {
		return nil, errors.Errorf("failed to get file diff list from URL %s, status code: %d, body: %s",
			url,
			code,
			body,
		)
	}

	var commitsDiff CommitsDiff
	if err := json.Unmarshal([]byte(body), &commitsDiff); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file diff data from Bitbucket Cloud instance %s", instanceURL)
	}

	var fileDiffs []vcs.FileDiff
	for _, v := range commitsDiff.Values {
		diff := vcs.FileDiff{
			Path: v.New.Path,
		}
		switch v.Status {
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
		fileDiffs = append(fileDiffs, diff)
	}
	return fileDiffs, nil
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

func (p *Provider) GetBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, branchName string) (*vcs.BranchInfo, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) CreateBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, branch *vcs.BranchInfo) error {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) ListPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*vcs.PullRequestFile, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) CreatePullRequest(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, pullRequestCreate *vcs.PullRequestCreate) (*vcs.PullRequest, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) UpsertEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, key, value string) error {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error {
	// TODO implement me
	panic("implement me")
}

func (p *Provider) DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error {
	// TODO implement me
	panic("implement me")
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
