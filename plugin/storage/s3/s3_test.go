package s3

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/bytebase/bytebase/common/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	region = "us-east-1"
	bucket = "bytebase-lyl-dev"
)

var (
	credentials = Credentials{
		AccessKeyID:     os.Getenv("ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("SECRET_ACCESS_KEY"),
	}
)

func TestListObjects(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	client, err := NewClient(ctx, region, bucket, credentials)
	a.NoError(err)

	resp, err := client.ListObjects(ctx, "backup/")
	a.NoError(err)
	for _, obj := range resp.Contents {
		log.Info("Object", zap.Any("*", obj))
	}
}

func TestUploadObjects(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	client, err := NewClient(ctx, region, bucket, credentials)
	a.NoError(err)

	metadata := make(map[string]string)
	metadata["project_id"] = "233"
	buf := make([]byte, 10*1024*1024)
	blob := bytes.NewReader(buf)
	resp, err := client.UploadObject(ctx, "backup/test/blob", metadata, blob)
	a.NoError(err)
	log.Info("Uploaded", zap.String("name", *resp.Key))
}

func TestDownloadObjects(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	client, err := NewClient(ctx, region, bucket, credentials)
	a.NoError(err)

	file, err := os.CreateTemp(t.TempDir(), "blob")
	a.NoError(err)
	n, err := client.DownloadObject(ctx, "backup/test/blob", file)
	a.NoError(err)
	log.Info("Downloaded", zap.Int64("length", n))
}

func TestDeleteObjects(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	client, err := NewClient(ctx, region, bucket, credentials)
	a.NoError(err)

	resp, err := client.DeleteObject(ctx, "backup/test/blob")
	a.NoError(err)
	log.Info("Deleted", zap.Any("meta", resp.ResultMetadata))
}
