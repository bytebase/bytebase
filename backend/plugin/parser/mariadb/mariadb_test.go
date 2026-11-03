package mariadb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseStatements(t *testing.T) {
	statement := "SELECT 1; SELECT 2;"

	statements, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)

	require.Len(t, statements, 2)

	// Check first statement
	require.Equal(t, "SELECT 1;", statements[0].Text)
	require.False(t, statements[0].Empty)
	require.NotNil(t, statements[0].AST)
	require.NotNil(t, statements[0].Start)

	// Check second statement
	require.Contains(t, statements[1].Text, "SELECT 2")
	require.False(t, statements[1].Empty)
	require.NotNil(t, statements[1].AST)
}

// TestParseStatementsReadsMariaDBOnlySyntax pins the point of this package: a
// statement omni's MySQL parser rejects reaches every MariaDB code path as an
// ordinary node, rather than an AST those paths skip.
func TestParseStatementsReadsMariaDBOnlySyntax(t *testing.T) {
	for _, statement := range []string{
		"CREATE OR REPLACE TABLE u (a INT);",
		"ALTER TABLE t DROP COLUMN IF EXISTS b;",
		"CREATE SEQUENCE s START WITH 1;",
		"CREATE TABLE t (a INT) WITH SYSTEM VERSIONING;",
		"DELETE FROM t WHERE a = 1 RETURNING a;",
		"SELECT * FROM t FOR SYSTEM_TIME AS OF NOW();",
	} {
		statements, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
		require.NoError(t, err, statement)
		require.Len(t, statements, 1, statement)

		omniAST, ok := statements[0].AST.(*OmniAST)
		require.True(t, ok, statement)
		require.NotNil(t, omniAST.Node, statement)
		require.Equal(t, statements[0].Text, omniAST.Text, statement)
		require.Equal(t, statements[0].Start, omniAST.StartPosition, statement)
	}
}

func TestParseStatementsReportsSyntaxErrorPosition(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      *storepb.Position
	}{
		// The error is at the ')' after DEFAULT.
		{"SELECT 1;\nCREATE OR REPLACE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 2, Column: 41}},
		// A statement sharing a line with an earlier one keeps that line's column.
		{"SELECT 1; CREATE OR REPLACE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 1, Column: 51}},
		{"SELECT 1; CREATE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 1, Column: 40}},
		{"SELECT 1;\nSELECT 2; CREATE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 2, Column: 40}},
	} {
		_, err := base.ParseStatements(storepb.Engine_MARIADB, tc.statement)

		var syntaxErr *base.SyntaxError
		require.ErrorAs(t, err, &syntaxErr, tc.statement)
		require.Equal(t, tc.want, syntaxErr.Position, tc.statement)
	}
}
