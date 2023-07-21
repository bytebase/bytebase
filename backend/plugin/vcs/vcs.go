package vcs

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// Type is the type of a VCS.
// nolint
type Type string

const (
	// GitLab is the VCS type for GitLab (both GitLab.com and self-hosted).
	GitLab Type = "GITLAB"
	// GitHub is the VCS type for GitHub (both GitHub.com and GitHun Enterprise).
	GitHub Type = "GITHUB"
	// Bitbucket is the VCS type for Bitbucket Cloud (bitbucket.org).
	Bitbucket Type = "BITBUCKET"
	// AzureDevOps is the VCS type for Azure DevOps.
	AzureDevOps Type = "AZURE_DEVOPS"

	// SQLReviewAPISecretName is the api secret name used in GitHub action or GitLab CI workflow.
	SQLReviewAPISecretName = "SQL_REVIEW_API_SECRET"

	// BytebaseAuthorName is the author name of bytebase.
	BytebaseAuthorName = "Bytebase"
	// BytebaseAuthorEmail is the author email of bytebase.
	BytebaseAuthorEmail = "support@bytebase.com"
)

// OAuthToken is the API message for OAuthToken.
type OAuthToken struct {
	AccessToken  string `json:"access_token" `
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
	// ExpiresTs is a derivative from ExpiresIn and CreatedAt.
	// ExpiresTs = ExpiresIn == 0 ? 0 : CreatedAt + ExpiresIn
	ExpiresTs int64 `json:"expires_ts"`
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention

// Commit records the commit data.
type Commit struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Message      string   `json:"message"`
	CreatedTs    int64    `json:"createdTs"`
	URL          string   `json:"url"`
	AuthorName   string   `json:"authorName"`
	AuthorEmail  string   `json:"authorEmail"`
	AddedList    []string `json:"addedList"`
	ModifiedList []string `json:"modifiedList"`
}

// FileCommit is the API message for a VCS file commit.
type FileCommit struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	CreatedTs   int64  `json:"createdTs"`
	URL         string `json:"url"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	Added       string `json:"added"`
}

// FileCommitCreate is the payload for committing a new file.
type FileCommitCreate struct {
	Branch        string
	Content       string
	CommitMessage string
	LastCommitID  string
	SHA           string
	AuthorName    string
	AuthorEmail   string
}

// FileMeta records the file metadata.
type FileMeta struct {
	Name         string
	Path         string
	Size         int64
	LastCommitID string
	SHA          string
}

// FileDiffType is the type of file diff.
type FileDiffType int

const (
	// FileDiffTypeUnknown means the file is an unknown diff type.
	FileDiffTypeUnknown FileDiffType = iota
	// FileDiffTypeAdded means the file is newly added.
	FileDiffTypeAdded
	// FileDiffTypeModified means the file is modified.
	FileDiffTypeModified
	// FileDiffTypeRemoved means the file is removed.
	FileDiffTypeRemoved
)

// FileDiff contains file diffs between two commits.
// It's obtained by comparing the base and head commits of a PR/MR so that we know the real changes.
type FileDiff struct {
	Path string
	Type FileDiffType
}

// RepositoryTreeNode records the node(file/folder) of a repository tree from `git ls-tree`.
type RepositoryTreeNode struct {
	Path string
	Type string
}

// PushEvent is the API message for a VCS push event.
type PushEvent struct {
	VCSType            Type     `json:"vcsType"`
	BaseDirectory      string   `json:"baseDir"`
	Ref                string   `json:"ref"`
	Before             string   `json:"before"`
	After              string   `json:"after"`
	RepositoryID       string   `json:"repositoryId"`
	RepositoryURL      string   `json:"repositoryUrl"`
	RepositoryFullPath string   `json:"repositoryFullPath"`
	AuthorName         string   `json:"authorName"`
	CommitList         []Commit `json:"commits"`
	// Legacy field, kept for parsing existing data
	FileCommit FileCommit `json:"fileCommit"`
}

// State is the state of a VCS user account.
type State string

const (
	// StateActive is the active state for VCS user state.
	StateActive State = "active"
	// StateArchived is the archived state for VCS user state.
	StateArchived State = "archived"
)

// UserInfo is the API message for user info.
type UserInfo struct {
	// NOTICE: we use public email here because user's primary email can only be accessed by the admin
	PublicEmail string `json:"public_email"`
	Name        string `json:"name"`
	State       State  `json:"state"`
}

// Repository is the API message for repository info.
type Repository struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"fullPath"`
	WebURL   string `json:"webUrl"`
}

