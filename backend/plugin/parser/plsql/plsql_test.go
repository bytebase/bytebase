package plsql

import (
	"errors"
	"testing"

	"github.com/bytebase/omni/oracle/ast"
	parser "github.com/bytebase/parser/plsql"
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
			errorMessage: "Syntax error at line 1:27 \nrelated text: SELECT * FROM t1 WHERE c1 =",
		},
		{
			statement:    "SELECT 1 FROM DUAL;\n   SELEC 5 FROM DUAL;\nSELECT 6 FROM DUAL;",
			errorMessage: "Syntax error at line 2:10 \nrelated text: SELECT 1 FROM DUAL;\n   SELEC 5",
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
		results, err := ParsePLSQL(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err)
			require.NotEmpty(t, results)
			_, ok := results[0].Tree.(*parser.Sql_scriptContext)
			require.True(t, ok)
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
		results, err := ParsePLSQL(test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expectedCount, len(results), "Statement: %s", test.statement)

		for i, result := range results {
			require.Equal(t, test.expectedLines[i], base.GetLineOffset(result.StartPosition), "Statement %d", i+1)
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

func TestParsePLSQLStatementsFallsBackToANTLR(t *testing.T) {
	stmts, err := base.ParseStatements(storepb.Engine_ORACLE, `SELECT * FROM T;
CREATE TABLE GCP.LEAD_DROP_MC_NATIVE_DATA
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
);`)
	require.NoError(t, err)
	require.Len(t, stmts, 2)

	_, ok := stmts[0].AST.(*OmniAST)
	require.True(t, ok)

	_, ok = stmts[1].AST.(*base.ANTLRAST)
	require.True(t, ok)
}

func TestParsePLSQLStatementsFallbackErrorUsesOriginalLine(t *testing.T) {
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

func TestParsePLSQLStatementsFallbackErrorUsesOriginalColumn(t *testing.T) {
	statement := `SELECT * FROM T; CREATE TABLE GCP.LEAD_DROP_MC_NATIVE_DATA (TXN_DATE DATE) PARTITION BY RANGE (TXN_DATE) INTERVAL (NUMTODSINTERVAL(1,'DAY')) (PARTITION P0 VALUES LESS THAN (DATE '2026-01-01') BROKEN`

	_, wholeErr := ParsePLSQL(statement)
	require.Error(t, wholeErr)
	var wholeSyntaxErr *base.SyntaxError
	require.True(t, errors.As(wholeErr, &wholeSyntaxErr))
	require.NotNil(t, wholeSyntaxErr.Position)

	_, err := base.ParseStatements(storepb.Engine_ORACLE, statement)
	require.Error(t, err)
	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr))
	require.NotNil(t, syntaxErr.Position)
	require.Equal(t, wholeSyntaxErr.Position.Line, syntaxErr.Position.Line)
	require.Equal(t, wholeSyntaxErr.Position.Column, syntaxErr.Position.Column)
}
