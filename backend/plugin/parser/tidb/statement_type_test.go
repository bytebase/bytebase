package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementTypesSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_TIDB, "CREATE TABLE t (a INT);\nINSERT INTO t VALUES (1);\nSET @x = 1;")
	require.NoError(t, err)

	got, err := base.GetStatementTypes(storepb.Engine_TIDB, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)

	_, err = GetStatementTypes([]base.AST{stubAST{}})
	require.ErrorContains(t, err, "expected TiDB AST")
}
