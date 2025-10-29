package legacy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParsePostgresForRegistryError(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		expectedLine int32 // Expected to be the END line of the statement with error
	}{
		{
			name:         "single statement with syntax error",
			statement:    "SELECT * FRAM t1;",
			expectedLine: 1,
		},
		{
			name: "multi-line single statement with syntax error",
			statement: `SELECT
	1,
	2
FRAM t1;`,
			expectedLine: 4, // Last line of the statement
		},
		{
			name: "multiple statements - error in first",
			statement: `SELECT * FRAM t1;
SELECT 1;
SELECT 2;`,
			expectedLine: 1, // End of first statement
		},
		{
			name: "multiple statements - error in second",
			statement: `SELECT 1;
SELECT * FRAM t2;
SELECT 2;`,
			expectedLine: 2, // End of second statement
		},
		{
			name: "multiple statements - error in third",
			statement: `SELECT 1;
SELECT 2;
SELECT * FRAM t3;`,
			expectedLine: 3, // End of third statement
		},
		{
			name: "multi-line statements - error in second statement",
			statement: `SELECT
	1,
	2
FROM t1;
SELECT
	*
FRAM t2;`,
			expectedLine: 7, // End of second statement
		},
		{
			name: "complex multi-line - error on specific line",
			statement: `-- Comment line 1
SELECT 1;
-- Comment line 3
SELECT
	a,
	b,
	c,
	d
FRAM t1;
SELECT 2;`,
			expectedLine: 9, // End of the SELECT statement with error
		},
		{
			name: "error with WHERE typo",
			statement: `SELECT 1;
SELECT 2;
SELECT * FROM t1 WHER id = 1;`,
			expectedLine: 3, // End of third statement
		},
		{
			name: "error in second statement with indentation",
			statement: `select 1;
   selec 2;
select 3;`,
			expectedLine: 2, // End of second statement
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parsePostgresForRegistry(test.statement)
			require.Error(t, err, "expected syntax error for statement: %s", test.statement)
			syntaxErr, ok := err.(*base.SyntaxError)
			require.True(t, ok, "expected error to be *base.SyntaxError, got %T", err)
			require.NotNil(t, syntaxErr.Position, "expected position to be set")
			require.Equal(t, test.expectedLine, syntaxErr.Position.GetLine(),
				"incorrect line number for statement:\n%s\nError: %s", test.statement, syntaxErr.Message)
		})
	}
}

func TestParsePostgresForRegistrySuccess(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "single statement",
			statement: "SELECT 1;",
		},
		{
			name: "multiple statements",
			statement: `SELECT 1;
SELECT 2;
SELECT 3;`,
		},
		{
			name: "multi-line statement",
			statement: `SELECT
	1,
	2,
	3
FROM t1;`,
		},
		{
			name: "complex multi-line statements",
			statement: `-- Comment
SELECT 1;

SELECT
	a,
	b,
	c
FROM t1
WHERE a > 1;

SELECT * FROM t2;`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := parsePostgresForRegistry(test.statement)
			require.NoError(t, err, "unexpected error for statement: %s", test.statement)
			require.NotNil(t, result, "expected non-nil result")
		})
	}
}
