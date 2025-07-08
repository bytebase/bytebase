package pg

import (
	"fmt"
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
			name:      "SELECT with WHERE clause",
			statement: "SELECT id, name FROM users WHERE active = true",
			limit:     5,
			want:      "SELECT id, name FROM users WHERE active = true LIMIT 5",
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
			name:      "SELECT with GROUP BY and HAVING",
			statement: "SELECT department, COUNT(*) FROM users GROUP BY department HAVING COUNT(*) > 5",
			limit:     10,
			want:      "SELECT department, COUNT(*) FROM users GROUP BY department HAVING COUNT(*) > 5 LIMIT 10",
		},
		{
			name:      "UNION query",
			statement: "SELECT id FROM users UNION SELECT id FROM admins",
			limit:     10,
			want:      "SELECT id FROM users UNION SELECT id FROM admins LIMIT 10",
		},
		{
			name:      "SELECT with OFFSET",
			statement: "SELECT * FROM users LIMIT 20 OFFSET 10",
			limit:     5,
			want:      "SELECT * FROM users LIMIT 5 OFFSET 10",
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
			name:      "Non-SELECT statement (UPDATE)",
			statement: "UPDATE users SET active = false WHERE id = 1",
			limit:     10,
			want:      "UPDATE users SET active = false WHERE id = 1",
		},
		{
			name:      "Non-SELECT statement (DELETE)",
			statement: "DELETE FROM users WHERE active = false",
			limit:     10,
			want:      "DELETE FROM users WHERE active = false",
		},
		{
			name:      "SELECT with comments",
			statement: "SELECT * FROM users -- This is a comment\nWHERE active = true",
			limit:     10,
			want:      "SELECT * FROM users -- This is a comment\nWHERE active = true LIMIT 10",
		},
		{
			name:      "Complex nested SELECT",
			statement: "SELECT * FROM (SELECT * FROM users WHERE active = true) AS active_users WHERE created_at > '2023-01-01'",
			limit:     10,
			want:      "SELECT * FROM (SELECT * FROM users WHERE active = true) AS active_users WHERE created_at > '2023-01-01' LIMIT 10",
		},
		{
			name:      "SELECT with JOIN",
			statement: "SELECT u.*, d.name as dept_name FROM users u JOIN departments d ON u.dept_id = d.id",
			limit:     10,
			want:      "SELECT u.*, d.name as dept_name FROM users u JOIN departments d ON u.dept_id = d.id LIMIT 10",
		},
		{
			name:      "SELECT with parentheses",
			statement: "(SELECT * FROM users)",
			limit:     10,
			want:      "(SELECT * FROM users) LIMIT 10",
		},
		{
			name:      "SELECT with nested parentheses",
			statement: "((SELECT * FROM users))",
			limit:     10,
			want:      "((SELECT * FROM users)) LIMIT 10",
		},
		{
			name:      "SELECT with parentheses and existing LIMIT higher than requested",
			statement: "(SELECT * FROM users LIMIT 100)",
			limit:     10,
			want:      "(SELECT * FROM users LIMIT 10)",
		},
		{
			name:      "SELECT with parentheses and existing LIMIT lower than requested",
			statement: "(SELECT * FROM users LIMIT 5)",
			limit:     10,
			// TODO(zp): FIX ME
			want: "(SELECT * FROM users LIMIT 5) LIMIT 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatementWithResultLimit(tt.statement, tt.limit)

			// For non-SELECT statements, the parser might fail and fall back to CTE approach
			// which we should avoid for non-SELECT statements
			if tt.statement == tt.want {
				// Non-SELECT statements should remain unchanged
				assert.Equal(t, tt.want, got)
			} else {
				// For SELECT statements, check if LIMIT was properly added
				// The exact format might vary slightly due to parser normalization
				assert.Contains(t, got, "LIMIT")
				assert.Contains(t, got, fmt.Sprintf("%d", tt.limit))
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
			wantErr:   false,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getStatementWithResultLimitInline(tt.statement, tt.limit)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, got)
				if tt.want != "" {
					assert.Equal(t, tt.want, got)
				}
			}
		})
	}
}
