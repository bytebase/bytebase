package storage

import (
	"context"
	"io"
)

// Uploader uploads contents to the cloud.
type Uploader interface {
	UploadObject(ctx context.Context, path string, body io.Reader) error
}

// Downloader downloads contents from the cloud.
type Downloader interface {
	ListObject(ctx context.Context, prefix string) ([]string, error)
	DownloadObject(ctx context.Context, path string, w io.WriterAt) error
}
