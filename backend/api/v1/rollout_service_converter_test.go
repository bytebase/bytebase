package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertToTaskRunLogEntries_PriorBackupWithDatabaseSync(t *testing.T) {
	baseTime := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	logs := []*store.TaskRunLog{
		{
			T: baseTime,
			Payload: &storepb.TaskRunLog{
				Type:             storepb.TaskRunLog_PRIOR_BACKUP_START,
				PriorBackupStart: &storepb.TaskRunLog_PriorBackupStart{},
			},
		},
		{
			T: baseTime.Add(time.Second),
			Payload: &storepb.TaskRunLog{
				Type:              storepb.TaskRunLog_DATABASE_SYNC_START,
				DatabaseSyncStart: &storepb.TaskRunLog_DatabaseSyncStart{},
			},
		},
		{
			T: baseTime.Add(2 * time.Second),
			Payload: &storepb.TaskRunLog{
				Type: storepb.TaskRunLog_DATABASE_SYNC_END,
				DatabaseSyncEnd: &storepb.TaskRunLog_DatabaseSyncEnd{
					Error: "sync failed",
				},
			},
		},
		{
			T: baseTime.Add(3 * time.Second),
			Payload: &storepb.TaskRunLog{
				Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
				PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
					PriorBackupDetail: &storepb.PriorBackupDetail{
						Items: []*storepb.PriorBackupDetail_Item{
							{
								SourceTable: &storepb.PriorBackupDetail_Item_Table{
									Database: "instances/prod/databases/app",
									Schema:   "public",
									Table:    "users",
								},
								TargetTable: &storepb.PriorBackupDetail_Item_Table{
									Database: "instances/prod/databases/backup",
									Table:    "users_backup",
								},
							},
						},
					},
				},
			},
		},
	}

	entries := convertToTaskRunLogEntries(logs)

	require.Len(t, entries, 2)
	require.Equal(t, v1pb.TaskRunLogEntry_PRIOR_BACKUP, entries[0].Type)
	require.Equal(t, baseTime, entries[0].GetPriorBackup().GetStartTime().AsTime())
	require.Equal(t, baseTime.Add(3*time.Second), entries[0].GetPriorBackup().GetEndTime().AsTime())
	require.Len(t, entries[0].GetPriorBackup().GetPriorBackupDetail().GetItems(), 1)
	require.Equal(t, v1pb.TaskRunLogEntry_DATABASE_SYNC, entries[1].Type)
	require.Equal(t, "sync failed", entries[1].GetDatabaseSync().GetError())
}
