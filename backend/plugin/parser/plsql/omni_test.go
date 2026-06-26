package plsql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/oracle/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParsePLSQLOmni(t *testing.T) {
	list, err := ParsePLSQLOmni("SELECT * FROM T; INSERT INTO T VALUES (1);")
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 2)

	first, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, first.Stmt)

	second, ok := list.Items[1].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.InsertStmt{}, second.Stmt)
}

func TestParsePLSQLOmniSplitsSlashTerminatedPLSQLScript(t *testing.T) {
	list, err := ParsePLSQLOmni(`
CREATE TABLE AUDIT_LOG (
  LOG_ID NUMBER PRIMARY KEY
);

CREATE OR REPLACE TRIGGER audit_log_trigger
BEFORE INSERT ON AUDIT_LOG
FOR EACH ROW
BEGIN
  NULL;
END;
/
`)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 2)

	first, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.CreateTableStmt{}, first.Stmt)

	second, ok := list.Items[1].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.CreateTriggerStmt{}, second.Stmt)
}

func TestParsePLSQLOmniSkipsSQLPlusCommands(t *testing.T) {
	list, err := ParsePLSQLOmni(`
SET DEFINE OFF
PROMPT setup

CREATE TABLE AUDIT_LOG (
  LOG_ID NUMBER PRIMARY KEY
);

SPOOL out.log
CREATE OR REPLACE TRIGGER audit_log_trigger
BEFORE INSERT ON AUDIT_LOG
FOR EACH ROW
BEGIN
  NULL;
END;
/
SPOOL OFF
`)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 2)

	first, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.CreateTableStmt{}, first.Stmt)

	second, ok := list.Items[1].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.CreateTriggerStmt{}, second.Stmt)
}

func TestParsePLSQLOmniKeepsRemarkColumnInCreateTable(t *testing.T) {
	list, err := ParsePLSQLOmni(`
CREATE TABLE parser_regression_remark_column (
  id                  NUMBER NOT NULL
      CONSTRAINT parser_regression_remark_column_pk
          PRIMARY KEY,
  display_name        VARCHAR2(100),
  source              VARCHAR2(100),
  remark              VARCHAR2(100)
)`)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 1)

	raw, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.CreateTableStmt{}, raw.Stmt)
}

func TestParsePLSQLOmniMatchRecognize(t *testing.T) {
	statement := `SELECT * FROM TRADES MATCH_RECOGNIZE (
  PARTITION BY ACCOUNT_ID
  ORDER BY TRADE_TIME
  MEASURES FIRST(PRICE) AS FIRST_PRICE, LAST(PRICE) AS LAST_PRICE
  ONE ROW PER MATCH
  PATTERN (A B+)
  DEFINE B AS B.PRICE > A.PRICE
) MR`
	stmts, err := SplitSQL(statement)
	require.NoError(t, err)
	require.Len(t, stmts, 1)
	require.Equal(t, statement, stmts[0].Text)

	list, err := ParsePLSQLOmni(statement)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	raw, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.IsType(t, &ast.SelectStmt{}, raw.Stmt)
}

func TestParsePLSQLOmniPreservesScriptOffsets(t *testing.T) {
	sql := `PROMPT setup
SELECT 1 FROM DUAL;

SELECT name FROM users;
`
	list, err := ParsePLSQLOmni(sql)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 2)

	first, ok := list.Items[0].(*ast.RawStmt)
	require.True(t, ok)
	require.Equal(t, strings.Index(sql, "SELECT 1"), first.Loc.Start)
	require.Equal(t, strings.Index(sql, ";"), first.Loc.End)

	second, ok := list.Items[1].(*ast.RawStmt)
	require.True(t, ok)
	require.Equal(t, strings.Index(sql, "SELECT name"), second.Loc.Start)
	require.Equal(t, strings.LastIndex(sql, ";"), second.Loc.End)
}

func TestParsePLSQLOmniReturnsParseError(t *testing.T) {
	_, err := ParsePLSQLOmni("SELECT * FROM")
	require.Error(t, err)
}

func TestOracleOmniASTWrapper(t *testing.T) {
	node := &ast.SelectStmt{}
	start := &storepb.Position{Line: 2, Column: 3}
	omniAST := &OmniAST{
		Node:          node,
		Text:          "SELECT 1 FROM DUAL",
		StartPosition: start,
	}

	var parsed base.AST = omniAST
	require.Equal(t, start, parsed.ASTStartPosition())

	got, ok := GetOmniNode(parsed)
	require.True(t, ok)
	require.Same(t, node, got)

	got, ok = GetOmniNode(nil)
	require.False(t, ok)
	require.Nil(t, got)
}

func TestOracleOmniASTUsesOmniNode(t *testing.T) {
	start := &storepb.Position{Line: 4, Column: 1}
	omniAST := &OmniAST{
		Node:          &ast.SelectStmt{},
		Text:          "SELECT * FROM T",
		StartPosition: start,
	}

	node, ok := GetOmniNode(omniAST)
	require.True(t, ok)
	require.Same(t, omniAST.Node, node)
}

func TestOracleByteOffsetToRunePosition(t *testing.T) {
	position := ByteOffsetToRunePosition("SELECT\n日本 FROM DUAL", len("SELECT\n日"))
	require.Equal(t, int32(2), position.Line)
	require.Equal(t, int32(2), position.Column)
}
