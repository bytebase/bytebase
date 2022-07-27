package fake

import (
	"net"
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
	CreateRepository(id string)
	// SendWebhookPush sends out a webhook for a push event for the repository using
	// given payload.
	SendWebhookPush(repositoryID string, payload []byte) error
	// AddFiles adds given files to the repository.
	AddFiles(repositoryID string, files map[string]string) error
	// GetFiles returns files with given paths from the repository.
	GetFiles(repositoryID string, filePaths ...string) (map[string]string, error)
}
