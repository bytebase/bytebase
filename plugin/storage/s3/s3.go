// Package s3 provides the client for AWS S3 storage.
package s3

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
)

// Client wraps the AWS S3 client.
type Client struct {
	c      *s3.Client
	bucket string
}

// GetCredentialsFromFile load AWS credentials from file.
func GetCredentialsFromFile(ctx context.Context, credentialsFileName string) (aws.Credentials, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedCredentialsFiles([]string{credentialsFileName}),
	)
	if err != nil {
		return aws.Credentials{}, errors.Wrap(err, "failed to load AWS S3 config")
	}
	credentials, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, errors.Wrap(err, "failed to retrieve AWS S3 credentials")
	}
	return credentials, nil
}

// NewClient returns a new AWS S3 client.
func NewClient(ctx context.Context, region, bucket string, credentials aws.Credentials) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(credentials.AccessKeyID, credentials.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS S3 config")
	}
	return &Client{
		c:      s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// ListObjects lists objects with prefix in their names.
func (c *Client) ListObjects(ctx context.Context, prefix string) (*s3.ListObjectsV2Output, error) {
	return c.c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})
}

// DownloadObject downloads the object with path.
// Defaults to multipart download with chunk size 5MB.
func (c *Client) DownloadObject(ctx context.Context, path string, w io.WriterAt) (int64, error) {
	downloader := manager.NewDownloader(c.c)
	return downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	})
}

// UploadObject uploads an object with the path.
// Defaults to multipart upload with chunk size 5MB.
func (c *Client) UploadObject(ctx context.Context, path string, body io.Reader) (*manager.UploadOutput, error) {
	uploader := manager.NewUploader(c.c)
	return uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:            &c.bucket,
		Key:               &path,
		Body:              body,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	})
}

// DeleteObjects deletes the objects with path.
func (c *Client) DeleteObjects(ctx context.Context, pathList ...string) (*s3.DeleteObjectsOutput, error) {
	var oidList []types.ObjectIdentifier
	for _, path := range pathList {
		path := path // create a new 'path'.
		oidList = append(oidList, types.ObjectIdentifier{Key: &path})
	}
	return c.c.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: &c.bucket,
		Delete: &types.Delete{Objects: oidList},
	})
}

// GetBucket returns the bucket.
func (c *Client) GetBucket() string {
	return c.bucket
}

// DownloadFileFromCloud downloads a binlog or metadata file from the cloud storage.
// In case of network errors which will get partially downloaded files, we first download to a temporary file.
// After that, we then rename it to the target file path.
func (c *Client) DownloadFileFromCloud(ctx context.Context, filePathLocal, filePathOnCloud string) error {
	tempDir := os.TempDir()
	baseName := filepath.Base(filePathLocal)
	filePathTemp := filepath.Join(tempDir, baseName)
	fileTemp, err := os.Create(filePathTemp)
	if err != nil {
		return errors.Wrapf(err, "failed to create the local temporary file %s", filePathTemp)
	}
	if _, err := c.DownloadObject(ctx, filePathOnCloud, fileTemp); err != nil {
		return errors.Wrapf(err, "failed to download file %q from the cloud storage", filePathOnCloud)
	}
	if err := os.Rename(filePathTemp, filePathLocal); err != nil {
		return errors.Wrapf(err, "failed to rename %q to %q", filePathTemp, filePathLocal)
	}
	return nil
}
