package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type foreignAST struct{}

func (foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetStatementTypesSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_MYSQL, "CREATE TABLE t (a INT);\nINSERT INTO t VALUES (1);\nSET @x = 1;")
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)

	got, err := base.GetStatementTypes(storepb.Engine_MYSQL, asts)
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)

	// GetStatementTypesWithPositions preserves order, computes 1-based end line,
	// and keeps STATEMENT_TYPE_UNSPECIFIED so SDL gating fails closed.
	multiStmts, err := base.ParseStatements(storepb.Engine_MYSQL, "CREATE TABLE t (\n  a INT\n);\nSET @x = 1;")
	require.NoError(t, err)
	withPos, err := GetStatementTypesWithPositions(base.ExtractASTs(multiStmts))
	require.NoError(t, err)
	require.Len(t, withPos, 2)
	require.Equal(t, storepb.StatementType_CREATE_TABLE, withPos[0].Type)
	require.Equal(t, 3, withPos[0].Line)
	require.Equal(t, storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, withPos[1].Type)
	require.Equal(t, 4, withPos[1].Line)

	// Foreign AST rejection.
	_, err = GetStatementTypes([]base.AST{foreignAST{}})
	require.ErrorContains(t, err, "expected AST for MySQL")
	_, err = GetStatementTypesWithPositions([]base.AST{foreignAST{}})
	require.ErrorContains(t, err, "expected AST for MySQL")
}
