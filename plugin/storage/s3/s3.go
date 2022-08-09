package s3

import (
	"context"
	"fmt"
	"io"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Credentials is the AWS S3 credentials.
type Credentials struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// Client wraps the AWS S3 client.
type Client struct {
	c      *s3.Client
	bucket string
}

// NewClient returns a new AWS S3 client.
func NewClient(ctx context.Context, region, bucket string, credentials Credentials) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscredentials.NewStaticCredentialsProvider(credentials.AccessKeyID, credentials.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS S3 config, error: %w", err)
	}
	s3client := s3.NewFromConfig(cfg)
	client := &Client{
		c:      s3client,
		bucket: bucket,
	}
	return client, nil
}

// ListObjects lists objects with prefix in their names.
func (c *Client) ListObjects(ctx context.Context, prefix string) (*s3.ListObjectsV2Output, error) {
	params := &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	}
	return c.c.ListObjectsV2(ctx, params)
}

// DownloadObject downloads the object with path.
// Defaults to multipart download with chunk size 5MB.
func (c *Client) DownloadObject(ctx context.Context, path string, w io.WriterAt) (int64, error) {
	downloader := manager.NewDownloader(c.c)
	params := &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	}
	return downloader.Download(ctx, w, params)
}

// UploadObject uploads an object with the path.
// Defaults to multipart upload with chunk size 5MB.
func (c *Client) UploadObject(ctx context.Context, path string, metadata map[string]string, body io.Reader) (*manager.UploadOutput, error) {
	uploader := manager.NewUploader(c.c)
	params := &s3.PutObjectInput{
		Bucket:            &c.bucket,
		Key:               &path,
		Metadata:          metadata,
		Body:              body,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	}
	return uploader.Upload(ctx, params)
}

// DeleteObject deletes the object with path.
func (c *Client) DeleteObject(ctx context.Context, path string) (*s3.DeleteObjectOutput, error) {
	params := &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	}
	return c.c.DeleteObject(ctx, params)
}
