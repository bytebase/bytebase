package mysql

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

	// Both instance-scoped and project-scoped database resource names work through the registry.
	for _, prefix := range []string{"instances/i1/databases/", "projects/project-a/instances/i1/databases/"} {
		result, err := base.GenerateRestoreSQL(context.Background(), store.Engine_MYSQL, rCtx, "UPDATE test SET a = 1 WHERE c = 1;", &store.PriorBackupDetail_Item{
			SourceTable:   &store.PriorBackupDetail_Item_Table{Database: prefix + "db", Table: "test"},
			TargetTable:   &store.PriorBackupDetail_Item_Table{Database: prefix + "bbarchive", Table: "prefix_1_test"},
			StartPosition: &store.Position{Line: 0, Column: 0},
			EndPosition:   &store.Position{Line: math.MaxInt32, Column: 0},
		})
		require.NoError(t, err, prefix)
		require.Equal(t, "/*\nOriginal SQL:\nUPDATE test SET a = 1 WHERE c = 1;\n*/\nINSERT INTO `db`.`test` (`a`, `b`, `c`) SELECT `a`, `b`, `c` FROM `bbarchive`.`prefix_1_test` ON DUPLICATE KEY UPDATE `a` = VALUES(`a`);", result)
	}

	// Nil backup item guard.
	_, err := GenerateRestoreSQL(context.Background(), rCtx, "DELETE FROM test;", nil)
	require.ErrorContains(t, err, "backup item is nil")
}
