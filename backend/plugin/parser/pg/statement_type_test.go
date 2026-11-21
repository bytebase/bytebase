package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type wrongAST struct{}

func (wrongAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetStatementTypesSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, "CREATE TABLE t1 (\n  id INT\n);\nDROP TABLE t2;\nINSERT INTO t1 VALUES (1);")
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)

	withPos, err := GetStatementTypes(asts)
	require.NoError(t, err)
	require.Len(t, withPos, 3)
	require.Equal(t, storepb.StatementType_CREATE_TABLE, withPos[0].Type)
	require.Equal(t, 3, withPos[0].Line)
	require.Contains(t, withPos[0].Text, "CREATE TABLE t1")
	require.Equal(t, storepb.StatementType_DROP_TABLE, withPos[1].Type)
	require.Equal(t, 4, withPos[1].Line)
	require.Equal(t, storepb.StatementType_INSERT, withPos[2].Type)
	require.Equal(t, 5, withPos[2].Line)

	regTypes, err := base.GetStatementTypes(storepb.Engine_POSTGRES, asts)
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_DROP_TABLE,
		storepb.StatementType_INSERT,
	}, regTypes)

	empty, err := GetStatementTypes(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = GetStatementTypes([]base.AST{wrongAST{}})
	require.ErrorContains(t, err, "expected AST for PostgreSQL")
}
