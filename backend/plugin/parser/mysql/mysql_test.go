package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

func TestParseMySQLStatements(t *testing.T) {
	statement := "SELECT 1; SELECT 2;"

	statements, err := base.ParseStatements(storepb.Engine_MYSQL, statement)
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

func TestParseMariaDBStatementsFallsBackToMariaDBParser(t *testing.T) {
	statement := "CREATE TABLE t (a INT);\nCREATE OR REPLACE TABLE u (a INT);\nALTER TABLE t DROP COLUMN IF EXISTS b;"

	statements, err := base.ParseStatements(storepb.Engine_MARIADB, statement)
	require.NoError(t, err)

	require.Len(t, statements, 3)
	require.IsType(t, &OmniAST{}, statements[0].AST)
	for _, stmt := range statements[1:] {
		mariaDBAST, ok := stmt.AST.(*mariadb.OmniAST)
		require.True(t, ok, stmt.Text)
		require.NotNil(t, mariaDBAST.Node)
		require.Equal(t, stmt.Text, mariaDBAST.Text)
		require.Equal(t, stmt.Start, mariaDBAST.StartPosition)
	}
}

func TestParseMariaDBStatementsReportsMariaDBSyntaxError(t *testing.T) {
	// omni's MySQL parser would stop at TABLE, column 19.
	_, err := base.ParseStatements(storepb.Engine_MARIADB, "SELECT 1;\nCREATE OR REPLACE TABLE t (a INT DEFAULT);")

	var syntaxErr *base.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
	require.Equal(t, &storepb.Position{Line: 2, Column: 41}, syntaxErr.Position)
}

func TestParseStatementsSyntaxErrorColumnOnSharedLine(t *testing.T) {
	for _, tc := range []struct {
		engine    storepb.Engine
		statement string
		want      *storepb.Position
	}{
		// The error is at the ')' after DEFAULT, column 51 of the line.
		{storepb.Engine_MARIADB, "SELECT 1; CREATE OR REPLACE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 1, Column: 51}},
		{storepb.Engine_MYSQL, "SELECT 1; CREATE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 1, Column: 40}},
		{storepb.Engine_MYSQL, "SELECT 1;\nSELECT 2; CREATE TABLE t (a INT DEFAULT);", &storepb.Position{Line: 2, Column: 40}},
	} {
		_, err := base.ParseStatements(tc.engine, tc.statement)

		var syntaxErr *base.SyntaxError
		require.ErrorAs(t, err, &syntaxErr, tc.statement)
		require.Equal(t, tc.want, syntaxErr.Position, tc.statement)
	}
}

func TestParseMySQLStatementsRejectsMariaDBSyntax(t *testing.T) {
	for _, engine := range []storepb.Engine{storepb.Engine_MYSQL, storepb.Engine_OCEANBASE} {
		_, err := base.ParseStatements(engine, "CREATE OR REPLACE TABLE t (a INT);")

		var syntaxErr *base.SyntaxError
		require.ErrorAs(t, err, &syntaxErr, engine.String())
	}
}
