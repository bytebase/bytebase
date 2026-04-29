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

func TestParseMetadataDBAuthConfigDisabled(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")
	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.Equal(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
}

func TestParseMetadataDBAuthConfigEnabled(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")
	require.NoError(t, err)
	require.Equal(t, "postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
}

func TestParseMetadataDBAuthConfigStripsDisabledBytebaseAWSParams(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=false&bytebase_aws_region=us-east-1&sslmode=verify-full")
	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.Equal(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
}

func TestParseMetadataDBAuthConfigRequiresFields(t *testing.T) {
	tests := []struct {
		name    string
		pgURL   string
		wantErr string
	}{
		{
			name:    "region",
			pgURL:   "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=true",
			wantErr: "bytebase_aws_region is required",
		},
		{
			name:    "user",
			pgURL:   "postgres://example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database user is required",
		},
		{
			name:    "host",
			pgURL:   "postgres://bb@:5432/bytebase?bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database host is required",
		},
		{
			name:    "port",
			pgURL:   "postgres://bb@example.us-east-1.rds.amazonaws.com/bytebase?bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseMetadataDBAuthConfig(tt.pgURL)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestParseMetadataDBAuthConfigRejectsIAMForNonURI(t *testing.T) {
	_, _, err := parseMetadataDBAuthConfig("host=example.us-east-1.rds.amazonaws.com port=5432 user=bb bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1")
	require.ErrorContains(t, err, "metadata database AWS RDS IAM auth requires a postgres:// or postgresql:// URL")
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
