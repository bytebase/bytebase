package dbauth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type contextKey struct{}

type fakeTokenProvider struct {
	token string
	err   error
	calls []fakeTokenProviderCall
}

type fakeTokenProviderCall struct {
	ctx      context.Context
	endpoint string
	region   string
	user     string
}

func (p *fakeTokenProvider) BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error) {
	p.calls = append(p.calls, fakeTokenProviderCall{
		ctx:      ctx,
		endpoint: endpoint,
		region:   region,
		user:     user,
	})
	return p.token, p.err
}

func TestAWSConfigFromPGXConfigDisabled(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestAWSConfigFromPGXConfigEnabledURL(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigEnabledKeywordValue(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb_meta dbname=bytebase sslmode=verify-full bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigStripsDisabledParams(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=false&bytebase_aws_region=us-east-1&sslmode=verify-full")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigRequiresFields(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*pgx.ConnConfig)
		wantErr   string
	}{
		{
			name:      "region",
			configure: func(config *pgx.ConnConfig) { delete(config.RuntimeParams, awsRegionParam) },
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
			pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")
			tt.configure(pgxConfig)

			_, err := awsConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAWSConfigFromPGXConfigNonURIAllowsParamNamesInValues(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb password=bytebase_aws_region application_name=bytebase_aws_rds_iam")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestAWSConfigFromPGXConfigRequiresVerifiedTLS(t *testing.T) {
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
			pgxConfig := mustParsePGXConfig(t, tt.pgURL)

			_, err := awsConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, "verified TLS is required")
		})
	}
}

func TestAWSConfigFromPGXConfigRejectsFallbacks(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=prefer&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	_, err := awsConfigFromPGXConfig(pgxConfig)

	require.ErrorContains(t, err, "fallback hosts or TLS fallback")
}

func TestNewAWSBeforeConnectSetsPassword(t *testing.T) {
	tokenProvider := &fakeTokenProvider{token: "generated-token"}
	hook := newAWSBeforeConnect(&awsConfig{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}, tokenProvider)
	connConfig := &pgx.ConnConfig{}
	ctx := context.WithValue(context.Background(), contextKey{}, "marker")

	err := hook(ctx, connConfig)

	require.NoError(t, err)
	require.Equal(t, "generated-token", connConfig.Password)
	require.Equal(t, []fakeTokenProviderCall{
		{
			ctx:      ctx,
			endpoint: "example.us-east-1.rds.amazonaws.com:5432",
			region:   "us-east-1",
			user:     "bb_meta",
		},
	}, tokenProvider.calls)
}

func TestNewAWSBeforeConnectReturnsTokenError(t *testing.T) {
	tokenProvider := &fakeTokenProvider{err: errors.New("credential chain failed")}
	hook := newAWSBeforeConnect(&awsConfig{
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

func TestAWSOpenOptions(t *testing.T) {
	require.Empty(t, awsOpenOptions(nil, &fakeTokenProvider{}))

	authConfig := &awsConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}

	require.Len(t, awsOpenOptions(authConfig, &fakeTokenProvider{}), 1)
}

func TestConfigure(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")

	options, err := Configure(context.Background(), pgxConfig)

	require.NoError(t, err)
	require.Empty(t, options)
}

func TestConfigureRequiresAWSRegion(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true")

	_, err := Configure(context.Background(), pgxConfig)

	require.ErrorContains(t, err, "bytebase_aws_region is required")
}

func TestConfigureEnabledAWSReturnsOpenOption(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	options, err := Configure(context.Background(), pgxConfig)

	require.NoError(t, err)
	require.Len(t, options, 1)
}

func TestIsKeywordValueRuntimeParam(t *testing.T) {
	require.True(t, IsKeywordValueRuntimeParam("bytebase_aws_rds_iam"))
	require.True(t, IsKeywordValueRuntimeParam("bytebase_aws_region"))
	require.False(t, IsKeywordValueRuntimeParam("host"))
}

func mustParsePGXConfig(t *testing.T, pgURL string) *pgx.ConnConfig {
	t.Helper()

	pgxConfig, err := pgx.ParseConfig(pgURL)
	require.NoError(t, err)
	return pgxConfig
}
