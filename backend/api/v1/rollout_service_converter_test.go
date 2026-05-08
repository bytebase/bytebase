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
