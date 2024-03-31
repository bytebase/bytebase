package vcs

import (
	"context"
	"sync"
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

// RefType is the type of a ref.
type RefType string

const (
	// RefTypeBranch is the branch ref type.
	RefTypeBranch RefType = "branch"
	// RefTypeTag is the tag ref type.
	RefTypeTag RefType = "tag"
	// RefTypeCommit is the commit ref type.
	RefTypeCommit RefType = "commit"
)

// RefInfo is the API message for a VCS ref.
type RefInfo struct {
	RefType RefType
	RefName string
}

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
	Ref                string   `json:"ref"`
	Before             string   `json:"before"`
	After              string   `json:"after"`
	RepositoryID       string   `json:"repositoryId"`
	RepositoryURL      string   `json:"repositoryUrl"`
	RepositoryFullPath string   `json:"repositoryFullPath"`
	AuthorName         string   `json:"authorName"`
	CommitList         []Commit `json:"commits"`
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

	// Fetch all repository within a given user's scope
	FetchAllRepositoryList(ctx context.Context) ([]*Repository, error)

	// Reads the file content
	ReadFileContent(ctx context.Context, repositoryID, filePath string, refInfo RefInfo) (string, error)

	// GetBranch gets the given branch in the repository.
	GetBranch(ctx context.Context, repositoryID, branchName string) (*BranchInfo, error)

	// CreatePullRequest creates the pull request in the repository.
	ListPullRequestFile(ctx context.Context, repositoryID, pullRequestID string) ([]*PullRequestFile, error)

	// Creates a webhook. Returns the created webhook ID on success.
	CreateWebhook(ctx context.Context, repositoryID string, payload []byte) (string, error)

	// Deletes a webhook.
	DeleteWebhook(ctx context.Context, repositoryID, webhookID string) error
}

var (
	providerMu sync.RWMutex
	providers  = make(map[Type]providerFunc)
)

// ProviderConfig is the provider configuration.
type ProviderConfig struct {
	InstanceURL string
	AuthToken   string
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
