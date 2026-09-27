//nolint:revive
package util

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/pkg/errors"
	"google.golang.org/api/option"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func GetAWSConnectionConfig(ctx context.Context, connCfg db.ConnectionConfig) (aws.Config, error) {
	region := connCfg.DataSource.GetRegion()

	// Only use static credentials if access key is provided
	// If awsCredential exists but AccessKeyId is empty, fall back to default credential chain
	// (EC2 instance role, env vars, etc.) for cross-account role assumption
	if awsCredential := connCfg.DataSource.GetAwsCredential(); awsCredential != nil && awsCredential.AccessKeyId != "" {
		return config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				awsCredential.AccessKeyId,
				awsCredential.SecretAccessKey,
				awsCredential.SessionToken,
			)),
		)
	}

	// Use default credential chain when no static credentials provided
	return config.LoadDefaultConfig(ctx, config.WithRegion(region))
}

func GetGCPConnectionConfig(ctx context.Context, connCfg db.ConnectionConfig) (*cloudsqlconn.Dialer, error) {
	// WithIAMAuthN enables IAM database authentication.
	opts := []cloudsqlconn.Option{cloudsqlconn.WithIAMAuthN()}
	// WithCredentialsJSON provides Cloud SQL Admin API access when a service account key is set.
	if gcpCredential := connCfg.DataSource.GetGcpCredential(); gcpCredential != nil && len(gcpCredential.Content) > 0 {
		opts = append(opts, cloudsqlconn.WithCredentialsJSON([]byte(gcpCredential.Content)))
	}
	opts = append(opts, cloudSQLDialOptions(connCfg.DataSource)...)
	return cloudsqlconn.NewDialer(ctx, opts...)
}

// cloudSQLDialOptions selects the Cloud SQL IP type (private or PSC) for Google
// Cloud SQL IAM connections. Public IP is the cloudsqlconn default, so no option
// is needed for PUBLIC or unspecified.
func cloudSQLDialOptions(ds *storepb.DataSource) []cloudsqlconn.Option {
	switch ds.GetCloudSqlIpType() {
	case storepb.DataSource_PRIVATE:
		return []cloudsqlconn.Option{cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP())}
	case storepb.DataSource_PSC:
		return []cloudsqlconn.Option{cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPSC())}
	default:
		return nil
	}
}

// GCPCredentialOption returns the appropriate option.ClientOption for the given
// GCP credential JSON, supporting both service account keys and external account
// (Workload Identity Federation) configurations.
//
// SECURITY NOTE: ExternalAccount and ImpersonatedServiceAccount credential types
// do not validate the credential configuration. Malicious URLs in the config could
// pose a security risk. See https://cloud.google.com/docs/authentication/external/externally-sourced-credentials.
func GCPCredentialOption(credJSON []byte) (option.ClientOption, error) {
	var cred struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(credJSON, &cred); err != nil {
		return nil, errors.Wrap(err, "failed to parse GCP credential JSON")
	}
	switch cred.Type {
	case "service_account":
		return option.WithAuthCredentialsJSON(option.ServiceAccount, credJSON), nil
	case "external_account":
		return option.WithAuthCredentialsJSON(option.ExternalAccount, credJSON), nil
	case "impersonated_service_account":
		return option.WithAuthCredentialsJSON(option.ImpersonatedServiceAccount, credJSON), nil
	case "authorized_user":
		return option.WithAuthCredentialsJSON(option.AuthorizedUser, credJSON), nil
	case "":
		return nil, errors.Errorf("GCP credential JSON missing \"type\" field")
	default:
		return nil, errors.Errorf("unsupported GCP credential type: %q", cred.Type)
	}
}

func GetAzureConnectionConfig(connCfg db.ConnectionConfig) (azcore.TokenCredential, error) {
	if azureCredential := connCfg.DataSource.GetAzureCredential(); azureCredential != nil {
		c, err := azidentity.NewClientSecretCredential(azureCredential.TenantId, azureCredential.ClientId, azureCredential.ClientSecret, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create client secret credential")
		}
		return c, nil
	}

	c, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to found default Azure credential")
	}
	return c, nil
}
