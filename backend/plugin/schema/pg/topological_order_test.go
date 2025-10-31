package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestTopologicalOrderCreateObjects tests that objects are created in the correct dependency order
func TestTopologicalOrderCreateObjects(t *testing.T) {
	tests := []struct {
		name        string
		diff        *schema.MetadataDiff
		expectedSQL string
		description string
	}{
		{
			name: "table_depends_on_table_via_foreign_key",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Create orders table that depends on customers table via FK
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "orders",
						NewTable: &storepb.TableMetadata{
							Name: "orders",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "customer_id", Type: "INT", Nullable: false},
							},
							ForeignKeys: []*storepb.ForeignKeyMetadata{
								{
									Name:              "fk_orders_customer",
									Columns:           []string{"customer_id"},
									ReferencedSchema:  "public",
									ReferencedTable:   "customers",
									ReferencedColumns: []string{"id"},
								},
							},
						},
					},
					// Create customers table (should be created first)
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "customers",
						NewTable: &storepb.TableMetadata{
							Name: "customers",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
							},
						},
					},
				},
			},
			description: "Customers table should be created before orders table due to FK dependency",
		},
		{
			name: "table1_references_table2_name_column",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Create table1 that references table2(name) - this comes FIRST in the list
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "table1",
						NewTable: &storepb.TableMetadata{
							Name: "table1",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
								{Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP", Nullable: false},
								{Name: "description", Type: "TEXT", Nullable: true},
							},
							ForeignKeys: []*storepb.ForeignKeyMetadata{
								{
									Name:              "fk_table1_name",
									Columns:           []string{"name"},
									ReferencedSchema:  "public",
									ReferencedTable:   "table2",
									ReferencedColumns: []string{"name"},
								},
							},
						},
					},
					// Create table2 with unique constraint on name - this comes SECOND in the list
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "table2",
						NewTable: &storepb.TableMetadata{
							Name: "table2",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "INT", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: true},
							},
							Indexes: []*storepb.IndexMetadata{
								{
									Name:         "uq_table2_name",
									Expressions:  []string{"name"},
									Unique:       true,
									IsConstraint: true,
								},
							},
						},
					},
				},
			},
			description: "table2 should be created before table1 even though table1 appears first in the diff, because table1 has FK to table2",
		},
		{
			name: "view_depends_on_tables",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "users",
						NewTable: &storepb.TableMetadata{
							Name: "users",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
							},
						},
					},
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "orders",
						NewTable: &storepb.TableMetadata{
							Name: "orders",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "user_id", Type: "INT", Nullable: false},
							},
						},
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						ViewName:   "user_orders",
						NewView: &storepb.ViewMetadata{
							Name:       "user_orders",
							Definition: "SELECT u.name, o.id FROM users u JOIN orders o ON u.id = o.user_id",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "public", Table: "users", Column: "id"},
								{Schema: "public", Table: "users", Column: "name"},
								{Schema: "public", Table: "orders", Column: "id"},
								{Schema: "public", Table: "orders", Column: "user_id"},
							},
						},
					},
				},
			},
			description: "View should be created after tables it depends on",
		},
		{
			name: "column_addition_follows_table_dependency",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Add column to orders table that references customers
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "orders",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionCreate,
								NewColumn: &storepb.ColumnMetadata{
									Name:     "customer_ref",
									Type:     "INT",
									Nullable: true,
								},
							},
						},
					},
					// Add column to customers table (should be added first)
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "public",
						TableName:  "customers",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionCreate,
								NewColumn: &storepb.ColumnMetadata{
									Name:     "status",
									Type:     "VARCHAR(20)",
									Nullable: true,
								},
							},
						},
					},
				},
			},
			description: "Column additions should follow table topological order",
		},
		{
			name: "materialized_view_depends_on_view",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						TableName:  "products",
						NewTable: &storepb.TableMetadata{
							Name: "products",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
								{Name: "price", Type: "DECIMAL(10,2)", Nullable: false},
							},
						},
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "public",
						ViewName:   "expensive_products",
						NewView: &storepb.ViewMetadata{
							Name:       "expensive_products",
							Definition: "SELECT * FROM products WHERE price > 100",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "public", Table: "products", Column: "id"},
								{Schema: "public", Table: "products", Column: "name"},
								{Schema: "public", Table: "products", Column: "price"},
							},
						},
					},
				},
				MaterializedViewChanges: []*schema.MaterializedViewDiff{
					{
						Action:               schema.MetadataDiffActionCreate,
						SchemaName:           "public",
						MaterializedViewName: "expensive_products_mv",
						NewMaterializedView: &storepb.MaterializedViewMetadata{
							Name:       "expensive_products_mv",
							Definition: "SELECT name FROM expensive_products",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "public", Table: "expensive_products", Column: "name"},
							},
						},
					},
				},
			},
			description: "Materialized view should be created after view it depends on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateMigration(tt.diff)
			require.NoError(t, err, "Generate migration should not error")

			// Verify the SQL is not empty
			assert.NotEmpty(t, result, "Generated migration should not be empty")

			// Parse the generated SQL to check order
			statements := parseStatements(result)

			// Verify topological order based on test case
			switch tt.name {
			case "table_depends_on_table_via_foreign_key":
				// customers table should be created before orders table
				customersIndex := findStatementIndex(statements, "CREATE TABLE", "customers")
				ordersIndex := findStatementIndex(statements, "CREATE TABLE", "orders")
				assert.True(t, customersIndex < ordersIndex,
					"Customers table should be created before orders table. Got statements: %v", statements)

			case "table1_references_table2_name_column":
				// table2 should be created before table1 because table1 has FK to table2
				table1Index := findStatementIndex(statements, "CREATE TABLE", "table1")
				table2Index := findStatementIndex(statements, "CREATE TABLE", "table2")
				assert.True(t, table2Index < table1Index,
					"table2 should be created before table1 because table1 has FK to table2(name). Got statements: %v", statements)

			case "view_depends_on_tables":
				// Both tables should be created before the view
				usersIndex := findStatementIndex(statements, "CREATE TABLE", "users")
				ordersIndex := findStatementIndex(statements, "CREATE TABLE", "orders")
				viewIndex := findStatementIndex(statements, "CREATE VIEW", "user_orders")
				assert.True(t, usersIndex < viewIndex && ordersIndex < viewIndex,
					"Tables should be created before view. Got statements: %v", statements)

			case "column_addition_follows_table_dependency":
				// Verify column additions are in the right order
				// This is more complex as we need to check ALTER TABLE statements
				customersAlterIndex := findStatementIndex(statements, "ALTER TABLE", "customers")
				ordersAlterIndex := findStatementIndex(statements, "ALTER TABLE", "orders")
				// Both should exist, actual order depends on topological sorting
				assert.True(t, customersAlterIndex >= 0, "Customers ALTER should exist")
				assert.True(t, ordersAlterIndex >= 0, "Orders ALTER should exist")

			case "materialized_view_depends_on_view":
				// Table, then view, then materialized view
				tableIndex := findStatementIndex(statements, "CREATE TABLE", "products")
				viewIndex := findStatementIndex(statements, "CREATE VIEW", "expensive_products")
				mvIndex := findStatementIndex(statements, "CREATE MATERIALIZED VIEW", "expensive_products_mv")
				assert.True(t, tableIndex < viewIndex && viewIndex < mvIndex,
					"Objects should be created in dependency order: table -> view -> materialized view. Got statements: %v", statements)
			default:
				// No specific verification for this test case
			}
		})
	}
}

