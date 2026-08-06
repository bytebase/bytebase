package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertToTaskRunStatus(t *testing.T) {
	tests := []struct {
		storeStatus storepb.TaskRun_Status
		want        v1pb.TaskRun_Status
	}{
		{storeStatus: storepb.TaskRun_STATUS_UNSPECIFIED, want: v1pb.TaskRun_STATUS_UNSPECIFIED},
		{storeStatus: storepb.TaskRun_PENDING, want: v1pb.TaskRun_PENDING},
		{storeStatus: storepb.TaskRun_AVAILABLE, want: v1pb.TaskRun_AVAILABLE},
		{storeStatus: storepb.TaskRun_RUNNING, want: v1pb.TaskRun_RUNNING},
		{storeStatus: storepb.TaskRun_DONE, want: v1pb.TaskRun_DONE},
		{storeStatus: storepb.TaskRun_FAILED, want: v1pb.TaskRun_FAILED},
		{storeStatus: storepb.TaskRun_CANCELED, want: v1pb.TaskRun_CANCELED},
	}

	for _, test := range tests {
		t.Run(test.storeStatus.String(), func(t *testing.T) {
			require.Equal(t, test.want, convertToTaskRunStatus(test.storeStatus))
		})
	}
}

func TestConvertToTaskRunLogEntries_GhostMigration(t *testing.T) {
	start := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Second)

	entries := convertToTaskRunLogEntries([]*store.TaskRunLog{
		{
			T: start,
			Payload: &storepb.TaskRunLog{
				Type:                storepb.TaskRunLog_GHOST_MIGRATION_START,
				ReplicaId:           "replica-a",
				GhostMigrationStart: &storepb.TaskRunLog_GhostMigrationStart{},
			},
		},
		{
			T: end,
			Payload: &storepb.TaskRunLog{
				Type:      storepb.TaskRunLog_GHOST_MIGRATION_END,
				ReplicaId: "replica-a",
				GhostMigrationEnd: &storepb.TaskRunLog_GhostMigrationEnd{
					Error: "copy failed",
				},
			},
		},
	})

	require.Len(t, entries, 1)
	entry := entries[0]
	require.Equal(t, v1pb.TaskRunLogEntry_GHOST_MIGRATION, entry.Type)
	require.Equal(t, "replica-a", entry.ReplicaId)
	require.NotNil(t, entry.GhostMigration)
	require.Nil(t, entry.GhostMigration.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name("statement")))
	require.Equal(t, start.Unix(), entry.GhostMigration.StartTime.AsTime().Unix())
	require.Equal(t, end.Unix(), entry.GhostMigration.EndTime.AsTime().Unix())
	require.Equal(t, "copy failed", entry.GhostMigration.Error)
}

// task_run_log rows can share a created_at microsecond and carry no per-row
// sequence, so end/response events may not directly follow their start entry.
func TestConvertToTaskRunLogEntries_TieOrderPairing(t *testing.T) {
	at := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)
	execute := func(statement string) *store.TaskRunLog {
		return &store.TaskRunLog{T: at, Payload: &storepb.TaskRunLog{
			Type:           storepb.TaskRunLog_COMMAND_EXECUTE,
			CommandExecute: &storepb.TaskRunLog_CommandExecute{Statement: statement},
		}}
	}
	response := func(affectedRows int64) *store.TaskRunLog {
		return &store.TaskRunLog{T: at, Payload: &storepb.TaskRunLog{
			Type:            storepb.TaskRunLog_COMMAND_RESPONSE,
			CommandResponse: &storepb.TaskRunLog_CommandResponse{AffectedRows: affectedRows},
		}}
	}

	t.Run("response attaches across an interleaved entry", func(t *testing.T) {
		entries := convertToTaskRunLogEntries([]*store.TaskRunLog{
			execute("SELECT 1"),
			{T: at, Payload: &storepb.TaskRunLog{
				Type:               storepb.TaskRunLog_TRANSACTION_CONTROL,
				TransactionControl: &storepb.TaskRunLog_TransactionControl{Type: storepb.TaskRunLog_TransactionControl_COMMIT},
			}},
			response(7),
		})

		require.Len(t, entries, 2)
		require.Equal(t, v1pb.TaskRunLogEntry_COMMAND_EXECUTE, entries[0].Type)
		require.NotNil(t, entries[0].CommandExecute.Response)
		require.Equal(t, int64(7), entries[0].CommandExecute.Response.AffectedRows)
	})

	t.Run("crossed responses pair with the nearest unpaired execute", func(t *testing.T) {
		entries := convertToTaskRunLogEntries([]*store.TaskRunLog{
			execute("SELECT 1"),
			execute("SELECT 2"),
			response(1),
			response(2),
		})

		require.Len(t, entries, 2)
		require.NotNil(t, entries[0].CommandExecute.Response)
		require.NotNil(t, entries[1].CommandExecute.Response)
		require.Equal(t, int64(1), entries[1].CommandExecute.Response.AffectedRows)
		require.Equal(t, int64(2), entries[0].CommandExecute.Response.AffectedRows)
	})

	t.Run("orphan response is dropped", func(t *testing.T) {
		entries := convertToTaskRunLogEntries([]*store.TaskRunLog{response(1)})
		require.Empty(t, entries)
	})

	t.Run("schema dump end attaches across an interleaved entry", func(t *testing.T) {
		entries := convertToTaskRunLogEntries([]*store.TaskRunLog{
			{T: at, Payload: &storepb.TaskRunLog{
				Type:            storepb.TaskRunLog_SCHEMA_DUMP_START,
				SchemaDumpStart: &storepb.TaskRunLog_SchemaDumpStart{},
			}},
			execute("SELECT 1"),
			{T: at.Add(time.Second), Payload: &storepb.TaskRunLog{
				Type:          storepb.TaskRunLog_SCHEMA_DUMP_END,
				SchemaDumpEnd: &storepb.TaskRunLog_SchemaDumpEnd{Error: "boom"},
			}},
		})

		require.Len(t, entries, 2)
		require.Equal(t, v1pb.TaskRunLogEntry_SCHEMA_DUMP, entries[0].Type)
		require.NotNil(t, entries[0].SchemaDump.EndTime)
		require.Equal(t, "boom", entries[0].SchemaDump.Error)
	})
}
