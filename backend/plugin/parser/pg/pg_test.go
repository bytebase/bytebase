package pg

import (
	"errors"
	"testing"

	omniparser "github.com/bytebase/omni/pg/parser"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLParser(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		errorMessage string
	}{
		{
			name:      "valid multi-statement",
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			name:      "valid join",
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1",
		},
		{
			name:         "trailing FROM without table",
			statement:    "SELECT a > (select max(a) from t1) FROM",
			errorMessage: `syntax error at end of input`,
		},
		{
			name:         "multi-statement with typo",
			statement:    "SELECT 1;\n   SELEC 5;\nSELECT 6;",
			errorMessage: `syntax error at or near "SELEC"`,
		},
		{
			name:         "single statement with typo",
			statement:    "SELECT * FRAM t1;",
			errorMessage: `syntax error at or near "FRAM"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stmts, err := SplitSQL(test.statement)
			require.NoError(t, err)

			var firstErr error
			for _, stmt := range stmts {
				if stmt.Empty {
					continue
				}
				_, parseErr := ParsePg(stmt.Text)
				if parseErr != nil && firstErr == nil {
					firstErr = parseErr
				}
			}

			if test.errorMessage == "" {
				require.NoError(t, firstErr, "statement: %s", test.statement)
			} else {
				require.Error(t, firstErr)
				var pe *omniparser.ParseError
				require.True(t, errors.As(firstErr, &pe), "expected *ParseError, got %T", firstErr)
				require.Equal(t, test.errorMessage, pe.Message)
			}
		})
	}
}
