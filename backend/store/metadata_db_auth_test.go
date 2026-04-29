package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakeMetadataDBTokenProvider struct {
	token string
	err   error
	calls []fakeMetadataDBTokenProviderCall
}

type fakeMetadataDBTokenProviderCall struct {
	endpoint string
	region   string
	user     string
}

func (p *fakeMetadataDBTokenProvider) BuildAuthToken(_ context.Context, endpoint, region, user string) (string, error) {
	p.calls = append(p.calls, fakeMetadataDBTokenProviderCall{
		endpoint: endpoint,
		region:   region,
		user:     user,
	})
	return p.token, p.err
}

func TestMetadataDBAuthConfigFromPGXConfigDisabled(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")

	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestMetadataDBAuthConfigFromPGXConfigEnabledURL(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRegionParam)
}

func TestMetadataDBAuthConfigFromPGXConfigEnabledKeywordValue(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb_meta dbname=bytebase sslmode=verify-full bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1")

	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRegionParam)
}

func TestMetadataDBAuthConfigFromPGXConfigStripsDisabledBytebaseAWSParams(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=false&bytebase_aws_region=us-east-1&sslmode=verify-full")

	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, metadataDBAWSRegionParam)
}

func TestMetadataDBAuthConfigFromPGXConfigRequiresFields(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*pgx.ConnConfig)
		wantErr   string
	}{
		{
			name:      "region",
			configure: func(config *pgx.ConnConfig) { delete(config.RuntimeParams, metadataDBAWSRegionParam) },
			wantErr:   "bytebase_aws_region is required",
		},
		{
			name:      "user",
			configure: func(config *pgx.ConnConfig) { config.User = "" },
			wantErr:   "database user is required",
		},
		{
			name:      "host",
			configure: func(config *pgx.ConnConfig) { config.Host = "" },
			wantErr:   "database host is required",
		},
		{
			name:      "port",
			configure: func(config *pgx.ConnConfig) { config.Port = 0 },
			wantErr:   "database port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgxConfig := mustParseMetadataDBPGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")
			tt.configure(pgxConfig)

			_, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMetadataDBAuthConfigFromPGXConfigNonURIAllowsAWSParamNamesInValues(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb password=bytebase_aws_region application_name=bytebase_aws_rds_iam")

	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestMetadataDBAuthConfigFromPGXConfigRequiresVerifiedTLS(t *testing.T) {
	tests := []struct {
		name  string
		pgURL string
	}{
		{
			name:  "disable",
			pgURL: "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=disable&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
		},
		{
			name:  "require",
			pgURL: "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=require&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgxConfig := mustParseMetadataDBPGXConfig(t, tt.pgURL)

			_, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, "verified TLS is required")
		})
	}
}

func TestMetadataDBAuthConfigFromPGXConfigRejectsFallbacks(t *testing.T) {
	pgxConfig := mustParseMetadataDBPGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=prefer&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	_, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)

	require.ErrorContains(t, err, "fallback hosts or TLS fallback")
}

func TestNewMetadataDBBeforeConnectSetsPassword(t *testing.T) {
	tokenProvider := &fakeMetadataDBTokenProvider{token: "generated-token"}
	hook := newMetadataDBBeforeConnect(&metadataDBAuthConfig{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}, tokenProvider)
	connConfig := &pgx.ConnConfig{}

	err := hook(context.Background(), connConfig)

	require.NoError(t, err)
	require.Equal(t, "generated-token", connConfig.Password)
	require.Equal(t, []fakeMetadataDBTokenProviderCall{
		{
			endpoint: "example.us-east-1.rds.amazonaws.com:5432",
			region:   "us-east-1",
			user:     "bb_meta",
		},
	}, tokenProvider.calls)
}

func TestNewMetadataDBBeforeConnectReturnsTokenError(t *testing.T) {
	tokenProvider := &fakeMetadataDBTokenProvider{err: errors.New("credential chain failed")}
	hook := newMetadataDBBeforeConnect(&metadataDBAuthConfig{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}, tokenProvider)
	connConfig := &pgx.ConnConfig{}

	err := hook(context.Background(), connConfig)

	require.ErrorContains(t, err, "failed to build metadata database AWS RDS IAM auth token")
	require.ErrorContains(t, err, "example.us-east-1.rds.amazonaws.com:5432")
	require.ErrorContains(t, err, "us-east-1")
	require.ErrorContains(t, err, "bb_meta")
	require.Empty(t, connConfig.Password)
}

func TestMetadataDBOpenOptions(t *testing.T) {
	require.Empty(t, metadataDBOpenOptions(nil, &fakeMetadataDBTokenProvider{}))

	authConfig := &metadataDBAuthConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}

	require.Len(t, metadataDBOpenOptions(authConfig, &fakeMetadataDBTokenProvider{}), 1)
}

func mustParseMetadataDBPGXConfig(t *testing.T, pgURL string) *pgx.ConnConfig {
	t.Helper()

	pgxConfig, err := pgx.ParseConfig(pgURL)
	require.NoError(t, err)
	return pgxConfig
}
