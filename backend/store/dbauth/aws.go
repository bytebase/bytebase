package dbauth

import (
	"context"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

const (
	awsRDSIAMParam = "bytebase_aws_rds_iam"
	awsRegionParam = "bytebase_aws_region"
)

type awsConfig struct {
	enabled  bool
	region   string
	endpoint string
	user     string
}

type awsTokenProvider interface {
	BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error)
}

type awsMetadataDBTokenProvider struct {
	credentials aws.CredentialsProvider
}

func newAWSMetadataDBTokenProvider(ctx context.Context, region string) (*awsMetadataDBTokenProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS config")
	}

	return &awsMetadataDBTokenProvider{
		credentials: cfg.Credentials,
	}, nil
}

func (p *awsMetadataDBTokenProvider) BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error) {
	token, err := auth.BuildAuthToken(ctx, endpoint, region, user, p.credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}
	return token, nil
}

func awsConfigFromPGXConfig(pgxConfig *pgx.ConnConfig) (*awsConfig, error) {
	iamEnabled := pgxConfig.RuntimeParams[awsRDSIAMParam] == "true"
	region := pgxConfig.RuntimeParams[awsRegionParam]
	delete(pgxConfig.RuntimeParams, awsRDSIAMParam)
	delete(pgxConfig.RuntimeParams, awsRegionParam)

	if !iamEnabled {
		return nil, nil
	}

	if region == "" {
		return nil, errors.Errorf("%s is required when metadata database AWS RDS IAM auth is enabled", awsRegionParam)
	}

	if pgxConfig.User == "" {
		return nil, errors.New("database user is required when metadata database AWS RDS IAM auth is enabled")
	}

	if pgxConfig.Host == "" {
		return nil, errors.New("database host is required when metadata database AWS RDS IAM auth is enabled")
	}

	if pgxConfig.Port == 0 {
		return nil, errors.New("database port is required when metadata database AWS RDS IAM auth is enabled")
	}

	if len(pgxConfig.Fallbacks) > 0 {
		return nil, errors.New("metadata database AWS RDS IAM auth does not support fallback hosts or TLS fallback")
	}

	if pgxConfig.TLSConfig == nil || pgxConfig.TLSConfig.InsecureSkipVerify || pgxConfig.TLSConfig.ServerName == "" {
		return nil, errors.New("verified TLS is required when metadata database AWS RDS IAM auth is enabled")
	}

	return &awsConfig{
		enabled:  true,
		region:   region,
		endpoint: net.JoinHostPort(pgxConfig.Host, strconv.FormatUint(uint64(pgxConfig.Port), 10)),
		user:     pgxConfig.User,
	}, nil
}

func newAWSBeforeConnect(authConfig *awsConfig, tokenProvider awsTokenProvider) func(context.Context, *pgx.ConnConfig) error {
	return func(ctx context.Context, connConfig *pgx.ConnConfig) error {
		token, err := tokenProvider.BuildAuthToken(ctx, authConfig.endpoint, authConfig.region, authConfig.user)
		if err != nil {
			return errors.Wrapf(err, "failed to build metadata database AWS RDS IAM auth token for endpoint %q, region %q, user %q", authConfig.endpoint, authConfig.region, authConfig.user)
		}
		connConfig.Password = token
		return nil
	}
}

func awsOpenOptions(authConfig *awsConfig, tokenProvider awsTokenProvider) []stdlib.OptionOpenDB {
	if authConfig == nil || !authConfig.enabled {
		return nil
	}
	return []stdlib.OptionOpenDB{stdlib.OptionBeforeConnect(newAWSBeforeConnect(authConfig, tokenProvider))}
}
