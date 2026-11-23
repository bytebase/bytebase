package tsql

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestBackupRestoreDoNotDependOnANTLR(t *testing.T) {
	for _, path := range []string{"backup_restore.go"} {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		source := string(content)
		require.NotContains(t, source, "github.com/antlr4-go/antlr/v4", path)
		require.NotContains(t, source, "github.com/bytebase/parser/tsql", path)
		require.NotContains(t, source, "ParseTSQL(", path)
	}
}

func TestTransformDMLToSelectSeam(t *testing.T) {
	sql := "\nUSE APP;\nUPDATE dbo.users SET name = 'bob' WHERE id = 1;"

	// IsCaseSensitive=false treats USE APP as the same database as sourceDatabase "app".
	ciStmts, err := base.TransformDMLToSelect(
		context.Background(),
		storepb.Engine_MSSQL,
		base.TransformContext{IsCaseSensitive: false},
		sql,
		"app",
		"backup_db",
		"bb_",
	)
	require.NoError(t, err)
	require.Len(t, ciStmts, 1)
	require.Equal(t, &storepb.Position{Line: 3, Column: 0}, ciStmts[0].StartPosition)
	require.Equal(t, int32(3), ciStmts[0].EndPosition.Line)

	// IsCaseSensitive=true rejects USE APP when sourceDatabase is "app".
	_, err = base.TransformDMLToSelect(
		context.Background(),
		storepb.Engine_MSSQL,
		base.TransformContext{IsCaseSensitive: true},
		sql,
		"app",
		"backup_db",
		"bb_",
	)
	require.ErrorContains(t, err, "USE APP")

	// Parse failure is wrapped with transformation context.
	_, err = base.TransformDMLToSelect(context.Background(), storepb.Engine_MSSQL, base.TransformContext{}, "UPDATE ;", "app", "backup_db", "bb_")
	require.ErrorContains(t, err, "failed to prepare transformation")
}
