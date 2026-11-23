package cockroachdb

import (
	"context"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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

func TestTaskRunSessionFromResult(t *testing.T) {
	t.Parallel()
	sessionStart := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	queryStart := sessionStart.Add(5 * time.Minute)
	sessionID := "18e9a1f97a49d2ca0000000000000001"

	got := taskRunSessionFromResult(&omnidriver.Result{
		Rows: [][]omnidriver.Value{
			{
				{Kind: omnidriver.String, String: sessionID},
				{Kind: omnidriver.String, String: "SELECT pg_sleep(20)"},
				{Kind: omnidriver.String, String: "executing"},
				{Kind: omnidriver.String, String: "smoke"},
				{Kind: omnidriver.String, String: "root"},
				{Kind: omnidriver.String, String: "bytebase-taskrun-7"},
				{Kind: omnidriver.String, String: "[::1]:44768"},
				{Kind: omnidriver.TimestampTZ, Time: sessionStart},
				{Kind: omnidriver.TimestampTZ, Time: queryStart},
			},
		},
	}, "bytebase-taskrun-7")

	postgres := got.GetPostgres()
	require.NotNil(t, postgres)
	require.NotNil(t, postgres.Session)
	require.Equal(t, sessionID, postgres.Session.Pid)
	require.Equal(t, "bytebase-taskrun-7", postgres.Session.ApplicationName)
	require.Equal(t, "SELECT pg_sleep(20)", postgres.Session.Query)
	require.Equal(t, "executing", postgres.Session.GetState())
	require.Equal(t, "smoke", postgres.Session.GetDatname())
	require.Equal(t, "root", postgres.Session.GetUsename())
	require.Equal(t, "[::1]:44768", postgres.Session.GetClientAddr())
	require.Nil(t, postgres.Session.WaitEventType)
	require.Nil(t, postgres.Session.WaitEvent)
	require.Nil(t, postgres.Session.ClientPort)
	require.Empty(t, postgres.Session.BlockedByPids)
	require.Equal(t, timestamppb.New(sessionStart), postgres.Session.BackendStart)
	require.Nil(t, postgres.Session.XactStart)
	require.Equal(t, timestamppb.New(queryStart), postgres.Session.QueryStart)

	missingSessionStartID := "18e9a1f97a49d2ca0000000000000002"
	gotWithoutSessionStart := taskRunSessionFromResult(&omnidriver.Result{
		Rows: [][]omnidriver.Value{
			{
				{Kind: omnidriver.String, String: missingSessionStartID},
				{Kind: omnidriver.String, String: "SELECT pg_sleep(20)"},
				{Kind: omnidriver.String, String: "executing"},
				{Kind: omnidriver.String, String: "smoke"},
				{Kind: omnidriver.String, String: "root"},
				{Kind: omnidriver.String, String: "bytebase-taskrun-8"},
				{Kind: omnidriver.Null},
				{Kind: omnidriver.Null},
				{Kind: omnidriver.TimestampTZ, Time: queryStart},
			},
		},
	}, "bytebase-taskrun-8")

	withoutSessionStart := gotWithoutSessionStart.GetPostgres().GetSession()
	require.NotNil(t, withoutSessionStart)
	require.Equal(t, missingSessionStartID, withoutSessionStart.Pid)
	require.Nil(t, withoutSessionStart.ClientAddr)
	require.Nil(t, withoutSessionStart.BackendStart)
	require.Equal(t, timestamppb.New(queryStart), withoutSessionStart.QueryStart)
}

func TestExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", util.OmniExplainFormat(v1pb.QueryOption_JSON, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_TEXT, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_XML, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "TEXT", v1pb.QueryOption_JSON))
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
			script, options := util.BuildExecuteOptions(tc.script, tc.options, cockroachExecuteConfig)
			require.Equal(t, tc.wantScript, script)
			require.Equal(t, tc.wantMode, options.Transaction)
			require.Equal(t, tc.options.CreateDatabase, options.CreateDatabase)
			require.NotNil(t, options.Log)
		})
	}
}

// TestExecuteLogger verifies ExecuteLogger bridges the contract's steps onto the
// task run log in the shape the product reads.
func TestExecuteLogger(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	logger := util.NewExecuteLogger(opts)
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
	logger = util.NewExecuteLogger(db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
		logs = append(logs, entry)
		return nil
	}})
	logger.CommandExecute(omnidriver.Command{Text: "SELECT 1;", ByteStart: 14, ByteEnd: 23})
	require.Len(t, logs, 1)
	require.Empty(t, logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, int32(14), logs[0].GetCommandExecute().GetRange().GetStart())
	require.Equal(t, int32(23), logs[0].GetCommandExecute().GetRange().GetEnd())
}
