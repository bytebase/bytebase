package dbauth

import (
	"context"
	"net"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakeGCPDialer struct {
	calls []fakeGCPDialerCall
	err   error
}

type fakeGCPDialerCall struct {
	ctx                    context.Context
	instanceConnectionName string
}

func (d *fakeGCPDialer) Dial(ctx context.Context, instanceConnectionName string) (net.Conn, error) {
	d.calls = append(d.calls, fakeGCPDialerCall{
		ctx:                    ctx,
		instanceConnectionName: instanceConnectionName,
	})
	return nil, d.err
}

func TestGCPConfigFromPGXConfigDisabled(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.com:5432/bytebase?sslmode=disable")

	authConfig, err := gcpConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestGCPConfigFromPGXConfigEnabledURL(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.com:5432/bytebase?bytebase_gcp_cloud_sql_iam=true&bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance")

	authConfig, err := gcpConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "project:region:instance", authConfig.instanceConnectionName)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLInstanceConnectionNameParam)
}

func TestGCPConfigFromPGXConfigEnabledKeywordValue(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "user=bb dbname=bytebase bytebase_gcp_cloud_sql_iam=true bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance")

	authConfig, err := gcpConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "project:region:instance", authConfig.instanceConnectionName)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLInstanceConnectionNameParam)
}

func TestGCPConfigFromPGXConfigStripsDisabledParams(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.com:5432/bytebase?bytebase_gcp_cloud_sql_iam=false&bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance")

	authConfig, err := gcpConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, gcpCloudSQLInstanceConnectionNameParam)
}

func TestGCPConfigFromPGXConfigRequiresInstanceConnectionName(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.com:5432/bytebase?bytebase_gcp_cloud_sql_iam=true")

	_, err := gcpConfigFromPGXConfig(pgxConfig)

	require.ErrorContains(t, err, "bytebase_gcp_cloud_sql_instance_connection_name is required")
}

func TestApplyGCPConfigSetsDialFunc(t *testing.T) {
	pgxConfig := &pgx.ConnConfig{}
	dialer := &fakeGCPDialer{}
	ctx := context.WithValue(context.Background(), contextKey{}, "marker")

	applyGCPConfig(pgxConfig, &gcpConfig{
		enabled:                true,
		instanceConnectionName: "project:region:instance",
	}, dialer)

	require.NotNil(t, pgxConfig.DialFunc)

	conn, err := pgxConfig.DialFunc(ctx, "tcp", "ignored:5432")

	require.NoError(t, err)
	require.Nil(t, conn)
	require.Equal(t, []fakeGCPDialerCall{
		{
			ctx:                    ctx,
			instanceConnectionName: "project:region:instance",
		},
	}, dialer.calls)
}

func TestConfigureRejectsMultipleMetadataDBAuthProviders(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1&bytebase_gcp_cloud_sql_iam=true&bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance")

	_, err := Configure(context.Background(), pgxConfig)

	require.ErrorContains(t, err, "multiple metadata database IAM auth providers are enabled")
}
