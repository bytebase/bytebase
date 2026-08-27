package googlesql

import (
	"testing"

	"github.com/bytebase/omni/googlesql/analysis"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var bigQueryConfig = Config{Dialect: analysis.DialectBigQuery, SetStatementIsSelect: true}

func typesOf(t *testing.T, statement string) []storepb.StatementType {
	t.Helper()
	stmts, err := ParseStatements(statement, bigQueryConfig)
	require.NoError(t, err)
	types, err := GetStatementTypes(base.ExtractASTs(stmts))
	require.NoError(t, err)
	return types
}

func TestGetStatementTypes(t *testing.T) {
	tests := []struct {
		statement string
		want      storepb.StatementType
	}{
		// DML.
		{"INSERT INTO `users` (id, name) VALUES (1, 'a');", storepb.StatementType_INSERT},
		{"INSERT INTO ds.users SELECT * FROM ds.staging;", storepb.StatementType_INSERT},
		{"UPDATE users SET name = 'b' WHERE id = 1;", storepb.StatementType_UPDATE},
		{"DELETE FROM users WHERE id = 1;", storepb.StatementType_DELETE},
		{"MERGE INTO users t USING staging s ON t.id = s.id WHEN MATCHED THEN UPDATE SET t.name = s.name;", storepb.StatementType_MERGE},
		{"TRUNCATE TABLE users;", storepb.StatementType_TRUNCATE},
		// DDL - create.
		{"CREATE TABLE users (id INT64, name STRING);", storepb.StatementType_CREATE_TABLE},
		{"CREATE OR REPLACE TABLE users (id INT64);", storepb.StatementType_CREATE_TABLE},
		{"CREATE SNAPSHOT TABLE snap CLONE users;", storepb.StatementType_CREATE_TABLE},
		{"CREATE VIEW v AS SELECT 1;", storepb.StatementType_CREATE_VIEW},
		{"CREATE MATERIALIZED VIEW mv AS SELECT 1;", storepb.StatementType_CREATE_VIEW},
		{"CREATE SCHEMA ds;", storepb.StatementType_CREATE_SCHEMA},
		{"CREATE SEARCH INDEX idx ON users(name);", storepb.StatementType_CREATE_INDEX},
		{"CREATE FUNCTION f(x INT64) AS (x + 1);", storepb.StatementType_CREATE_FUNCTION},
		{"CREATE PROCEDURE p() BEGIN SELECT 1; END;", storepb.StatementType_CREATE_PROCEDURE},
		// DDL - alter. The reported case from BYT-10131 is the first one.
		{"ALTER TABLE `users` ADD COLUMN status STRING;", storepb.StatementType_ALTER_TABLE},
		{"ALTER TABLE users SET OPTIONS(description='x');", storepb.StatementType_ALTER_TABLE},
		{"ALTER VIEW v SET OPTIONS(description='x');", storepb.StatementType_ALTER_VIEW},
		{"ALTER MATERIALIZED VIEW mv SET OPTIONS(enable_refresh=true);", storepb.StatementType_ALTER_VIEW},
		// DDL - drop.
		{"DROP TABLE users;", storepb.StatementType_DROP_TABLE},
		{"DROP VIEW v;", storepb.StatementType_DROP_VIEW},
		{"DROP SCHEMA ds;", storepb.StatementType_DROP_SCHEMA},
		{"DROP MATERIALIZED VIEW mv;", storepb.StatementType_DROP_VIEW},
		{"DROP FUNCTION f;", storepb.StatementType_DROP_FUNCTION},
		{"DROP PROCEDURE p;", storepb.StatementType_DROP_PROCEDURE},
		{"DROP SEARCH INDEX idx ON users;", storepb.StatementType_DROP_INDEX},
		// No enum value; see the deferral on statementType.
		{"ALTER SCHEMA ds SET OPTIONS(description='x');", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"DROP ROW ACCESS POLICY rap ON users;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		{"SELECT * FROM users;", storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.statement, func(t *testing.T) {
			got := typesOf(t, tt.statement)
			require.Len(t, got, 1)
			require.Equal(t, tt.want, got[0])
		})
	}
}

// TestGetStatementTypesAgreesWithClassifier is the acceptance test: omni's
// analysis.ClassifySQL derives DML membership by a different route than the AST
// switch, so a statement it calls DML must map onto a DML enum value. A
// disagreement means an approval rule keyed on statement.sql_type would
// mis-gate that statement, which is the defect BYT-10131 reported.
func TestGetStatementTypesAgreesWithClassifier(t *testing.T) {
	dml := map[storepb.StatementType]bool{
		storepb.StatementType_INSERT:   true,
		storepb.StatementType_UPDATE:   true,
		storepb.StatementType_DELETE:   true,
		storepb.StatementType_MERGE:    true,
		storepb.StatementType_TRUNCATE: true,
	}
	statements := []string{
		"INSERT INTO users (id) VALUES (1);",
		"UPDATE users SET name = 'b' WHERE id = 1;",
		"DELETE FROM users WHERE id = 1;",
		"MERGE INTO users t USING staging s ON t.id = s.id WHEN MATCHED THEN UPDATE SET t.name = s.name;",
		"TRUNCATE TABLE users;",
		"CREATE TABLE users (id INT64);",
		"ALTER TABLE users ADD COLUMN status STRING;",
		"DROP TABLE users;",
		"CREATE VIEW v AS SELECT 1;",
		"SELECT * FROM users;",
	}
	for _, statement := range statements {
		t.Run(statement, func(t *testing.T) {
			got := typesOf(t, statement)
			require.Len(t, got, 1)
			classified := analysis.ClassifySQL(statement, analysis.DialectBigQuery)
			require.Equal(t, classified == analysis.DML, dml[got[0]],
				"classifier says %v, mapping says %v", classified, got[0])
		})
	}
}

func TestParseStatementsMultiStatement(t *testing.T) {
	got := typesOf(t, "ALTER TABLE users ADD COLUMN status STRING;\nINSERT INTO users (id) VALUES (1);")
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_ALTER_TABLE,
		storepb.StatementType_INSERT,
	}, got)
}

// An unparseable statement must survive as UNSPECIFIED rather than vanish.
// base.ExtractASTs drops entries with a nil AST, so dropping it here would hide
// the statement from approval rules entirely and reopen the BYT-10131 skip
// through a parse gap.
func TestParseStatementsReportsAnUnparseableStatement(t *testing.T) {
	got := typesOf(t, "THIS IS NOT SQL;\nINSERT INTO users (id) VALUES (1);")
	require.Equal(t, []storepb.StatementType{
		storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
		storepb.StatementType_INSERT,
	}, got)
}
