package bigquery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type foreignAST struct{}

func (*foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestStatementTypeSeam(t *testing.T) {
	// Multi-statement ParseStatements + GetStatementTypes conversion.
	stmts, err := base.ParseStatements(storepb.Engine_BIGQUERY, "ALTER TABLE users ADD COLUMN status STRING;\nINSERT INTO users (id) VALUES (1);\nCALL ds.proc();")
	require.NoError(t, err)
	require.Len(t, stmts, 3)
	require.Equal(t, int32(1), stmts[0].AST.ASTStartPosition().GetLine())
	require.Equal(t, int32(1), stmts[1].AST.ASTStartPosition().GetLine())

	got, err := GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_ALTER_TABLE,
		storepb.StatementType_INSERT,
		storepb.StatementType_EXECUTE,
	}, got)

	// Unparseable statement survives as UNSPECIFIED with a non-nil AST wrapper
	// so base.ExtractASTs does not drop it.
	stmts, err = base.ParseStatements(storepb.Engine_BIGQUERY, "THIS IS NOT SQL;\nINSERT INTO users (id) VALUES (1);")
	require.NoError(t, err)
	got, err = GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
		storepb.StatementType_INSERT,
	}, got)

	// A lone ';' and a trailing comment hold no statement.
	stmts, err = base.ParseStatements(storepb.Engine_BIGQUERY, "INSERT INTO users (id) VALUES (1);;\n-- done")
	require.NoError(t, err)
	got, err = GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{storepb.StatementType_INSERT}, got)

	// Foreign and nil ASTs map safely to UNSPECIFIED.
	got, err = GetStatementTypes([]base.AST{&foreignAST{}, &base.NodeAST[any]{}})
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)
}

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_BIGQUERY, "SELECT 1;")
	require.NoError(t, err)
	require.Empty(t, diags)
	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_BIGQUERY, "SELECT FROM")
	require.NoError(t, err)
	require.NotEmpty(t, diags)
}
