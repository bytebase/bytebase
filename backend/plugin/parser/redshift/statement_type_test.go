package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type foreignAST struct{}

func (*foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetStatementTypesSeam(t *testing.T) {
	statement := "CREATE TABLE t1 (id INT);\nDROP TABLE t2;\nINSERT INTO t1 VALUES (1);\nCALL archive_orders(30);"
	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)

	got, err := base.GetStatementTypes(storepb.Engine_REDSHIFT, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_DROP_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_EXECUTE,
	}, got)

	withPos, err := getStatementTypesWithPosition(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Len(t, withPos, 4)
	require.Equal(t, 1, withPos[0].Line)
	require.Equal(t, "CREATE TABLE t1 (id INT);", withPos[0].Text)
	require.Equal(t, 2, withPos[1].Line)
	require.Equal(t, 3, withPos[2].Line)
	require.Equal(t, 4, withPos[3].Line)

	_, err = GetStatementTypes([]base.AST{&foreignAST{}})
	require.ErrorContains(t, err, "expected Redshift omni AST")
}
