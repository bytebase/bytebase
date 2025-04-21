package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TestTrinoQuerySpanTypes tests the query span extraction for different Trino SQL statement types.
func TestTrinoQuerySpanTypes(t *testing.T) {
	testCases := []struct {
		name     string
		sql      string
		database string
		schema   string
		wantType base.QueryType
	}{
		// SELECT queries
		{
			name:     "Simple SELECT",
			sql:      "SELECT id, name FROM users;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SELECT with JOIN",
			sql:      "SELECT u.id, u.name, o.order_id FROM users u JOIN orders o ON u.id = o.user_id;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SELECT with subquery",
			sql:      "SELECT id, name FROM (SELECT id, name FROM users) t;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SELECT with CTE",
			sql:      "WITH temp AS (SELECT id, name FROM users) SELECT id, name FROM temp;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SELECT from system tables",
			sql:      "SELECT * FROM system.runtime.nodes;",
			database: "catalog1",
			schema:   "public",
			wantType: base.SelectInfoSchema,
		},

		// DML queries
		{
			name:     "INSERT",
			sql:      "INSERT INTO users (id, name) VALUES (1, 'John');",
			database: "catalog1",
			schema:   "public",
			wantType: base.DML,
		},
		{
			name:     "UPDATE",
			sql:      "UPDATE users SET name = 'Jane' WHERE id = 1;",
			database: "catalog1",
			schema:   "public",
			wantType: base.DML,
		},
		{
			name:     "DELETE",
			sql:      "DELETE FROM users WHERE id = 1;",
			database: "catalog1",
			schema:   "public",
			wantType: base.DML,
		},

		// DDL queries
		{
			name:     "CREATE TABLE",
			sql:      "CREATE TABLE new_table (id INT, name VARCHAR);",
			database: "catalog1",
			schema:   "public",
			wantType: base.DDL,
		},
		{
			name:     "DROP TABLE",
			sql:      "DROP TABLE users;",
			database: "catalog1",
			schema:   "public",
			wantType: base.DDL,
		},
		{
			name:     "ALTER TABLE",
			sql:      "ALTER TABLE users ADD COLUMN email VARCHAR;",
			database: "catalog1",
			schema:   "public",
			wantType: base.DDL,
		},
		{
			name:     "CREATE VIEW",
			sql:      "CREATE VIEW user_view AS SELECT id, name FROM users;",
			database: "catalog1",
			schema:   "public",
			wantType: base.DDL,
		},

		// Special queries
		{
			name:     "EXPLAIN",
			sql:      "EXPLAIN SELECT id, name FROM users;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Explain,
		},
		{
			name:     "EXPLAIN ANALYZE",
			sql:      "EXPLAIN ANALYZE SELECT id, name FROM users;",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "SHOW TABLES",
			sql:      "SHOW TABLES;",
			database: "catalog1",
			schema:   "public",
			wantType: base.SelectInfoSchema,
		},

		// Trino-specific features
		{
			name:     "UNNEST array",
			sql:      "SELECT id, t.name FROM users CROSS JOIN UNNEST(names) AS t(name);",
			database: "catalog1",
			schema:   "public",
			wantType: base.Select,
		},
		{
			name:     "Multiple catalog query",
			sql:      "SELECT a.id, b.id FROM catalog1.public.users a JOIN catalog2.public.orders b ON a.id = b.user_id;",
			database: "catalog3",
			schema:   "public",
			wantType: base.Select,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock GetQuerySpanContext
			gCtx := base.GetQuerySpanContext{
				InstanceID: "test-instance",
				Engine:     storepb.Engine_TRINO,
				// No metadata function provided - mock implementation will handle this
			}

			// Get query span
			span, err := GetQuerySpan(context.Background(), gCtx, tc.sql, tc.database, tc.schema, true)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.wantType, span.Type, "Incorrect query type for SQL: %s", tc.sql)
			}
		})
	}
}

// TestTrinoQuerySpanSources tests source column resolution for common Trino SQL queries.
func TestTrinoQuerySpanSources(t *testing.T) {
	testCases := []struct {
		name              string
		sql               string
		database          string
		schema            string
		expectTableSource string
	}{
		{
			name:              "SELECT with simple table",
			sql:               "SELECT id, name FROM users;",
			database:          "catalog1",
			schema:            "public",
			expectTableSource: "catalog1.public.users",
		},
		{
			name:              "SELECT with qualified schema",
			sql:               "SELECT id, name FROM public.users;",
			database:          "catalog1",
			schema:            "default",
			expectTableSource: "catalog1.public.users",
		},
		{
			name:              "SELECT with fully qualified name",
			sql:               "SELECT id, name FROM catalog1.public.users;",
			database:          "catalog2",
			schema:            "default",
			expectTableSource: "catalog1.public.users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock GetQuerySpanContext
			gCtx := base.GetQuerySpanContext{
				InstanceID: "test-instance",
				Engine:     storepb.Engine_TRINO,
				// No metadata function provided
			}

			// Get query span
			span, err := GetQuerySpan(context.Background(), gCtx, tc.sql, tc.database, tc.schema, true)
			if assert.NoError(t, err) {
				// Check that the expected table source is in the source columns
				found := false
				for sourceCol := range span.SourceColumns {
					resource := base.ColumnResource{
						Database: sourceCol.Database,
						Schema:   sourceCol.Schema,
						Table:    sourceCol.Table,
					}
					if resource.String() == tc.expectTableSource {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected table source %s not found in source columns", tc.expectTableSource)
			}
		})
	}
}

// TestTrinoCaseSensitivity tests case sensitivity handling in Trino SQL parsing.
func TestTrinoCaseSensitivity(t *testing.T) {
	// The test's behavior depends on implementation details, but we can at least
	// verify that ignoring case sensitivity allows us to query case-insensitively

	testSQL := "SELECT id, name FROM Users;" // Note the uppercase 'Users'
	database := "catalog1"
	schema := "public"

	// Case 1: Ignore case sensitivity
	gCtx := base.GetQuerySpanContext{
		InstanceID: "test-instance",
		Engine:     storepb.Engine_TRINO,
	}

	// With case sensitivity ignored
	spanIgnoreCase, err := GetQuerySpan(context.Background(), gCtx, testSQL, database, schema, true)
	if assert.NoError(t, err) {
		assert.Equal(t, base.Select, spanIgnoreCase.Type)

		// Check if the table appears in source columns (lowercase 'users')
		foundTable := false
		expectedResource := "catalog1.public.users"
		for sourceCol := range spanIgnoreCase.SourceColumns {
			resource := base.ColumnResource{
				Database: sourceCol.Database,
				Schema:   sourceCol.Schema,
				Table:    sourceCol.Table,
			}
			if resource.String() == expectedResource {
				foundTable = true
				break
			}
		}
		assert.True(t, foundTable, "Case-insensitive query should find 'users' table")
	}
}
