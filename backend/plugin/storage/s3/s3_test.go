package s3

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
)

const (
	region = "us-east-1"
	bucket = "bytebase-lyl-dev"
)

var (
	credentials = aws.Credentials{
		AccessKeyID:     os.Getenv("ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("SECRET_ACCESS_KEY"),
	}
)

// Only for manual test.
// Should be skipped in CI.
func TestS3Operations(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	client, err := NewClient(ctx, region, bucket, credentials)
	a.NoError(err)

	t.Run("ListObjects", func(t *testing.T) {
		list, err := client.ListObjects(ctx, "backup/")
		a.NoError(err)
		for _, obj := range list {
			slog.Info("Object", slog.String("Key", *obj.Key), slog.Time("LastModified", *obj.LastModified))
		}
	})

	t.Run("UploadObjects", func(t *testing.T) {
		buf := make([]byte, 10*1024*1024)
		blob := bytes.NewReader(buf)
		resp, err := client.UploadObject(ctx, "backup/test/blob", blob)
		a.NoError(err)
		slog.Info("Uploaded", slog.String("name", *resp.Key))
	})

	t.Run("DownloadObjects", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "blob")
		a.NoError(err)
		n, err := client.DownloadObject(ctx, "backup/test/blob", file)
		a.NoError(err)
		slog.Info("Downloaded", slog.Int64("length", n))
	})

	t.Run("DeleteObjects", func(t *testing.T) {
		resp, err := client.DeleteObjects(ctx, "backup/test/blob")
		a.NoError(err)
		slog.Info("Deleted", slog.Any("meta", resp.ResultMetadata))
	})
}
