package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
