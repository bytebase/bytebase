package drivers

import (
	"context"
	"net"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestCockroachDBConfig(t *testing.T) {
	taskRunUID := int64(7)
	for _, tc := range []struct {
		name   string
		config db.ConnectionConfig
		want   odriver.Config
	}{
		{
			name: "data source database",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Username: "dba", Password: "secret", Host: "crdb.example.com", Port: "26257", Database: "app"},
				Password:   "resolved",
			},
			want: odriver.Config{
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
			want: odriver.Config{
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
			got, err := cockroachdbConfig(tc.config)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCockroachDBConfigMapsTLSOnlyWhenAsked(t *testing.T) {
	got, err := cockroachdbConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{Username: "dba", Host: "crdb.example.com", Port: "26257"}})
	require.NoError(t, err)
	require.Nil(t, got.TLS)

	got, err = cockroachdbConfig(db.ConnectionConfig{DataSource: &storepb.DataSource{
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

func TestCockroachDBConfigRefusesHalfAClientCertificate(t *testing.T) {
	for _, dataSource := range []*storepb.DataSource{
		{Username: "dba", Host: "crdb.example.com", Port: "26257", SslCert: "cert"},
		{Username: "dba", Host: "crdb.example.com", Port: "26257", SslKey: "key"},
	} {
		_, err := cockroachdbConfig(db.ConnectionConfig{DataSource: dataSource})
		require.EqualError(t, err, "ssl-cert and ssl-key must be both set or unset")
	}
}

func TestCockroachDBTaskRunSessionFromResult(t *testing.T) {
	t.Parallel()
	sessionStart := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	queryStart := sessionStart.Add(5 * time.Minute)
	sessionID := "18e9a1f97a49d2ca0000000000000001"

	postgres := cockroachdbSessions(&odriver.Result{
		Rows: [][]odriver.Value{
			{
				{Kind: odriver.String, String: sessionID},
				{Kind: odriver.String, String: "SELECT pg_sleep(20)"},
				{Kind: odriver.String, String: "executing"},
				{Kind: odriver.String, String: "smoke"},
				{Kind: odriver.String, String: "root"},
				{Kind: odriver.String, String: "bytebase-taskrun-7"},
				{Kind: odriver.String, String: "[::1]:44768"},
				{Kind: odriver.TimestampTZ, Time: sessionStart},
				{Kind: odriver.TimestampTZ, Time: queryStart},
			},
		},
	}, "bytebase-taskrun-7")

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
	withoutSessionStart := cockroachdbSessions(&odriver.Result{
		Rows: [][]odriver.Value{
			{
				{Kind: odriver.String, String: missingSessionStartID},
				{Kind: odriver.String, String: "SELECT pg_sleep(20)"},
				{Kind: odriver.String, String: "executing"},
				{Kind: odriver.String, String: "smoke"},
				{Kind: odriver.String, String: "root"},
				{Kind: odriver.String, String: "bytebase-taskrun-8"},
				{Kind: odriver.Null},
				{Kind: odriver.Null},
				{Kind: odriver.TimestampTZ, Time: queryStart},
			},
		},
	}, "bytebase-taskrun-8")

	require.NotNil(t, withoutSessionStart.Session)
	require.Equal(t, missingSessionStartID, withoutSessionStart.Session.Pid)
	require.Nil(t, withoutSessionStart.Session.ClientAddr)
	require.Nil(t, withoutSessionStart.Session.BackendStart)
	require.Equal(t, timestamppb.New(queryStart), withoutSessionStart.Session.QueryStart)
}

func TestCockroachDBExplainFormat(t *testing.T) {
	require.Equal(t, "JSON", util.OmniExplainFormat(v1pb.QueryOption_JSON, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_TEXT, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_XML, "TEXT", v1pb.QueryOption_JSON))
	require.Equal(t, "TEXT", util.OmniExplainFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED, "TEXT", v1pb.QueryOption_JSON))
}

func TestCockroachDBExecuteOptions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		script     string
		options    db.ExecuteOptions
		wantScript string
		wantMode   odriver.TransactionMode
	}{
		{
			name:       "no directive is transactional",
			script:     "SELECT 1;",
			wantScript: "SELECT 1;",
			wantMode:   odriver.Transactional,
		},
		{
			name:       "directive off is auto commit and is stripped",
			script:     "-- txn-mode = off\nSELECT 1;",
			wantScript: "SELECT 1;",
			wantMode:   odriver.AutoCommit,
		},
		{
			name:       "an unknown mode stays transactional and keeps the comment",
			script:     "-- txn-mode = whatever\nSELECT 1;",
			wantScript: "-- txn-mode = whatever\nSELECT 1;",
			wantMode:   odriver.Transactional,
		},
		{
			name:       "create database passes through",
			script:     "CREATE DATABASE app;",
			options:    db.ExecuteOptions{CreateDatabase: true},
			wantScript: "CREATE DATABASE app;",
			wantMode:   odriver.Transactional,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script, options := util.BuildExecuteOptions(tc.script, tc.options, cockroachdbSpec.execute)
			require.Equal(t, tc.wantScript, script)
			require.Equal(t, tc.wantMode, options.Transaction)
			require.Equal(t, tc.options.CreateDatabase, options.CreateDatabase)
			require.NotNil(t, options.Log)
		})
	}
}

// TestCockroachDBExecuteLogger verifies ExecuteLogger bridges the contract's steps onto the
// task run log in the shape the product reads.
func TestCockroachDBExecuteLogger(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	logger := util.NewExecuteLogger(opts)
	logger.CommandExecute(odriver.Command{Text: "SELECT 1;", ByteStart: 14, ByteEnd: 23})
	logger.CommandResponse(2, []int64{1, 1}, nil)
	logger.TransactionControl(odriver.Begin, nil)
	logger.TransactionControl(odriver.Rollback, context.DeadlineExceeded)
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
	logger.CommandExecute(odriver.Command{Text: "SELECT 1;", ByteStart: 14, ByteEnd: 23})
	require.Len(t, logs, 1)
	require.Empty(t, logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, int32(14), logs[0].GetCommandExecute().GetRange().GetStart())
	require.Equal(t, int32(23), logs[0].GetCommandExecute().GetRange().GetEnd())
}

func TestCockroachDBPingThroughSSHTunnelReturnsWhenContextEnds(t *testing.T) {
	t.Parallel()
	sshServer := testcontainer.NewSSHServer(t)
	// The server accepts the connection and never answers the startup message.
	host, port := testcontainer.NewTCPServer(t, func(net.Conn) {})

	background := context.Background()
	driver, err := newDriver(cockroachdbSpec).Open(background, storepb.Engine_COCKROACHDB, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:          host,
			Port:          port,
			Username:      "bytebase",
			SshHost:       sshServer.GetHost(),
			SshPort:       sshServer.GetPort(),
			SshUser:       sshServer.GetUser(),
			SshPrivateKey: sshServer.GetPrivateKey(),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(background) })

	ctx, cancel := context.WithTimeout(background, 500*time.Millisecond)
	defer cancel()
	pinged := make(chan error, 1)
	go func() { pinged <- driver.Ping(ctx) }()

	select {
	case err := <-pinged:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "Ping was still blocked long after its context ended")
	}
}
