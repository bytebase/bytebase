package store

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestQueryExecModeForCustomPlans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode         pgx.QueryExecMode
		withCache    pgx.QueryExecMode
		withoutCache pgx.QueryExecMode
	}{
		{pgx.QueryExecModeCacheStatement, pgx.QueryExecModeCacheDescribe, pgx.QueryExecModeDescribeExec},
		{pgx.QueryExecModeCacheDescribe, pgx.QueryExecModeCacheDescribe, pgx.QueryExecModeDescribeExec},
		{pgx.QueryExecModeDescribeExec, pgx.QueryExecModeDescribeExec, pgx.QueryExecModeDescribeExec},
		{pgx.QueryExecModeExec, pgx.QueryExecModeExec, pgx.QueryExecModeExec},
		{pgx.QueryExecModeSimpleProtocol, pgx.QueryExecModeSimpleProtocol, pgx.QueryExecModeSimpleProtocol},
	} {
		for _, capacity := range []int{-1, 0, 512} {
			t.Run(tc.mode.String()+"/capacity="+strconv.Itoa(capacity), func(t *testing.T) {
				t.Parallel()
				config, err := pgx.ParseConfig("postgresql://user:password@127.0.0.1:1/db")
				require.NoError(t, err)
				config.DefaultQueryExecMode = tc.mode
				config.DescriptionCacheCapacity = capacity
				want := tc.withoutCache
				if capacity > 0 {
					want = tc.withCache
				}
				require.Equal(t, want, queryExecModeForCustomPlans(config))
			})
		}
	}
}

func TestIsFilePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "postgres URL",
			in:   "postgres://bb@example.com:5432/bytebase",
			want: false,
		},
		{
			name: "postgresql URL",
			in:   "postgresql://bb@example.com:5432/bytebase",
			want: false,
		},
		{
			name: "keyword value DSN",
			in:   "host=example.com port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "keyword value IAM DSN",
			in:   "host=example.com port=5432 user=bb dbname=bytebase bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1 sslmode=verify-full",
			want: false,
		},
		{
			name: "keyword value GCP IAM DSN",
			in:   "user=bb dbname=bytebase bytebase_gcp_cloud_sql_iam=true bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance",
			want: false,
		},
		{
			name: "keyword value GCP IAM DSN with runtime parameter first",
			in:   "bytebase_gcp_cloud_sql_iam=true bytebase_gcp_cloud_sql_instance_connection_name=project:region:instance user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "keyword value DSN with runtime parameter first",
			in:   "application_name=bytebase host=example.com port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "unix socket keyword value DSN",
			in:   "host=/tmp port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "file path",
			in:   "/var/run/bytebase/pg-url",
			want: true,
		},
		{
			name: "file path with equals",
			in:   "/var/run/bytebase/pg-url=prod",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isFilePath(tt.in))
		})
	}
}

// TestDBConnectionManagerGetDBAfterClose pins what a runner that outlives the
// server sees: a nil handle here panicked the process instead of erroring.
func TestDBConnectionManagerGetDBAfterClose(t *testing.T) {
	t.Parallel()
	// Nothing connects: sql.Open is lazy, and the test never queries.
	db, err := sql.Open("pgx", "postgresql://user:password@127.0.0.1:1/db")
	require.NoError(t, err)
	m := &DBConnectionManager{db: db, stopWatcher: make(chan struct{})}

	require.NoError(t, m.Close())

	require.Same(t, db, m.GetDB())
	require.ErrorContains(t, m.GetDB().PingContext(context.Background()), "database is closed")
	require.NoError(t, m.Close(), "Close is idempotent")
}