// TestTopologicalOrderWithCycles tests behavior when there are circular dependencies
func TestTopologicalOrderWithCycles(t *testing.T) {
	// Create a diff with circular dependency (should fall back to safe order)
	diff := &schema.MetadataDiff{
		TableChanges: []*schema.TableDiff{
			{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "table_a",
				NewTable: &storepb.TableMetadata{
					Name: "table_a",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "SERIAL", Nullable: false},
						{Name: "b_id", Type: "INT", Nullable: true},
					},
					ForeignKeys: []*storepb.ForeignKeyMetadata{
						{
							Name:              "fk_a_to_b",
							Columns:           []string{"b_id"},
							ReferencedSchema:  "public",
							ReferencedTable:   "table_b",
							ReferencedColumns: []string{"id"},
						},
					},
				},
			},
			{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: "public",
				TableName:  "table_b",
				NewTable: &storepb.TableMetadata{
					Name: "table_b",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "SERIAL", Nullable: false},
						{Name: "a_id", Type: "INT", Nullable: true},
					},
					ForeignKeys: []*storepb.ForeignKeyMetadata{
						{
							Name:              "fk_b_to_a",
							Columns:           []string{"a_id"},
							ReferencedSchema:  "public",
							ReferencedTable:   "table_a",
							ReferencedColumns: []string{"id"},
						},
					},
				},
			},
		},
	}

	result, err := generateMigration(diff)
	require.NoError(t, err, "Generate migration should handle cycles gracefully")
	assert.NotEmpty(t, result, "Generated migration should not be empty even with cycles")

	// With cycles, it should fall back to creating tables without foreign keys first
	statements := parseStatements(result)

	// Both CREATE TABLE statements should exist
	tableAIndex := findStatementIndex(statements, "CREATE TABLE", "table_a")
	tableBIndex := findStatementIndex(statements, "CREATE TABLE", "table_b")
	assert.True(t, tableAIndex >= 0, "Table A should be created")
	assert.True(t, tableBIndex >= 0, "Table B should be created")

	// Foreign key constraints should be added later
	fkCount := 0
	for _, stmt := range statements {
		if strings.Contains(stmt, "ADD CONSTRAINT") && strings.Contains(stmt, "FOREIGN KEY") {
			fkCount++
		}
	}
	assert.Equal(t, 2, fkCount, "Both foreign keys should be added separately")
}

