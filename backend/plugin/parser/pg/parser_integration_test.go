package pg

import (
	"errors"
	"testing"

	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/stretchr/testify/require"
)

// TestParsePostgresErrorLine tests that the omni parser correctly reports error line numbers.
func TestParsePostgresErrorLine(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		expectedLine int32
	}{
		{
			name:         "single statement error",
			statement:    "SELECT * FRAM t1;",
			expectedLine: 1,
		},
		{
			name: "second statement error",
			statement: `SELECT 1;
SELECT * FRAM t2;`,
			expectedLine: 2,
		},
		{
			name: "third statement error",
			statement: `SELECT 1;
SELECT 2;
SELECT * FRAM t3;`,
			expectedLine: 3,
		},
		{
			name: "error in multi-line statement",
			statement: `SELECT
	1,
	2
FRAM t1;`,
			expectedLine: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stmts, err := SplitSQL(test.statement)
			require.NoError(t, err)

			for _, stmt := range stmts {
				if stmt.Empty {
					continue
				}
				_, parseErr := ParsePg(stmt.Text)
				if parseErr != nil {
					var pe *omniparser.ParseError
					require.True(t, errors.As(parseErr, &pe), "expected *ParseError, got %T", parseErr)
					pos := ByteOffsetToRunePosition(stmt.Text, pe.Position)
					// Adjust line by the statement's base line.
					if stmt.Start != nil {
						pos.Line += stmt.Start.Line - 1
					}
					require.Equal(t, test.expectedLine, pos.Line,
						"incorrect error line for statement:\n%s", test.statement)
					return
				}
			}
			t.Fatalf("expected parse error for statement:\n%s", test.statement)
		})
	}
}
