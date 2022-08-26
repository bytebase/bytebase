// Package s3 provides the client for AWS S3 storage.
package s3

import (
	"context"
	"io"

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
// TODO(dragonly): support pagination.
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	resp, err := c.c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, err
	}
	var list []string
	for _, obj := range resp.Contents {
		list = append(list, *obj.Key)
	}
	return list, nil
}

// DownloadObject downloads the object with path.
// Defaults to multipart download with chunk size 5MB.
func (c *Client) DownloadObject(ctx context.Context, path string, w io.WriterAt) error {
	downloader := manager.NewDownloader(c.c)
	_, err := downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	})
	return err
}

// UploadObject uploads an object with the path.
// Defaults to multipart upload with chunk size 5MB.
func (c *Client) UploadObject(ctx context.Context, path string, body io.Reader) error {
	uploader := manager.NewUploader(c.c)
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:            &c.bucket,
		Key:               &path,
		Body:              body,
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	})
	return err
}

// DeleteObject deletes the object with path.
func (c *Client) DeleteObject(ctx context.Context, path string) (*s3.DeleteObjectOutput, error) {
	return c.c.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &c.bucket,
		Key:    &path,
	})
}

// GetBucket returns the bucket.
func (c *Client) GetBucket() string {
	return c.bucket
}
