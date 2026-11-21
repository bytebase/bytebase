package tidb

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGenerateRestoreSQLSeam(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	rCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}

	for _, prefix := range []string{"instances/i1/databases/", "projects/project-a/instances/i1/databases/"} {
		result, err := base.GenerateRestoreSQL(context.Background(), store.Engine_TIDB, rCtx, "DELETE FROM test WHERE b1 = 1;", &store.PriorBackupDetail_Item{
			SourceTable:   &store.PriorBackupDetail_Item_Table{Database: prefix + "db", Table: "test"},
			TargetTable:   &store.PriorBackupDetail_Item_Table{Database: prefix + "bbarchive", Table: "prefix_test"},
			StartPosition: &store.Position{Line: 0, Column: 0},
			EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
		})
		require.NoError(t, err, prefix)
		require.Equal(t, "/*\nOriginal SQL:\nDELETE FROM test WHERE b1 = 1;\n*/\nINSERT INTO `db`.`test` (`a`, `b`, `c`) SELECT `a`, `b`, `c` FROM `bbarchive`.`prefix_test`;", result)
	}
}

func TestGenerateRestoreSQLNilBackupItemGuards(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	rCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}

	cases := []struct {
		name       string
		backupItem *store.PriorBackupDetail_Item
		wantErr    string
	}{
		{
			name:       "nil backupItem",
			backupItem: nil,
			wantErr:    "backup item is nil",
		},
		{
			name: "nil SourceTable",
			backupItem: &store.PriorBackupDetail_Item{
				TargetTable: &store.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/bbarchive", Table: "prefix_1_test"},
			},
			wantErr: "backup item source table is nil",
		},
		{
			name: "nil TargetTable",
			backupItem: &store.PriorBackupDetail_Item{
				SourceTable: &store.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/db", Table: "test"},
			},
			wantErr: "backup item target table is nil",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := GenerateRestoreSQL(context.Background(), rCtx, "DELETE FROM test;", tc.backupItem)
				require.ErrorContains(t, err, tc.wantErr)
			})
		})
	}
}

// TestGenerateRestoreSQLEndPositionExclusive pins that backupItem.EndPosition
// is exclusive: a statement starting at EndPosition belongs to the next backup item.
func TestGenerateRestoreSQLEndPositionExclusive(t *testing.T) {
	const input = "UPDATE test SET a = 1 WHERE c = 1;\nUPDATE test SET b = 2 WHERE c = 2;"

	stmts, err := SplitSQL(input)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(stmts), 2)

	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := GenerateRestoreSQL(context.Background(), base.RestoreContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, input, &store.PriorBackupDetail_Item{
		SourceTable:   &store.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/db", Table: "test"},
		TargetTable:   &store.PriorBackupDetail_Item_Table{Database: "instances/i1/databases/bbarchive", Table: "prefix_1_test"},
		StartPosition: &store.Position{Line: 0, Column: 0},
		EndPosition:   stmts[0].End,
	})

	require.NoError(t, err)
	require.Contains(t, result, "`a` = VALUES(`a`)")
	require.NotContains(t, result, "`b` = VALUES(`b`)")
}
