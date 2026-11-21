package plsql

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestTransformDMLToSelectSeam(t *testing.T) {
	longTable := strings.Repeat("a", 40)
	sql := "\nUPDATE " + longTable + " SET c1 = 1 WHERE id = 2;"

	// Oracle 11g truncates backup table names to 30 bytes via tCtx.Version.(*Version).
	stmts11g, err := base.TransformDMLToSelect(
		context.Background(),
		storepb.Engine_ORACLE,
		base.TransformContext{Version: &Version{First: 11, Second: 2}},
		sql,
		"HR",
		"BACKUP",
		"bb_",
	)
	require.NoError(t, err)
	require.Len(t, stmts11g, 1)
	require.LessOrEqual(t, len(stmts11g[0].TargetTableName), 30)
	// StartPosition uses 0-based columns.
	require.Equal(t, &storepb.Position{Line: 2, Column: 0}, stmts11g[0].StartPosition)
	require.Equal(t, int32(2), stmts11g[0].EndPosition.Line)
	require.Greater(t, stmts11g[0].EndPosition.Column, int32(0))

	// Oracle 19c allows 128-byte identifiers.
	stmts19c, err := base.TransformDMLToSelect(
		context.Background(),
		storepb.Engine_ORACLE,
		base.TransformContext{Version: &Version{First: 19, Second: 0}},
		sql,
		"HR",
		"BACKUP",
		"bb_",
	)
	require.NoError(t, err)
	require.Len(t, stmts19c, 1)
	require.Greater(t, len(stmts19c[0].TargetTableName), 30)

	// Invalid SQL propagates error.
	_, err = base.TransformDMLToSelect(context.Background(), storepb.Engine_ORACLE, base.TransformContext{}, "UPDATE ;", "HR", "BACKUP", "bb_")
	require.Error(t, err)
}
