package mariadb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type foreignAST struct{}

func (foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestGetStatementTypesSeam(t *testing.T) {
	// Registry dispatch including MariaDB-only DDL syntax.
	stmts, err := base.ParseStatements(storepb.Engine_MARIADB, "CREATE TABLE t (a INT);\nCREATE OR REPLACE TABLE u (a INT);\nALTER TABLE t DROP COLUMN IF EXISTS b;\nSET @x = 1;")
	require.NoError(t, err)

	got, err := base.GetStatementTypes(storepb.Engine_MARIADB, base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.ElementsMatch(t, []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_ALTER_TABLE,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)

	_, err = GetStatementTypes([]base.AST{foreignAST{}})
	require.ErrorContains(t, err, "expected AST for MariaDB")
}