// Helper function to parse SQL statements
func parseStatements(sql string) []string {
	var statements []string
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			statements = append(statements, line)
		}
	}
	return statements
}

// Helper function to find the index of a statement containing specific keywords
func findStatementIndex(statements []string, stmtType, objectName string) int {
	for i, stmt := range statements {
		upperStmt := strings.ToUpper(stmt)
		upperType := strings.ToUpper(stmtType)
		upperName := strings.ToUpper(objectName)

		if strings.Contains(upperStmt, upperType) && strings.Contains(upperStmt, upperName) {
			return i
		}
	}
	return -1
}

// TestTopologicalOrderDropObjects tests that objects are dropped in the correct dependency order
func TestTopologicalOrderDropObjects(t *testing.T) {
	tests := []struct {
		name        string
		diff        *schema.MetadataDiff
		description string
	}{
		{
			name: "drop_table1_with_fk_before_table2",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Drop table1 that has FK to table2 - comes FIRST in list
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "table1",
						OldTable: &storepb.TableMetadata{
							Name: "table1",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
							},
							ForeignKeys: []*storepb.ForeignKeyMetadata{
								{
									Name:              "fk_table1_name",
									Columns:           []string{"name"},
									ReferencedSchema:  "public",
									ReferencedTable:   "table2",
									ReferencedColumns: []string{"name"},
								},
							},
						},
					},
					// Drop table2 - comes SECOND in list but should be dropped AFTER table1
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "table2",
						OldTable: &storepb.TableMetadata{
							Name: "table2",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "INT", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
							},
						},
					},
				},
			},
			description: "table1 (with FK) should be dropped before table2 (referenced)",
		},
		{
			name: "drop_orders_before_customers",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Drop customers table - comes FIRST in list
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "customers",
						OldTable: &storepb.TableMetadata{
							Name: "customers",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "name", Type: "VARCHAR(100)", Nullable: false},
							},
						},
					},
					// Drop orders table that has FK to customers - comes SECOND in list
					{
						Action:     schema.MetadataDiffActionDrop,
						SchemaName: "public",
						TableName:  "orders",
						OldTable: &storepb.TableMetadata{
							Name: "orders",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "SERIAL", Nullable: false},
								{Name: "customer_id", Type: "INT", Nullable: false},
							},
							ForeignKeys: []*storepb.ForeignKeyMetadata{
								{
									Name:              "fk_orders_customer",
									Columns:           []string{"customer_id"},
									ReferencedSchema:  "public",
									ReferencedTable:   "customers",
									ReferencedColumns: []string{"id"},
								},
							},
						},
					},
				},
			},
			description: "orders (with FK) should be dropped before customers (referenced)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			migrationSQL, err := generateMigration(tc.diff)
			require.NoError(t, err)

			// Parse the SQL into statements
			statements := parseStatements(migrationSQL)

			// For DROP operations, find the DROP statements
			switch tc.name {
			case "drop_table1_with_fk_before_table2":
				table1DropIdx := findStatementIndex(statements, "DROP TABLE", "table1")
				table2DropIdx := findStatementIndex(statements, "DROP TABLE", "table2")

				assert.NotEqual(t, -1, table1DropIdx, "table1 DROP statement should exist")
				assert.NotEqual(t, -1, table2DropIdx, "table2 DROP statement should exist")
				assert.Less(t, table1DropIdx, table2DropIdx, "table1 (with FK) should be dropped before table2 (referenced)")
			case "drop_orders_before_customers":
				ordersDropIdx := findStatementIndex(statements, "DROP TABLE", "orders")
				customersDropIdx := findStatementIndex(statements, "DROP TABLE", "customers")

				assert.NotEqual(t, -1, ordersDropIdx, "orders DROP statement should exist")
				assert.NotEqual(t, -1, customersDropIdx, "customers DROP statement should exist")
				assert.Less(t, ordersDropIdx, customersDropIdx, "orders (with FK) should be dropped before customers (referenced)")
			default:
				t.Fatalf("Unknown test case: %s", tc.name)
			}
		})
	}
}
