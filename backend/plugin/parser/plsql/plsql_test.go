package plsql

import (
	"testing"

	parser "github.com/bytebase/parser/plsql"
	"github.com/stretchr/testify/require"

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
