package store

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
	metadataDBAWSRDSIAMParam = "bytebase_aws_rds_iam"
	metadataDBAWSRegionParam = "bytebase_aws_region"
)

type metadataDBAuthConfig struct {
	enabled  bool
	region   string
	endpoint string
	user     string
}

type metadataDBTokenProvider interface {
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

func metadataDBAuthConfigFromPGXConfig(pgxConfig *pgx.ConnConfig) (*metadataDBAuthConfig, error) {
	iamEnabled := pgxConfig.RuntimeParams[metadataDBAWSRDSIAMParam] == "true"
	region := pgxConfig.RuntimeParams[metadataDBAWSRegionParam]
	delete(pgxConfig.RuntimeParams, metadataDBAWSRDSIAMParam)
	delete(pgxConfig.RuntimeParams, metadataDBAWSRegionParam)

	if !iamEnabled {
		return nil, nil
	}

	if region == "" {
		return nil, errors.Errorf("%s is required when metadata database AWS RDS IAM auth is enabled", metadataDBAWSRegionParam)
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

	return &metadataDBAuthConfig{
		enabled:  true,
		region:   region,
		endpoint: net.JoinHostPort(pgxConfig.Host, strconv.FormatUint(uint64(pgxConfig.Port), 10)),
		user:     pgxConfig.User,
	}, nil
}

func newMetadataDBBeforeConnect(authConfig *metadataDBAuthConfig, tokenProvider metadataDBTokenProvider) func(context.Context, *pgx.ConnConfig) error {
	return func(ctx context.Context, connConfig *pgx.ConnConfig) error {
		token, err := tokenProvider.BuildAuthToken(ctx, authConfig.endpoint, authConfig.region, authConfig.user)
		if err != nil {
			return errors.Wrapf(err, "failed to build metadata database AWS RDS IAM auth token for endpoint %q, region %q, user %q", authConfig.endpoint, authConfig.region, authConfig.user)
		}
		connConfig.Password = token
		return nil
	}
}

func metadataDBOpenOptions(authConfig *metadataDBAuthConfig, tokenProvider metadataDBTokenProvider) []stdlib.OptionOpenDB {
	if authConfig == nil || !authConfig.enabled {
		return nil
	}
	return []stdlib.OptionOpenDB{stdlib.OptionBeforeConnect(newMetadataDBBeforeConnect(authConfig, tokenProvider))}
}
