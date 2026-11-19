package cockroachdb

import (
	"context"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestCockroachConfig(t *testing.T) {
	taskRunUID := int64(7)
	for _, tc := range []struct {
		name   string
		config db.ConnectionConfig
		want   omnidriver.Config
	}{
		{
			name: "data source database",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Username: "dba", Password: "secret", Host: "crdb.example.com", Port: "26257", Database: "app"},
				Password:   "resolved",
			},
			want: omnidriver.Config{
				Host:            "crdb.example.com",
				Port:            "26257",
				Database:        "app",
				User:            "dba",
				Password:        "resolved",
				ApplicationName: "bytebase",
			},
		},
		{
			name: "connection context database wins",
			config: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Username: "dba", Host: "crdb.example.com", Port: "26257", Database: "app"},
				ConnectionContext: db.ConnectionContext{DatabaseName: "tenant", TaskRunUID: &taskRunUID, ReadOnly: true, EngineVersion: "25.2"},
			},
			want: omnidriver.Config{
				Host:            "crdb.example.com",
				Port:            "26257",
				Database:        "tenant",
				User:            "dba",
				ApplicationName: "bytebase-taskrun-7",
				ReadOnly:        true,
				EngineVersion:   "25.2",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cockroachConfig(tc.config)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCockroachConfigMapsTLSOnlyWhenAsked(t *testing.T) {
	got, err := cockroachConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{Username: "dba", Host: "crdb.example.com", Port: "26257"}})
	require.NoError(t, err)
	require.Nil(t, got.TLS)

	got, err = cockroachConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{
		Username: "dba",
		Host:     "crdb-1.example.com,crdb-2.example.com",
		Port:     "26257",
		UseSsl:   true,
	}})
	require.NoError(t, err)
	require.NotNil(t, got.TLS)
	tlsConfig, err := got.TLS("crdb-2.example.com")
	require.NoError(t, err)
	require.True(t, tlsConfig.InsecureSkipVerify)
	require.Nil(t, tlsConfig.Certificates)
}

func TestCockroachConfigRefusesHalfAClientCertificate(t *testing.T) {
	for _, dataSource := range []*storepb.DataSource{
		{Username: "dba", Host: "crdb.example.com", Port: "26257", SslCert: "cert"},
		{Username: "dba", Host: "crdb.example.com", Port: "26257", SslKey: "key"},
	} {
		_, err := cockroachConfig(db.ConnectionConfig{DataSource: dataSource})
		require.EqualError(t, err, "ssl-cert and ssl-key must be both set or unset")
	}
}

func TestGetRoutingIDFromCockroachCloudURL(t *testing.T) {
	for _, tc := range []struct {
		host string
		want string
	}{
		{host: "routing-id.cockroachlabs.cloud", want: "routing-id"},
		{host: "bytebase-cluster-7749.6xw.aws-ap-southeast-1.cockroachlabs.cloud", want: "bytebase-cluster-7749"},
		{host: "subdomain.routing-id.cockroachlabs.cloud", want: "subdomain"},
		{host: "cockroachlabs.cloud", want: ""},
		{host: "example.com", want: ""},
	} {
		require.Equal(t, tc.want, getRoutingIDFromCockroachCloudURL(tc.host), "host: %s", tc.host)
	}
}

// GetDB hands back a pool for db.SQLDB callers, and OwnsQuerySession keeps the
// editor from opening a second connection no statement uses.
func TestGetDBReturnsPool(t *testing.T) {
	driver, err := (&Driver{}).Open(context.Background(), storepb.Engine_COCKROACHDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Username: "dba", Host: "127.0.0.1", Port: "26257", Database: "app"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(context.Background()) })

	require.NotNil(t, db.SQLDB(driver))
	require.Same(t, db.SQLDB(driver), db.SQLDB(driver))
	require.True(t, db.OwnsQuerySession(driver))
}

func TestExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", explainFormat(v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", explainFormat(v1pb.QueryOption_TEXT))
	require.Equal(t, "TEXT", explainFormat(v1pb.QueryOption_XML))
	require.Equal(t, "TEXT", explainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
}

func TestExecuteOptions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		script     string
		options    db.ExecuteOptions
		wantScript string
		wantMode   omnidriver.TransactionMode
	}{
		{
			name:       "no directive is transactional",
			script:     "SELECT 1;",
			wantScript: "SELECT 1;",
			wantMode:   omnidriver.Transactional,
		},
		{
			name:       "directive off is auto commit and is stripped",
			script:     "-- txn-mode = off\nSELECT 1;",
			wantScript: "SELECT 1;",
			wantMode:   omnidriver.AutoCommit,
		},
		{
			name:       "an unknown mode stays transactional and keeps the comment",
			script:     "-- txn-mode = whatever\nSELECT 1;",
			wantScript: "-- txn-mode = whatever\nSELECT 1;",
			wantMode:   omnidriver.Transactional,
		},
		{
			name:       "create database passes through",
			script:     "CREATE DATABASE app;",
			options:    db.ExecuteOptions{CreateDatabase: true},
			wantScript: "CREATE DATABASE app;",
			wantMode:   omnidriver.Transactional,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script, options := executeOptions(tc.script, tc.options)
			require.Equal(t, tc.wantScript, script)
			require.Equal(t, tc.wantMode, options.Transaction)
			require.Equal(t, tc.options.CreateDatabase, options.CreateDatabase)
			require.NotNil(t, options.Log)
		})
	}
}

// executeLogger bridges the contract's steps onto the task run log in the shape
// the product reads.
func TestExecuteLogger(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	logger := executeLogger{opts: opts}
	logger.CommandExecute(omnidriver.Command{Text: "SELECT 1;", ByteStart: 14, ByteEnd: 23})
	logger.CommandResponse(2, []int64{1, 1}, nil)
	logger.TransactionControl(omnidriver.Begin, nil)
	logger.TransactionControl(omnidriver.Rollback, context.DeadlineExceeded)
	logger.Retry(context.DeadlineExceeded, 2, 3)

	require.Len(t, logs, 5)
	require.Equal(t, "SELECT 1;", logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, int64(2), logs[1].GetCommandResponse().GetAffectedRows())
	require.Equal(t, []int64{1, 1}, logs[1].GetCommandResponse().GetAllAffectedRows())
	require.Empty(t, logs[1].GetCommandResponse().GetError())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[2].GetTransactionControl().GetType())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK, logs[3].GetTransactionControl().GetType())
	require.Equal(t, context.DeadlineExceeded.Error(), logs[3].GetTransactionControl().GetError())
	require.Equal(t, int32(2), logs[4].GetRetryInfo().GetRetryCount())

	// Without the statement flag the log carries the command's range instead,
	// which is what the task run view highlights.
	logs = nil
	logger = executeLogger{opts: db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
		logs = append(logs, entry)
		return nil
	}}}
	logger.CommandExecute(omnidriver.Command{Text: "SELECT 1;", ByteStart: 14, ByteEnd: 23})
	require.Len(t, logs, 1)
	require.Empty(t, logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, int32(14), logs[0].GetCommandExecute().GetRange().GetStart())
	require.Equal(t, int32(23), logs[0].GetCommandExecute().GetRange().GetEnd())
}
