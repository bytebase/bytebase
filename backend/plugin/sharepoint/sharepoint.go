// Package sharepoint delivers export files to a SharePoint document library
// through Microsoft Graph, authenticating as a workspace's Entra application.
package sharepoint

import (
	"context"

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
	// SiteName and LibraryName are display names.
	SiteName    string
	LibraryName string
	// FolderPath is the path within the library; empty for the library root.
	FolderPath string
	WebURL     string
}

// File is one file of an export.
type File struct {
	Name    string
	Content []byte
}

// Delivery is the folder an export was delivered into.
type Delivery struct {
	FolderName string
	WebURL     string
}

var (
	// ErrNotFoundOrNoAccess means the destination does not exist or the
	// application cannot access it. SharePoint does not distinguish the two.
	ErrNotFoundOrNoAccess = errors.New("the SharePoint destination does not exist or Bytebase cannot access it")
	// ErrCredential means Entra rejected the application's credentials.
	ErrCredential = errors.New("Microsoft Entra rejected the SharePoint application credentials")
	// ErrUnconfirmed means delivery could not be confirmed after retries and
	// reconciliation; some files may exist in SharePoint.
	ErrUnconfirmed = errors.New("delivery to SharePoint could not be confirmed; files may exist in the destination")
	// ErrNothingDelivered marks a Deliver failure before any file was
	// uploaded. It wraps the cause, so errors.Is still matches the cause.
	ErrNothingDelivered = errors.New("nothing was delivered to SharePoint")
)

type nothingDeliveredError struct{ err error }

func (e *nothingDeliveredError) Error() string      { return e.err.Error() }
func (e *nothingDeliveredError) Unwrap() error      { return e.err }
func (*nothingDeliveredError) Is(target error) bool { return target == ErrNothingDelivered }

// Client resolves destinations and delivers exports.
type Client interface {
	// Resolve resolves a canonical URL from NormalizeURL to a library root or
	// folder. Resolving proves only that the destination is readable.
	Resolve(ctx context.Context, canonicalURL string) (*Location, error)
	// Deliver creates folderName under loc and uploads files into it. Retrying
	// Deliver with the same arguments reuses the folder and files already
	// delivered by the application instead of creating copies.
	Deliver(ctx context.Context, loc *Location, folderName string, files []File) (*Delivery, error)
}

// NewClient returns a Client for the Microsoft global commercial cloud.
func NewClient(cfg Config) (Client, error) {
	return newGraphClient(cfg)
}
