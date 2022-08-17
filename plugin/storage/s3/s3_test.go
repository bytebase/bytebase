package s3

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytebase/bytebase/common/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Credentials is the AWS S3 credentials.
type Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

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

func TestS3Operations(t *testing.T) {
	t.Skip()
	a := require.New(t)
	ctx := context.Background()
	credentialsFileName := writeTempCredentialsFile(t, credentials)
	client, err := NewClient(ctx, region, bucket, credentialsFileName)
	a.NoError(err)

	t.Run("ListObjects", func(t *testing.T) {
		resp, err := client.ListObjects(ctx, "backup/")
		a.NoError(err)
		for _, obj := range resp.Contents {
			log.Info("Object", zap.Any("*", obj))
		}
	})

	t.Run("UploadObjects", func(t *testing.T) {
		buf := make([]byte, 10*1024*1024)
		blob := bytes.NewReader(buf)
		resp, err := client.UploadObject(ctx, "backup/test/blob", blob)
		a.NoError(err)
		log.Info("Uploaded", zap.String("name", *resp.Key))
	})

	t.Run("DownloadObjects", func(t *testing.T) {
		file, err := os.CreateTemp(t.TempDir(), "blob")
		a.NoError(err)
		n, err := client.DownloadObject(ctx, "backup/test/blob", file)
		a.NoError(err)
		log.Info("Downloaded", zap.Int64("length", n))
	})

	t.Run("DeleteObjects", func(t *testing.T) {
		resp, err := client.DeleteObject(ctx, "backup/test/blob")
		a.NoError(err)
		log.Info("Deleted", zap.Any("meta", resp.ResultMetadata))
	})
}

// nolint
func writeTempCredentialsFile(t *testing.T, credentials Credentials) string {
	a := require.New(t)
	file, err := os.Create(filepath.Join(t.TempDir(), "aws_credential"))
	defer file.Close()
	a.NoError(err)
	template := `[default]
aws_access_key_id = %s
aws_secret_access_key = %s`
	_, err = file.WriteString(fmt.Sprintf(template, credentials.AccessKeyID, credentials.SecretAccessKey))
	a.NoError(err)
	return file.Name()
}
