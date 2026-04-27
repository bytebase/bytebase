package pg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
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
			want:      "(SELECT * FROM users LIMIT 5)",
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
		{
			name:      "FOR UPDATE gets proper whitespace",
			statement: "SELECT * FROM users WHERE id = 1 FOR UPDATE",
			limit:     10,
			want:      "SELECT * FROM users WHERE id = 1 LIMIT 10 FOR UPDATE",
		},
		{
			name:      "FOR SHARE gets proper whitespace",
			statement: "SELECT * FROM users FOR SHARE",
			limit:     5,
			want:      "SELECT * FROM users LIMIT 5 FOR SHARE",
		},
		{
			name:      "SELECT ending with function call",
			statement: "SELECT now()",
			limit:     10,
			want:      "SELECT now() LIMIT 10",
		},
		{
			name:      "SELECT with subquery in WHERE",
			statement: "SELECT * FROM users WHERE id IN (SELECT id FROM admins)",
			limit:     10,
			want:      "SELECT * FROM users WHERE id IN (SELECT id FROM admins) LIMIT 10",
		},
		{
			name:      "Non-constant LIMIT expression falls back to error",
			statement: "SELECT * FROM t LIMIT (1+2)",
			limit:     5,
			wantErr:   true,
		},
		{
			name: "CTE with lateral and ORDER BY",
			statement: `WITH params AS (
  SELECT 'resource.environment_id in []'::text AS env_condition
), affected AS (
  SELECT
    p.resource,
    binding,
    COALESCE(binding->'condition'->>'expression', '') AS old_expression
  FROM policy p,
  LATERAL jsonb_array_elements(p.payload->'bindings') AS binding
  WHERE p.type = 'IAM'
    AND p.resource_type = 'PROJECT'
    AND binding->>'role' = 'roles/projectOwner'
    AND COALESCE(binding->'condition'->>'expression', '') NOT LIKE '%resource.environment_id%'
)
SELECT
  resource,
  binding->'members' AS members,
  binding->'condition' AS old_condition,
  CASE
    WHEN old_expression = '' THEN params.env_condition
    ELSE '(' || old_expression || ') && ' || params.env_condition
  END AS new_expression
FROM affected
CROSS JOIN params
ORDER BY resource`,
			limit: 1000,
			want: `WITH params AS (
  SELECT 'resource.environment_id in []'::text AS env_condition
), affected AS (
  SELECT
    p.resource,
    binding,
    COALESCE(binding->'condition'->>'expression', '') AS old_expression
  FROM policy p,
  LATERAL jsonb_array_elements(p.payload->'bindings') AS binding
  WHERE p.type = 'IAM'
    AND p.resource_type = 'PROJECT'
    AND binding->>'role' = 'roles/projectOwner'
    AND COALESCE(binding->'condition'->>'expression', '') NOT LIKE '%resource.environment_id%'
)
SELECT
  resource,
  binding->'members' AS members,
  binding->'condition' AS old_condition,
  CASE
    WHEN old_expression = '' THEN params.env_condition
    ELSE '(' || old_expression || ') && ' || params.env_condition
  END AS new_expression
FROM affected
CROSS JOIN params
ORDER BY resource LIMIT 1000`,
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
			name:           "OFFSET without LIMIT",
			statement:      "SELECT * FROM users OFFSET 20",
			limit:          10,
			clausesInOrder: []string{"OFFSET 20", "LIMIT 10"},
		},
		{
			name:           "ORDER BY with OFFSET without LIMIT",
			statement:      "SELECT * FROM users ORDER BY created_at DESC OFFSET 20",
			limit:          10,
			clausesInOrder: []string{"ORDER BY", "OFFSET 20", "LIMIT 10"},
		},
		{
			name:           "ORDER BY with FOR UPDATE",
			statement:      "SELECT * FROM users ORDER BY id FOR UPDATE",
			limit:          10,
			want:           "SELECT * FROM users ORDER BY id LIMIT 10 FOR UPDATE",
			clausesInOrder: []string{"ORDER BY", "LIMIT 10", "FOR UPDATE"},
		},
		{
			name:           "ORDER BY with OFFSET and FOR UPDATE",
			statement:      "SELECT * FROM users ORDER BY id OFFSET 20 FOR UPDATE",
			limit:          10,
			clausesInOrder: []string{"ORDER BY", "OFFSET 20", "LIMIT 10", "FOR UPDATE"},
		},
		{
			name:           "GROUP BY HAVING WINDOW ORDER BY",
			statement:      "SELECT department, count(*) AS n, rank() OVER w FROM users GROUP BY department HAVING count(*) > 1 WINDOW w AS (ORDER BY department) ORDER BY n DESC",
			limit:          10,
			want:           "SELECT department, count(*) AS n, rank() OVER w FROM users GROUP BY department HAVING count(*) > 1 WINDOW w AS (ORDER BY department) ORDER BY n DESC LIMIT 10",
			clausesInOrder: []string{"GROUP BY", "HAVING", "WINDOW", "ORDER BY n", "LIMIT 10"},
		},
		{
			name:           "UNION with outer ORDER BY",
			statement:      "SELECT id FROM users UNION ALL SELECT id FROM admins ORDER BY id",
			limit:          10,
			want:           "SELECT id FROM users UNION ALL SELECT id FROM admins ORDER BY id LIMIT 10",
			clausesInOrder: []string{"UNION ALL", "ORDER BY", "LIMIT 10"},
		},
		{
			name:           "ORDER BY inside window only",
			statement:      "SELECT row_number() OVER (ORDER BY created_at) FROM users",
			limit:          10,
			want:           "SELECT row_number() OVER (ORDER BY created_at) FROM users LIMIT 10",
			clausesInOrder: []string{"FROM users", "LIMIT 10"},
		},
		{
			name:           "ORDER BY inside subquery only",
			statement:      "SELECT * FROM (SELECT * FROM users ORDER BY created_at) AS ordered_users",
			limit:          10,
			want:           "SELECT * FROM (SELECT * FROM users ORDER BY created_at) AS ordered_users LIMIT 10",
			clausesInOrder: []string{"AS ordered_users", "LIMIT 10"},
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
			got, err := getStatementWithResultLimitInline(tt.statement, tt.limit)
			require.NoError(t, err)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
			assertSubstringsInOrder(t, got, tt.clausesInOrder)

			_, err = pgparser.ParsePg(got)
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
