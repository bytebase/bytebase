// Package sharepoint uploads export files to a SharePoint document library
// through Microsoft Graph, authenticating as a workspace's Entra application.
package sharepoint

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

// Config is the workspace's Entra application.
type Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// Location is a document library root or folder that a URL resolved to.
type Location struct {
	DriveID string
	ItemID  string
	WebURL  string
}

// File is one file of an export.
type File struct {
	Name    string
	Content []byte
}

// Folder is the folder an export was uploaded into.
type Folder struct {
	Name   string
	WebURL string
}

var (
	// ErrNotFoundOrNoAccess means the destination does not exist or the
	// application cannot access it. SharePoint does not distinguish the two.
	ErrNotFoundOrNoAccess = errors.New("the SharePoint destination does not exist or Bytebase cannot access it")
	// ErrCredential means Entra rejected the application's credentials.
	ErrCredential = errors.New("Microsoft Entra rejected the SharePoint application credentials")
	// ErrUnconfirmed means an upload could not be confirmed after retries and
	// reconciliation; some files may exist in SharePoint.
	ErrUnconfirmed = errors.New("the upload to SharePoint could not be confirmed; files may exist in the destination")
	// ErrNothingUploaded marks an Upload failure before any file was
	// uploaded. It wraps the cause, so errors.Is still matches the cause.
	ErrNothingUploaded = errors.New("nothing was uploaded to SharePoint")
)

type nothingUploadedError struct{ err error }

func (e *nothingUploadedError) Error() string      { return e.err.Error() }
func (e *nothingUploadedError) Unwrap() error      { return e.err }
func (*nothingUploadedError) Is(target error) bool { return target == ErrNothingUploaded }

// Client resolves destinations and uploads exports.
type Client interface {
	// Resolve resolves a canonical URL from NormalizeURL to a library root or
	// folder. Resolving proves only that the destination is readable.
	Resolve(ctx context.Context, canonicalURL string) (*Location, error)
	// Upload creates folderName under loc and uploads files into it. Retrying
	// Upload with the same arguments reuses the folder and files already
	// uploaded by the application instead of creating copies.
	Upload(ctx context.Context, loc *Location, folderName string, files []File) (*Folder, error)
}

type cachedClient struct {
	cfg    Config
	client Client
}

var (
	clientsMu sync.Mutex
	// clients holds one client per Entra application, so exports reuse its
	// cached access token.
	clients = map[string]cachedClient{}
)

// NewClient returns a Client for the Microsoft global commercial cloud. It
// returns the same Client for the same Config until the Config changes.
func NewClient(cfg Config) (Client, error) {
	key := cfg.TenantID + "/" + cfg.ClientID
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if cached, ok := clients[key]; ok && cached.cfg == cfg {
		return cached.client, nil
	}
	client, err := newGraphClient(cfg)
	if err != nil {
		return nil, err
	}
	clients[key] = cachedClient{cfg: cfg, client: client}
	return client, nil
}