// PullRequestFile is the API message for file in the pull request.
type PullRequestFile struct {
	Path         string
	LastCommitID string
	IsDeleted    bool
}

// BranchInfo is the API message for repository branch.
type BranchInfo struct {
	Name         string
	LastCommitID string
}

// PullRequestCreate is the API message to create pull request in repository.
type PullRequestCreate struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	// Flag indicating if a merge request should remove the source branch after merging.
	// Only support GitLab.
	RemoveHeadAfterMerged bool `json:"-"`
}

// PullRequest is the API message for pull request in repository.
type PullRequest struct {
	URL string `json:"url"`
}

// Provider is the interface for VCS provider.
type Provider interface {
	// Returns the API URL for a given VCS instance URL
	APIURL(instanceURL string) string

	// Exchange oauth token with the provided code
	//
	// instanceURL: VCS instance URL
	// oauthExchange: api message for exchanging oauth token
	ExchangeOAuthToken(ctx context.Context, instanceURL string, oauthExchange *common.OAuthExchange) (*OAuthToken, error)

	// Try to use this provider as an auth provider and fetch the user info from the OAuth context
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	TryLogin(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) (*UserInfo, error)
	// Fetch the commit data by id
	//
	// oauthCtx: OAuth context to fetch commit
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// commitID: the commit ID
	FetchCommitByID(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, commitID string) (*Commit, error)
	// Get the diff files list between two commits
	//
	// oauthCtx: OAuth context to fetch commit
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// beforeCommit: the previous commit
	// afterCommit: the current commit
	GetDiffFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, beforeCommit, afterCommit string) ([]FileDiff, error)

	// Fetch all repository within a given user's scope
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	FetchAllRepositoryList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) ([]*Repository, error)

	// Fetch the repository file list
	//
	// oauthCtx: OAuth context to read the repository tree
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// ref: the unique name of a repository tree, could be a branch name in GitLab or a tree sha in GitHub
	// filePath: the path inside repository, used to get content of subdirectories
	FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, ref, filePath string) ([]*RepositoryTreeNode, error)

	// Commits a new file
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// filePath: file path to be written
	// fileCommit: the new file commit info
	CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit FileCommitCreate) error
	// Overwrites an existing file
	//
	// Similar to CreateFile except it overwrites an existing file. The fileCommit should includes the "LastCommitID" field which is used to detect conflicting writes.
	OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath string, fileCommit FileCommitCreate) error
	// Reads the file metadata
	//
	// oauthCtx: OAuth context to fetch the file metadata
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// filePath: file path to be read
	// ref: the specific file version to be read, could be a name of branch, tag or commit
	ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (*FileMeta, error)
	// Reads the file content
	//
	// oauthCtx: OAuth context to read the file content
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// filePath: file path to be read
	// ref: the specific file version to be read, could be a name of branch, tag or commit
	ReadFileContent(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, filePath, ref string) (string, error)
	// GetBranch gets the given branch in the repository.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// branchName: the target branch name
	GetBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, branchName string) (*BranchInfo, error)
	// CreateBranch creates the branch in the repository.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// branch: the new branch info
	CreateBranch(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, branch *BranchInfo) error
	// CreatePullRequest creates the pull request in the repository.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// pullRequestID: the pull request id
	ListPullRequestFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, pullRequestID string) ([]*PullRequestFile, error)
	// pullRequestCreate: the new pull request info
	CreatePullRequest(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, pullRequestCreate *PullRequestCreate) (*PullRequest, error)
	// UpsertEnvironmentVariable creates or updates the environment variable in the repository.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// key: the environment variable name
	// value: the environment variable value
	UpsertEnvironmentVariable(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, key, value string) error
	// Creates a webhook. Returns the created webhook ID on success.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// payload: the webhook payload
	CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID string, payload []byte) (string, error)
	// Patches a webhook.
	//
	// The payload stores the patched field(s).
	PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string, payload []byte) error
	// Deletes a webhook.
	DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL, repositoryID, webhookID string) error
}

