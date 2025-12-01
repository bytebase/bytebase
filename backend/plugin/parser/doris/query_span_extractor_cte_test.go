package doris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestQuerySpanExtractor_CTE(t *testing.T) {
	tests := []struct {
		name            string
		sql             string
		expectedTables  []base.ColumnResource
		defaultDatabase string
	}{
		{
			name: "CTE with JOIN between CTE and physical table",
			sql: `WITH summary AS (
				SELECT
					user_id,
					SUM(amount) as total_amount
				FROM orders
				WHERE created_at > '2024-01-01'
				GROUP BY user_id
			)
			SELECT
				u.id,
				u.name,
				s.total_amount
			FROM summary s
			LEFT JOIN users u ON s.user_id = u.id`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "orders"},
				{Database: "test", Table: "users"},
			},
			defaultDatabase: "test",
		},
		{
			name: "Multiple CTEs with cross-database tables",
			sql: `WITH
				cte1 AS (SELECT * FROM db1.table1),
				cte2 AS (SELECT * FROM db2.table2 WHERE active = 1)
			SELECT
				c1.id,
				c2.name
			FROM cte1 c1
			INNER JOIN cte2 c2 ON c1.ref_id = c2.id`,
			expectedTables: []base.ColumnResource{
				{Database: "db1", Table: "table1"},
				{Database: "db2", Table: "table2"},
			},
			defaultDatabase: "test",
		},
		{
			name: "Simple CTE",
			sql: `WITH cte AS (SELECT * FROM users)
SELECT * FROM cte`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "users"},
			},
			defaultDatabase: "test",
		},
		{
			name: "Multiple CTEs",
			sql: `WITH
cte1 AS (SELECT * FROM table1),
cte2 AS (SELECT * FROM table2)
SELECT * FROM cte1 JOIN cte2`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "table1"},
				{Database: "test", Table: "table2"},
			},
			defaultDatabase: "test",
		},
		{
			name: "CTE shadowing table name",
			sql: `WITH products AS (
				SELECT p.*, c.name as category_name
				FROM items p
				JOIN categories c ON p.category_id = c.id
			)
			SELECT * FROM products WHERE category_name = 'Electronics'`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "items"},
				{Database: "test", Table: "categories"},
			},
			defaultDatabase: "test",
		},
		{
			name: "Nested CTEs",
			sql: `WITH outer_cte AS (
  WITH inner_cte AS (SELECT * FROM base_table)
  SELECT * FROM inner_cte
)
SELECT * FROM outer_cte`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "base_table"},
			},
			defaultDatabase: "test",
		},
		{
			name: "CTE referencing another CTE",
			sql: `WITH
				base_data AS (
					SELECT id, name, department_id
					FROM employees
					WHERE active = true
				),
				dept_summary AS (
					SELECT
						d.name as dept_name,
						COUNT(b.id) as emp_count
					FROM base_data b
					JOIN departments d ON b.department_id = d.id
					GROUP BY d.name
				)
			SELECT * FROM dept_summary WHERE emp_count > 10`,
			expectedTables: []base.ColumnResource{
				{Database: "test", Table: "employees"},
				{Database: "test", Table: "departments"},
			},
			defaultDatabase: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := newQuerySpanExtractor(tt.defaultDatabase, base.GetQuerySpanContext{}, false)
			querySpan, err := extractor.getQuerySpan(context.Background(), tt.sql)
			require.NoError(t, err)

			// Convert SourceColumnSet to slice for comparison
			var actualTables []base.ColumnResource
			for table := range querySpan.SourceColumns {
				actualTables = append(actualTables, table)
			}

			// Check that we have the expected number of tables
			require.Equal(t, len(tt.expectedTables), len(actualTables),
				"Expected %d tables but got %d", len(tt.expectedTables), len(actualTables))

			// Check that all expected tables are present
			for _, expected := range tt.expectedTables {
				found := false
				for _, actual := range actualTables {
					if actual.Database == expected.Database && actual.Table == expected.Table {
						found = true
						break
					}
				}
				require.True(t, found, "Expected table %s.%s not found in result",
					expected.Database, expected.Table)
			}
		})
	}
}
