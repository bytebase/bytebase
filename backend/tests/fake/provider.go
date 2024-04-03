package fake

import (
	"net"

	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

// VCSProvider is a fake implementation of a VCS provider.
type VCSProvider interface {
	// Run starts the server of the VCS provider.
	Run() error
	// Close shuts down the server of the VCS provider.
	Close() error
	// ListenerAddr returns listener address of the server.
	ListenerAddr() net.Addr

	// CreateRepository creates a new repository with given ID.
	CreateRepository(repository *vcs.Repository) error
	// CreateBranch creates a new branch in the repository with given ID.
	CreateBranch(id, branchName string) error
	// SendWebhookPush sends out a webhook for a push event for the repository using
	// given payload.
	SendWebhookPush(repositoryID string, payload []byte) error
	// AddFiles adds given files to the repository.
	AddFiles(repositoryID string, files map[string]string) error
	// AddPullRequest creates a new pull request and add changed files to it.
	AddPullRequest(repositoryID string, prID int, files []*vcs.PullRequestFile) error
}

// VCSProviderCreator a function to create a new VCSProvider.
type VCSProviderCreator func(port int) VCSProvider
