package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestParsePostgresErrorLine tests that the ANTLR parser correctly reports error line numbers.
// This test is in the pg package (not pg/legacy) to ensure the ANTLR parser is registered via init().
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
			_, err := base.Parse(storepb.Engine_POSTGRES, test.statement)
			require.Error(t, err)
			syntaxErr, ok := err.(*base.SyntaxError)
			require.True(t, ok, "expected *base.SyntaxError, got %T", err)
			require.NotNil(t, syntaxErr.Position)
			require.Equal(t, test.expectedLine, syntaxErr.Position.Line,
				"incorrect error line for statement:\n%s", test.statement)
		})
	}
}
