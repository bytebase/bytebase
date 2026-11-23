package googlesql

import (
	"testing"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var bigQueryConfig = Config{Dialect: analysis.DialectBigQuery, SetStatementIsSelect: true}

type foreignAST struct{}

func (*foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestStatementTypeSeam(t *testing.T) {
	// Multi-statement ParseStatements + GetStatementTypes conversion.
	stmts, err := ParseStatements("ALTER TABLE users ADD COLUMN status STRING;\nINSERT INTO users (id) VALUES (1);\nCALL ds.proc();", bigQueryConfig)
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

	// Unparseable statement survives as UNSPECIFIED with a non-nil *AST wrapper
	// so base.ExtractASTs does not drop it.
	stmts, err = ParseStatements("THIS IS NOT SQL;\nINSERT INTO users (id) VALUES (1);", bigQueryConfig)
	require.NoError(t, err)
	got, err = GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
		storepb.StatementType_INSERT,
	}, got)

	// A lone ';' and a trailing comment hold no statement.
	stmts, err = ParseStatements("INSERT INTO users (id) VALUES (1);;\n-- done", bigQueryConfig)
	require.NoError(t, err)
	got, err = GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{storepb.StatementType_INSERT}, got)

	// Foreign and nil ASTs map safely to UNSPECIFIED.
	got, err = GetStatementTypes([]base.AST{&foreignAST{}, &AST{}})
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
	}, got)

	// Spanner dialect config.
	stmts, err = ParseStatements("CALL cancel_query('12345');", Config{Dialect: analysis.DialectSpanner})
	require.NoError(t, err)
	got, err = GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Equal(t, []storepb.StatementType{storepb.StatementType_EXECUTE}, got)
}

func TestDiagnoseSeam(t *testing.T) {
	require.Empty(t, Diagnose("SELECT 1;"))
	diags := Diagnose("SELECT FROM")
	require.NotEmpty(t, diags)
	require.True(t, isValidButUnimplementedStub("IMPORT MODULE statement parsing is not yet supported"))
	require.False(t, isValidButUnimplementedStub("unknown or unsupported statement starting with FOO"))
}
