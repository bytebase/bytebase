package storage

import (
	"context"
	"io"
)

// Uploader uploads contents to the cloud.
type Uploader interface {
	UploadObject(ctx context.Context, path string, body io.Reader) error
}
