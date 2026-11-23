package oceanbase

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type foreignAST struct{}

func (foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetStatementTypesSeam(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_OCEANBASE, "CREATE TABLE t (a INT);\nCREATE MATERIALIZED VIEW mv AS SELECT 1;\nSET @x = 1;")
	require.NoError(t, err)

	got, err := base.GetStatementTypes(storepb.Engine_OCEANBASE, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_CREATE_VIEW,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)

	_, err = GetStatementTypes([]base.AST{foreignAST{}})
	require.ErrorContains(t, err, "expected AST for OceanBase")
}
