package db

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestExecuteOptionsLog(t *testing.T) {
	// Nil receiver and nil callback are no-ops.
	var nilOpts *ExecuteOptions
	nilOpts.LogComputeDiffStart()
	nilOpts.LogCommandExecute(&storepb.Range{Start: 0, End: 5}, "SELECT 1")
	nilOpts.LogRetryInfo(errors.New("lock timeout"), 1)
	(&ExecuteOptions{}).LogDatabaseSyncEnd("boom")

	var entries []*storepb.TaskRunLog
	opts := &ExecuteOptions{
		LogCommandStatement: true,
		MaximumRetries:      3,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			entries = append(entries, entry)
			return nil
		},
	}

	opts.LogComputeDiffStart()
	opts.LogComputeDiffEnd("diff err")
	opts.LogDatabaseSyncStart()
	opts.LogDatabaseSyncEnd("sync err")
	opts.LogCommandExecute(&storepb.Range{Start: 0, End: 8}, "SELECT 1")
	opts.LogCommandStatement = false
	opts.LogCommandExecute(&storepb.Range{Start: 0, End: 8}, "SELECT 1")
	opts.LogCommandResponse(2, []int64{1, 1}, "")
	opts.LogGhostMigrationStart()
	opts.LogGhostMigrationEnd("ghost err")
	opts.LogRetryInfo(errors.New("deadlock"), 2)
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")

	require.Len(t, entries, 11)
	require.Equal(t, storepb.TaskRunLog_COMPUTE_DIFF_START, entries[0].Type)
	require.Equal(t, "diff err", entries[1].GetComputeDiffEnd().GetError())
	require.Equal(t, storepb.TaskRunLog_DATABASE_SYNC_START, entries[2].Type)
	require.Equal(t, "sync err", entries[3].GetDatabaseSyncEnd().GetError())
	require.Equal(t, "SELECT 1", entries[4].GetCommandExecute().GetStatement())
	require.Nil(t, entries[4].GetCommandExecute().GetRange())
	require.Empty(t, entries[5].GetCommandExecute().GetStatement())
	require.Equal(t, int32(8), entries[5].GetCommandExecute().GetRange().GetEnd())
	require.Equal(t, int64(2), entries[6].GetCommandResponse().GetAffectedRows())
	require.Equal(t, storepb.TaskRunLog_GHOST_MIGRATION_START, entries[7].Type)
	require.Equal(t, "ghost err", entries[8].GetGhostMigrationEnd().GetError())
	require.Equal(t, "deadlock", entries[9].GetRetryInfo().GetError())
	require.Equal(t, int32(2), entries[9].GetRetryInfo().GetRetryCount())
	require.Equal(t, int32(3), entries[9].GetRetryInfo().GetMaximumRetries())
	require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT, entries[10].GetTransactionControl().GetType())
}
