package oceanbase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestExecuteLogsCommands pins the change-script log the task run renders: one
// command line and response per statement, in order, between the transaction
// control steps, with the statement text and byte range the splitter produced.
func TestExecuteLogsCommands(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "execute_logs"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw, "CREATE TABLE t (id INT PRIMARY KEY, v INT)")

	script := "INSERT INTO t VALUES (1, 1);\nINSERT INTO t VALUES (2, 1);"
	var logs []*storepb.TaskRunLog
	rows, err := driver.Execute(ctx, script, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)

	var sequence []string
	for _, entry := range logs {
		switch entry.Type {
		case storepb.TaskRunLog_TRANSACTION_CONTROL:
			sequence = append(sequence, "txn:"+entry.TransactionControl.Type.String())
		case storepb.TaskRunLog_COMMAND_EXECUTE:
			sequence = append(sequence, "exec:"+entry.CommandExecute.Statement)
		case storepb.TaskRunLog_COMMAND_RESPONSE:
			require.Equal(t, int64(1), entry.CommandResponse.AffectedRows)
			require.Equal(t, []int64{1}, entry.CommandResponse.AllAffectedRows)
			require.Empty(t, entry.CommandResponse.Error)
			sequence = append(sequence, "response")
		default:
		}
	}
	require.Equal(t, []string{
		"txn:BEGIN",
		"exec:INSERT INTO t VALUES (1, 1);",
		"response",
		"exec:\nINSERT INTO t VALUES (2, 1);",
		"response",
		"txn:COMMIT",
	}, sequence)

	// Without the statement text the task run logs the command's range, which
	// covers the statement and its terminating semicolon, as before.
	logs = nil
	_, err = driver.Execute(ctx, "INSERT INTO t VALUES (4, 1);\nINSERT INTO t VALUES (5, 1);", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	var executes []*storepb.TaskRunLog_CommandExecute
	for _, entry := range logs {
		if entry.Type == storepb.TaskRunLog_COMMAND_EXECUTE {
			executes = append(executes, entry.CommandExecute)
		}
	}
	require.Len(t, executes, 2)
	require.Equal(t, &storepb.Range{Start: 0, End: 28}, executes[0].Range)
	require.Equal(t, &storepb.Range{Start: 28, End: 57}, executes[1].Range)

	// An auto-commit script logs no transaction control, as before.
	logs = nil
	rows, err = driver.Execute(ctx, "-- txn-mode = off\nINSERT INTO t VALUES (3, 1);", db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	for _, entry := range logs {
		require.NotEqual(t, storepb.TaskRunLog_TRANSACTION_CONTROL, entry.Type)
	}
}
