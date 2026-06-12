package snowflake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestDiagnose(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		expectErr bool
	}{
		{
			name:      "Valid SELECT statement",
			sql:       "SELECT * FROM users",
			expectErr: false,
		},
		{
			name:      "Valid CREATE TABLE statement",
			sql:       "CREATE TABLE users (id INT, name VARCHAR)",
			expectErr: false,
		},
		{
			name:      "Valid CREATE WAREHOUSE statement",
			sql:       "CREATE WAREHOUSE my_wh WITH WAREHOUSE_SIZE = 'XSMALL'",
			expectErr: false,
		},
		{
			name:      "Missing select list",
			sql:       "SELECT FROM users",
			expectErr: true,
		},
		{
			// Unrecognized CREATE object keyword (TABLEE) — omni reports a
			// "CREATE statement parsing is not yet supported" stub here, which is
			// still a genuine syntax error (the object keyword is invalid).
			name:      "Invalid CREATE object",
			sql:       "CREATE TABLEE users (id INT)",
			expectErr: true,
		},
		{
			name:      "Truncated statement (end of input)",
			sql:       "SELECT * FROM t WHERE",
			expectErr: true,
		},
		{
			name:      "Unknown leading keyword",
			sql:       "FOO BAR BAZ",
			expectErr: true,
		},
		{
			// Resolved divergence: omni's strict parse (post-#303) now rejects
			// unconsumed trailing tokens, matching the legacy ANTLR behavior —
			// the stray "FFROM" token produces a syntax diagnostic.
			name:      "Trailing garbage token after valid prefix",
			sql:       "SELECT * FFROM users",
			expectErr: true,
		},
		{
			// DIVERGENCE (omni vs legacy ANTLR): empty / whitespace-only input is
			// zero statements with no error. The legacy path appended a ';' and
			// reported a syntax error on the lone ';'; omni correctly treats an
			// empty buffer as not-an-error.
			name:      "Empty statement",
			sql:       "",
			expectErr: false,
		},
		{
			name:      "Whitespace-only statement",
			sql:       "   \n\t ",
			expectErr: false,
		},
		{
			// DIVERGENCE (omni vs legacy ANTLR): a lone or doubled ';' is an empty
			// statement (zero statements), not a syntax error, under omni.
			name:      "Lone semicolons",
			sql:       ";;",
			expectErr: false,
		},
		{
			name:      "Valid multi-statement script",
			sql:       "SELECT 1; SELECT 2;",
			expectErr: false,
		},
	}

	ctx := context.Background()
	diagnoseCtx := base.DiagnoseContext{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics, err := Diagnose(ctx, diagnoseCtx, tt.sql)
			require.NoError(t, err, "Diagnose should not return an error")

			if tt.expectErr {
				assert.NotEmpty(t, diagnostics, "Expected diagnostics for invalid SQL")
			} else {
				assert.Empty(t, diagnostics, "Expected no diagnostics for valid SQL")
			}
		})
	}
}

func TestParseSnowflakeSQL(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		expectErr bool
	}{
		{
			name:      "Valid statement with semicolon",
			sql:       "SELECT * FROM users;",
			expectErr: false,
		},
		{
			name:      "Valid statement without semicolon",
			sql:       "SELECT * FROM users",
			expectErr: false,
		},
		{
			name:      "Invalid statement (missing select list)",
			sql:       "SELECT FROM users",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseSnowflakeSQL(tt.sql)
			if tt.expectErr {
				assert.NotNil(t, err, "Expected syntax error")
			} else {
				assert.Nil(t, err, "Expected no syntax error")
			}
		})
	}
}
