package store

import (
	"context"
	"net"
	"net/url"
	"strings"

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

type awsMetadataDBTokenProvider struct{}

func (*awsMetadataDBTokenProvider) BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", errors.Wrap(err, "failed to load AWS config")
	}
	token, err := auth.BuildAuthToken(ctx, endpoint, region, user, cfg.Credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}
	return token, nil
}

func parseMetadataDBAuthConfig(pgURL string) (string, *metadataDBAuthConfig, error) {
	if !strings.HasPrefix(pgURL, "postgres://") && !strings.HasPrefix(pgURL, "postgresql://") {
		if strings.Contains(pgURL, metadataDBAWSRDSIAMParam) || strings.Contains(pgURL, metadataDBAWSRegionParam) {
			return "", nil, errors.New("metadata database AWS RDS IAM auth requires a postgres:// or postgresql:// URL")
		}
		return pgURL, nil, nil
	}

	u, err := url.Parse(pgURL)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse database URL")
	}

	q := u.Query()
	iamEnabled := q.Get(metadataDBAWSRDSIAMParam) == "true"
	region := q.Get(metadataDBAWSRegionParam)
	q.Del(metadataDBAWSRDSIAMParam)
	q.Del(metadataDBAWSRegionParam)
	u.RawQuery = q.Encode()
	cleanURL := u.String()

	if !iamEnabled {
		return cleanURL, nil, nil
	}

	if region == "" {
		return "", nil, errors.Errorf("%s is required when metadata database AWS RDS IAM auth is enabled", metadataDBAWSRegionParam)
	}

	user := u.User.Username()
	if user == "" {
		return "", nil, errors.New("database user is required when metadata database AWS RDS IAM auth is enabled")
	}

	host := u.Hostname()
	if host == "" {
		return "", nil, errors.New("database host is required when metadata database AWS RDS IAM auth is enabled")
	}

	port := u.Port()
	if port == "" {
		return "", nil, errors.New("database port is required when metadata database AWS RDS IAM auth is enabled")
	}

	return cleanURL, &metadataDBAuthConfig{
		enabled:  true,
		region:   region,
		endpoint: net.JoinHostPort(host, port),
		user:     user,
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
