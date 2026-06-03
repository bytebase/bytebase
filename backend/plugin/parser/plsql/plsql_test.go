package plsql

import (
	"errors"
	"testing"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPLSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement: `UPDATE t1 SET (c1) = 1 WHERE c2 = 2;`,
		},
		{
			statement: `
			SELECT q'\This is String\' FROM DUAL;
			`,
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			statement: "CREATE TABLE t1 (c1 NUMBER(10,2), c2 VARCHAR2(10));",
		},
		{
			statement: "SELECT * FROM t1;",
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1",
		},
		{
			statement:    "SELECT * FROM t1 WHERE c1 = ",
			errorMessage: "ERROR: syntax error at end of input (SQLSTATE 42601)",
		},
		{
			statement:    "SELECT 1 FROM DUAL;\n   SELEC 5 FROM DUAL;\nSELECT 6 FROM DUAL;",
			errorMessage: "ERROR: syntax error at or near \"SELEC\" (SQLSTATE 42601)",
		},
		// BYT-9302: CREATE TABLE with INTERVAL partitioning and DATE literal bound.
		{
			statement: `CREATE TABLE GCP.LEAD_DROP_MC_NATIVE_DATA
(
  TXN_DATE  DATE,
  USERID    VARCHAR2(100),
  CUSTID    VARCHAR2(100),
  SCREENID  VARCHAR2(500),
  EVENTTIME DATE,
  STATUS    NUMBER
)
PARTITION BY RANGE (TXN_DATE)
INTERVAL (NUMTODSINTERVAL(1,'DAY'))
(
  PARTITION P0 VALUES LESS THAN (DATE '2026-01-01')
);`,
		},
	}

	for _, test := range tests {
		list, err := ParsePLSQLOmni(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err)
			require.NotEmpty(t, list.Items)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}

func TestPLSQLParser_MultipleStatements(t *testing.T) {
	tests := []struct {
		statement     string
		expectedCount int   // Number of individual statements
		expectedLines []int // StartPosition.Line - 1 for each result (0-based line of first character, including leading whitespace)
	}{
		{
			statement:     "SELECT * FROM t1;",
			expectedCount: 1,
			expectedLines: []int{0},
		},
		{
			// Statement 2's first character is the newline at end of line 1
			statement: `SELECT * FROM t1;
SELECT * FROM t2;`,
			expectedCount: 2,
			expectedLines: []int{0, 0},
		},
		{
			// Statement 2's first char is newline at end of line 1, statement 3's first char is newline at end of line 2
			statement: `SELECT * FROM t1;
SELECT * FROM t2;
INSERT INTO t3 VALUES (1, 2);`,
			expectedCount: 3,
			expectedLines: []int{0, 0, 1},
		},
	}

	for _, test := range tests {
		stmts, err := base.ParseStatements(storepb.Engine_ORACLE, test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expectedCount, len(stmts), "Statement: %s", test.statement)

		for i, stmt := range stmts {
			require.Equal(t, test.expectedLines[i], base.GetLineOffset(stmt.Start), "Statement %d", i+1)
		}
	}
}

func TestParsePLSQLStatementsOmniFirst(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, "SELECT * FROM T;")
	require.NoError(t, err)
	require.Len(t, stmts, 1)

	omniAST, ok := stmts[0].AST.(*OmniAST)
	require.True(t, ok)
	require.Equal(t, "SELECT * FROM T", omniAST.Text)
	require.IsType(t, &ast.SelectStmt{}, omniAST.Node)
}

func TestParsePLSQLStatementsParsesTriggerReferencingWithOmni(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, `SELECT * FROM T;
CREATE OR REPLACE TRIGGER trg
BEFORE INSERT OR UPDATE OF col1, col2 ON tbl
REFERENCING OLD AS o NEW AS n
FOR EACH ROW
WHEN (n.col1 > 0)
BEGIN
  :n.col2 := :o.col2 + 1;
END;`)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	_, ok := stmts[0].AST.(*OmniAST)
	require.True(t, ok)

	omniAST, ok := stmts[1].AST.(*OmniAST)
	require.True(t, ok)
	trigger, ok := omniAST.Node.(*ast.CreateTriggerStmt)
	require.True(t, ok)
	require.NotNil(t, trigger.Referencing)
	require.Equal(t, "O", trigger.Referencing.OldAlias)
	require.Equal(t, "N", trigger.Referencing.NewAlias)
}

func TestParsePLSQLStatementsOmniErrorUsesOriginalLine(t *testing.T) {
	_, err := base.ParseStatements(storepb.Engine_ORACLE, `SELECT *
FROM T;
CREATE TABLE GCP.LEAD_DROP_MC_NATIVE_DATA
(
  TXN_DATE DATE
)
PARTITION BY RANGE (TXN_DATE)
INTERVAL (NUMTODSINTERVAL(1,'DAY'))
(
  PARTITION P0 VALUES LESS THAN (DATE '2026-01-01')
BROKEN`)
	require.Error(t, err)

	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr))
	require.NotNil(t, syntaxErr.Position)
	require.Equal(t, int32(11), syntaxErr.Position.Line)
}

func TestParsePLSQLStatementsOmniErrorUsesOriginalColumn(t *testing.T) {
	statement := `SELECT * FROM T; CREATE TABLE GCP.LEAD_DROP_MC_NATIVE_DATA (TXN_DATE DATE) PARTITION BY RANGE (TXN_DATE) INTERVAL (NUMTODSINTERVAL(1,'DAY')) (PARTITION P0 VALUES LESS THAN (DATE '2026-01-01') BROKEN`

	_, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.Error(t, err)
	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr))
	require.NotNil(t, syntaxErr.Position)
	require.Equal(t, int32(1), syntaxErr.Position.Line)
	require.Equal(t, int32(199), syntaxErr.Position.Column)
}
