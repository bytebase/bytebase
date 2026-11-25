package cockroachdb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestParseSeam(t *testing.T) {
	// Registry dispatch combines PG splitter positions with CockroachDB ASTs.
	stmts, err := base.ParseStatements(storepb.Engine_COCKROACHDB, "CREATE TABLE t (id INT); ; UPSERT INTO t VALUES (1);")
	require.NoError(t, err)
	asts := base.ExtractASTs(stmts)
	require.Len(t, asts, 2)
	for _, a := range asts {
		_, ok := a.(*AST)
		require.True(t, ok)
		require.Equal(t, int32(1), a.ASTStartPosition().Line)
	}

	_, err = base.ParseStatements(storepb.Engine_COCKROACHDB, "SELECT FROM")
	require.Error(t, err)
}

func TestSplitSeam(t *testing.T) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_COCKROACHDB, "SELECT 1;\nCREATE FUNCTION f() RETURNS INT LANGUAGE SQL BEGIN ATOMIC SELECT 1; SELECT 2; END;")
	require.NoError(t, err)
	require.Len(t, stmts, 2)
	require.Equal(t, "\nCREATE FUNCTION f() RETURNS INT LANGUAGE SQL BEGIN ATOMIC SELECT 1; SELECT 2; END;", stmts[1].Text)
}

func TestDiagnoseSeam(t *testing.T) {
	diags, err := base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_COCKROACHDB, "SELECT 1;")
	require.NoError(t, err)
	require.Empty(t, diags)

	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_COCKROACHDB, "SELECT 1;\nSELECT * FROM")
	require.NoError(t, err)
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)

	// CockroachDB-only syntax the PostgreSQL grammar rejects.
	diags, err = base.Diagnose(context.Background(), base.DiagnoseContext{}, storepb.Engine_COCKROACHDB, "UPSERT INTO t VALUES (1);\nSELECT * FROM t@idx AS OF SYSTEM TIME '-10s';")
	require.NoError(t, err)
	require.Empty(t, diags)
}

func TestCompletionSeam(t *testing.T) {
	cCtx := base.CompletionContext{
		Scene:           base.SceneTypeAll,
		DefaultDatabase: "db",
		Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", databaseMetadata(), nil
		},
	}
	got, err := base.Completion(context.Background(), storepb.Engine_COCKROACHDB, cCtx, "SELECT * FROM ", 1, len("SELECT * FROM "))
	require.NoError(t, err)
	require.Contains(t, got, base.Candidate{Type: base.CandidateTypeTable, Text: "t"})
}

func TestValidateSQLForEditorSeam(t *testing.T) {
	valid, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_COCKROACHDB, "SELECT * FROM t;")
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, allQuery)

	valid, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_COCKROACHDB, "SHOW RANGES FROM TABLE t; SELECT * FROM t AS OF SYSTEM TIME '-10s';")
	require.NoError(t, err)
	require.True(t, valid)
	require.True(t, allQuery)

	valid, allQuery, err = base.ValidateSQLForEditor(storepb.Engine_COCKROACHDB, "SELECT * FROM [UPSERT INTO t VALUES (1) RETURNING *];")
	require.NoError(t, err)
	require.False(t, valid)
	require.False(t, allQuery)
}

func TestExplainStatementSeam(t *testing.T) {
	for _, tc := range []struct {
		statement string
		format    base.ExplainFormat
		want      string
	}{
		{"SELECT 1", base.ExplainFormatDefault, "EXPLAIN SELECT 1"},
		{"SELECT 1", base.ExplainFormatJSON, "EXPLAIN (DISTSQL) SELECT 1"},
		{"EXPLAIN SELECT 1", base.ExplainFormatJSON, "EXPLAIN (DISTSQL) SELECT 1"},
		// CockroachDB-only syntax the PostgreSQL grammar rejects.
		{"SHOW COMMIT TIMESTAMP", base.ExplainFormatDefault, "EXPLAIN SHOW COMMIT TIMESTAMP"},
	} {
		got, err := base.ExplainStatement(storepb.Engine_COCKROACHDB, tc.statement, tc.format)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}

	_, err := base.ExplainStatement(storepb.Engine_COCKROACHDB, "EXPLAIN ANALYZE SELECT 1", base.ExplainFormatJSON)
	require.ErrorIs(t, err, base.ErrExplainAnalyzeExecutes)
}

func TestStatementWithResultLimitSeam(t *testing.T) {
	for statement, want := range map[string]string{
		"SELECT * FROM t LIMIT 100": "SELECT * FROM t LIMIT 10",
		// A CTE wrapper would move AS OF SYSTEM TIME off the top-level statement.
		"SELECT * FROM t AS OF SYSTEM TIME '-10s'": "SELECT * FROM t AS OF SYSTEM TIME '-10s' LIMIT 10",
		"SHOW RANGES FROM TABLE t":                 "SHOW RANGES FROM TABLE t",
	} {
		require.Equal(t, want, base.StatementWithResultLimit(storepb.Engine_COCKROACHDB, statement, 10, ""))
	}
}

func TestGetQuerySpanSeam(t *testing.T) {
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", databaseMetadata(), nil
		},
	}
	// A view's column also names the table column behind it.
	spans, err := base.GetQuerySpan(context.Background(), gCtx, storepb.Engine_COCKROACHDB, []base.Statement{{Text: "SELECT ssn FROM v"}}, "db", "", false)
	require.NoError(t, err)
	require.Equal(t, base.SourceColumnSet{
		{Database: "db", Schema: "public", Table: "v", Column: "ssn"}: true,
		{Database: "db", Schema: "public", Table: "t", Column: "ssn"}: true,
	}, spans[0].Results[0].SourceColumns)
}

func databaseMetadata() *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "ssn", Type: "STRING"}}}},
			Views:  []*metadatapb.ViewMetadata{{Name: "v", Definition: "SELECT ssn FROM public.t"}},
		}},
	}, nil, nil, storepb.Engine_COCKROACHDB, true)
}
