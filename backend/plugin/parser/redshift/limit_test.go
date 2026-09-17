package redshift

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		limit     int
		want      string
	}{
		{
			name:      "Simple SELECT without LIMIT",
			statement: "SELECT * FROM users",
			limit:     10,
			want:      "SELECT * FROM users LIMIT 10",
		},
		{
			name:      "SELECT with existing LIMIT higher than requested",
			statement: "SELECT * FROM users LIMIT 100",
			limit:     10,
			want:      "SELECT * FROM users LIMIT 10",
		},
		{
			name:      "SELECT with existing LIMIT lower than requested",
			statement: "SELECT * FROM users LIMIT 5",
			limit:     10,
			want:      "SELECT * FROM users LIMIT 5",
		},
		{
			name:      "WITH query (CTE)",
			statement: "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users",
			limit:     10,
			want:      "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users LIMIT 10",
		},
		{
			name:      "SELECT with ORDER BY",
			statement: "SELECT * FROM users ORDER BY created_at DESC",
			limit:     10,
			want:      "SELECT * FROM users ORDER BY created_at DESC LIMIT 10",
		},
		{
			name:      "UNION query",
			statement: "SELECT id FROM users UNION SELECT id FROM admins",
			limit:     10,
			want:      "SELECT id FROM users UNION SELECT id FROM admins LIMIT 10",
		},
		{
			name:      "SELECT with FOR UPDATE",
			statement: "SELECT * FROM users WHERE id = 1 FOR UPDATE",
			limit:     10,
			want:      "SELECT * FROM users WHERE id = 1 LIMIT 10 FOR UPDATE",
		},
		{
			name:      "Non-SELECT statement (INSERT)",
			statement: "INSERT INTO users (name) VALUES ('John')",
			limit:     10,
			want:      "INSERT INTO users (name) VALUES ('John')",
		},
		{
			name:      "Non-SELECT statement (DELETE)",
			statement: "DELETE FROM users WHERE active = false",
			limit:     10,
			want:      "DELETE FROM users WHERE active = false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statementWithResultLimit(tt.statement, tt.limit, "")

			if tt.statement == tt.want {
				assert.Equal(t, tt.want, got)
			} else {
				assert.Contains(t, got, "LIMIT")
			}
		})
	}
}

func TestGetStatementWithResultLimitInline(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		limit     int
		wantErr   bool
		want      string
	}{
		{
			name:      "Valid SELECT statement",
			statement: "SELECT * FROM users",
			limit:     10,
			want:      "SELECT * FROM users LIMIT 10",
		},
		{
			name:      "Invalid SQL syntax",
			statement: "SELECT * FROM WHERE",
			limit:     10,
			wantErr:   true,
		},
		{
			name:      "Empty statement",
			statement: "",
			limit:     10,
			wantErr:   true,
		},
		{
			name:      "FOR UPDATE gets proper whitespace",
			statement: "SELECT * FROM users WHERE id = 1 FOR UPDATE",
			limit:     10,
			want:      "SELECT * FROM users WHERE id = 1 LIMIT 10 FOR UPDATE",
		},
		{
			name:      "Non-constant LIMIT expression falls back to error",
			statement: "SELECT * FROM t LIMIT (1+2)",
			limit:     5,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := statementWithResultLimitInline(tt.statement, tt.limit)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, got)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestGetStatementWithResultLimitFallback locks Redshift's own fallback text,
// distinct from PostgreSQL's: a blind CTE-wrap for a statement the AST rewrite
// cannot handle, still never applied to a statement that already parsed clean.
func TestGetStatementWithResultLimitFallback(t *testing.T) {
	got := statementWithResultLimit("SELECT * FROM t LIMIT (1+2)", 5, "")
	require.Equal(t, "WITH result AS (\nSELECT * FROM t LIMIT (1+2)\n) SELECT * FROM result LIMIT 5;", got)
}

func TestGetStatementWithResultLimitInlineClauseOrder(t *testing.T) {
	tests := []struct {
		name           string
		statement      string
		limit          int
		want           string
		clausesInOrder []string
	}{
		{
			name:           "ORDER BY",
			statement:      "SELECT * FROM users ORDER BY created_at DESC",
			limit:          10,
			want:           "SELECT * FROM users ORDER BY created_at DESC LIMIT 10",
			clausesInOrder: []string{"ORDER BY", "LIMIT 10"},
		},
		{
			name:           "ORDER BY with FOR UPDATE",
			statement:      "SELECT * FROM users ORDER BY id FOR UPDATE",
			limit:          10,
			want:           "SELECT * FROM users ORDER BY id LIMIT 10 FOR UPDATE",
			clausesInOrder: []string{"ORDER BY", "LIMIT 10", "FOR UPDATE"},
		},
		{
			name:           "WITH query with outer ORDER BY",
			statement:      "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users ORDER BY created_at",
			limit:          10,
			want:           "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users ORDER BY created_at LIMIT 10",
			clausesInOrder: []string{"WITH", "ORDER BY", "LIMIT 10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := statementWithResultLimitInline(tt.statement, tt.limit)
			require.NoError(t, err)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
			assertSubstringsInOrder(t, got, tt.clausesInOrder)

			_, err = ParseRedshift(got)
			require.NoError(t, err, "rewritten SQL should remain parseable: %s", got)
		})
	}
}

func assertSubstringsInOrder(t *testing.T, s string, substrings []string) {
	t.Helper()

	start := 0
	for _, substring := range substrings {
		index := strings.Index(s[start:], substring)
		require.NotEqualf(t, -1, index, "expected %q after offset %d in %q", substring, start, s)
		start += index + len(substring)
	}
}
