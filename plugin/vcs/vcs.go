package vcs

import (
	"context"
	"sync"

	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

// Type is the type of a VCS.
type Type string

const (
	// GitLabSelfHost is the VCS type for gitlab self host.
	GitLabSelfHost Type = "GITLAB_SELF_HOST"
)

func (e Type) String() string {
	switch e {
	case GitLabSelfHost:
		return "GITLAB_SELF_HOST"
	}
	return "UNKNOWN"
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention

// FileCommit is the API message for a VCS file commit.
type FileCommit struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	CreatedTs  int64  `json:"createdTs"`
	URL        string `json:"url"`
	AuthorName string `json:"authorName"`
	Added      string `json:"added"`
}

// FileCommitCreate is the payload for committing a new file.
type FileCommitCreate struct {
	Branch        string
	Content       string
	CommitMessage string
	LastCommitID  string
}

// FileMeta records the file metadata.
type FileMeta struct {
	LastCommitID string
}

// RepositoryTreeNode records the node(file/folder) of a repository tree from `git ls-tree`.
type RepositoryTreeNode struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// PushEvent is the API message for a VCS push event.
type PushEvent struct {
	VCSType            Type       `json:"vcsType"`
	BaseDirectory      string     `json:"baseDir"`
	Ref                string     `json:"ref"`
	RepositoryID       string     `json:"repositoryId"`
	RepositoryURL      string     `json:"repositoryUrl"`
	RepositoryFullPath string     `json:"repositoryFullPath"`
	AuthorName         string     `json:"authorName"`
	FileCommit         FileCommit `json:"fileCommit"`
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

// RepositoryMember is the API message for  repository member info.
type RepositoryMember struct {
	Email        string             `json:"email"`
	Name         string             `json:"name"`
	State        State              `json:"state"`
	Role         common.ProjectRole `json:"role"`
	VCSRole      string             `json:"vcsRole"`
	RoleProvider Type               `json:"roleProvider"`
}

// Provider is the interface for VCS provider.
type Provider interface {
	// Returns the API URL for a given VCS instance URL
	APIURL(instanceURL string) string
	// Try to use this provider as an auth provider and fetch the user info from the OAuth context
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	TryLogin(ctx context.Context, oauthCtx common.OauthContext, instanceURL string) (*UserInfo, error)
	// Fetch the user info of the given userID
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	// userID: the ID of the desired user
	FetchUserInfo(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, userID int) (*UserInfo, error)
	// Fetch all active members of a given repository
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	FetchRepositoryActiveMemberList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string) ([]*RepositoryMember, error)
	// Fetch the repository file list
	//
	// oauthCtx: OAuth context to read the repository tree
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// ref: the unique name of a repository tree, could be a branch name in GitLab or a tree sha in GitHub
	// path: the path inside repository, used to get content of subdirectories
	FetchRepositoryFileList(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, ref string, path string) ([]*RepositoryTreeNode, error)
	// Commits a new file
	//
	// oauthCtx: OAuth context to write the file content
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// filePath: file path to be written
	// fileCommit: the new file commit info
	CreateFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, fileCommit FileCommitCreate) error
	// Overwrites an existing file
	//
	// Similar to CreateFile except it overwrites an existing file. The fileCommit shoud includes the "LastCommitID" field which is used to detect conflicting writes.
	OverwriteFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, fileCommit FileCommitCreate) error
	// Reads the file content. Returns an io.ReadCloser on success. If file does not exist, returns NotFound error.
	//
	// oauthCtx: OAuth context to read the file content
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// filePath: file path to be read
	// commitID: the specific version to be read
	ReadFile(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, commitID string) (string, error)
	// Reads the file metadata. Returns the file meta on success.
	//
	// Similar to ReadFile except it specifies a branch instead of a commitID.
	ReadFileMeta(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, filePath string, branch string) (*FileMeta, error)
	// Creates a webhook. Returns the created webhook ID on succeess.
	//
	// oauthCtx: OAuth context to create the webhook
	// instanceURL: VCS instance URL
	// repositoryID: the repository ID from the external VCS system (note this is NOT the ID of Bytebase's own repository resource)
	// payload: the webhook payload
	CreateWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, payload []byte) (string, error)
	// Patches a webhook.
	//
	// The payload stores the patched field(s).
	PatchWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, webhookID string, payload []byte) error
	// Deletes a webhook.
	DeleteWebhook(ctx context.Context, oauthCtx common.OauthContext, instanceURL string, repositoryID string, webhookID string) error
}

var (
	providerMu sync.RWMutex
	providers  = make(map[Type]providerFunc)
)

// ProviderConfig is the provider configuration.
type ProviderConfig struct {
	Logger *zap.Logger
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

// Get returns a vcs provider specified by its vcs type
func Get(vcsType Type, providerConfig ProviderConfig) Provider {
	providerMu.RLock()
	f, ok := providers[vcsType]
	providerMu.RUnlock()
	if !ok {
		panic("vcs: unknown provider " + vcsType)
	}

	return f(providerConfig)
}