var (
	providerMu sync.RWMutex
	providers  = make(map[Type]providerFunc)
)

// ProviderConfig is the provider configuration.
type ProviderConfig struct {
	Client *http.Client
}

type providerFunc func(ProviderConfig) Provider

// Register makes a vcs provider available by the provided type.
// If Register is called twice with the same name or if provider is nil,
// it panics.
func Register(vcsType Type, f providerFunc) {
	providerMu.Lock()
	defer providerMu.Unlock()
	if f == nil {
		panic("vcs: Register provider is nil")
	}
	if _, dup := providers[vcsType]; dup {
		panic("vcs: Register called twice for provider " + vcsType)
	}
	providers[vcsType] = f
}

// Get returns a vcs provider specified by its vcs type.
func Get(vcsType Type, providerConfig ProviderConfig) Provider {
	providerMu.RLock()
	f, ok := providers[vcsType]
	providerMu.RUnlock()
	if !ok {
		panic("vcs: unknown provider " + vcsType)
	}

	return f(providerConfig)
}

// IsAsterisksInTemplateValid checks if the pathTemplate is valid about asterisk.
// Fow now, our rules are:
// 1. Support two asterisks at most.
// 2. Both ends of consecutive asterisks just can be '/'.
// 3. Consecutive asterisks cannot be placed at the beginning or end.
func IsAsterisksInTemplateValid(pathTemplate string) error {
	const maxAsteriskNum = 2
	if err := isMaxConsecutiveAsteriskValid(pathTemplate, maxAsteriskNum); err != nil {
		return err
	}
	if err := isSingleAsteriskInTemplateValid(pathTemplate); err != nil {
		return err
	}
	return isDoubleAsteriskInTemplateValid(pathTemplate)
}

// isSingleAsteriskInTemplateValid checks whether the single in file path template is valid.
func isSingleAsteriskInTemplateValid(pathTemplate string) error {
	return isMultipleTimesAsteriskInTemplateValid(pathTemplate, 1)
}

// isDoubleAsteriskInTemplateValid checks whether the consecutive double asterisks in file path template is valid.
func isDoubleAsteriskInTemplateValid(pathTemplate string) error {
	return isMultipleTimesAsteriskInTemplateValid(pathTemplate, 2)
}

// isMaxConsecutiveAsteriskValid returns true if the pathTemplate contains `n` consecutive asterisk at most.
func isMaxConsecutiveAsteriskValid(pathTemplate string, n int) error {
	re := regexp.MustCompile(strings.Repeat(`\*`, n+1))
	if re.MatchString(pathTemplate) {
		return errors.Errorf("path template %s contains more than %d asterisk", pathTemplate, n)
	}
	return nil
}

// isMultipleTimesAsteriskInTemplateValid checks whether the consecutive asterisks in file path template is valid.
// The rules are（）:
// 1. Consecutive asterisks cannot be placed at the beginning or end.
// 2. Both ends of consecutive asterisks just can be * or /.
// Take asteriskTimes = 2 as an example:
// "**/test" and "test/**" will break the rule1.
// "abc**" and "?**/" will break the rule2.
func isMultipleTimesAsteriskInTemplateValid(pathTemplate string, asteriskTimes int) error {
	base := strings.Repeat(`\*`, asteriskTimes)
	rs := []struct {
		regex  string
		errmsg string
	}{
		{
			regex:  fmt.Sprintf(`([^\/\*]+%s)`, base),
			errmsg: `In path template, * can only be preceded by another / or *`,
		},
		{
			regex:  fmt.Sprintf(`(%s[^\/\*]+)`, base),
			errmsg: `In path template, only / or * can be followed by *`,
		},
		{
			regex:  fmt.Sprintf(`(^(%s))`, base),
			errmsg: fmt.Sprintf(`path template %s contains consecutive asterisks at the beginning`, pathTemplate),
		},
		{
			regex:  fmt.Sprintf(`((%s)$)`, base),
			errmsg: fmt.Sprintf(`path template %s contains consecutive asterisks at the end`, pathTemplate),
		},
	}
	for _, rs := range rs {
		re := regexp.MustCompile(rs.regex)
		if re.MatchString(pathTemplate) {
			return errors.New(rs.errmsg)
		}
	}
	return nil
}
