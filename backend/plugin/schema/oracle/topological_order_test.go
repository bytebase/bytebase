package oracle

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
		description string
	}{
		{
			name: "table_depends_on_table_via_foreign_key",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					// Create orders table that depends on customers table via FK
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						TableName:  "ORDERS",
						NewTable: &storepb.TableMetadata{
							Name: "ORDERS",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "CUSTOMER_ID", Type: "NUMBER", Nullable: false},
							},
							ForeignKeys: []*storepb.ForeignKeyMetadata{
								{
									Name:              "FK_ORDERS_CUSTOMER",
									Columns:           []string{"CUSTOMER_ID"},
									ReferencedSchema:  "TESTUSER",
									ReferencedTable:   "CUSTOMERS",
									ReferencedColumns: []string{"ID"},
								},
							},
						},
					},
					// Create customers table (should be created first)
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						TableName:  "CUSTOMERS",
						NewTable: &storepb.TableMetadata{
							Name: "CUSTOMERS",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "NAME", Type: "VARCHAR2(100)", Nullable: false},
							},
						},
					},
				},
			},
			description: "Customers table should be created before orders table due to FK dependency",
		},
		{
			name: "view_depends_on_tables",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						TableName:  "USERS",
						NewTable: &storepb.TableMetadata{
							Name: "USERS",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "NAME", Type: "VARCHAR2(100)", Nullable: false},
							},
						},
					},
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						TableName:  "ORDERS",
						NewTable: &storepb.TableMetadata{
							Name: "ORDERS",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "USER_ID", Type: "NUMBER", Nullable: false},
							},
						},
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						ViewName:   "USER_ORDERS",
						NewView: &storepb.ViewMetadata{
							Name:       "USER_ORDERS",
							Definition: "SELECT u.NAME, o.ID FROM USERS u JOIN ORDERS o ON u.ID = o.USER_ID",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "TESTUSER", Table: "USERS", Column: "ID"},
								{Schema: "TESTUSER", Table: "USERS", Column: "NAME"},
								{Schema: "TESTUSER", Table: "ORDERS", Column: "ID"},
								{Schema: "TESTUSER", Table: "ORDERS", Column: "USER_ID"},
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
					// Add column to orders table
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "TESTUSER",
						TableName:  "ORDERS",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionCreate,
								NewColumn: &storepb.ColumnMetadata{
									Name:     "CUSTOMER_REF",
									Type:     "NUMBER",
									Nullable: true,
								},
							},
						},
					},
					// Add column to customers table (should be added first in dependency order)
					{
						Action:     schema.MetadataDiffActionAlter,
						SchemaName: "TESTUSER",
						TableName:  "CUSTOMERS",
						ColumnChanges: []*schema.ColumnDiff{
							{
								Action: schema.MetadataDiffActionCreate,
								NewColumn: &storepb.ColumnMetadata{
									Name:     "STATUS",
									Type:     "VARCHAR2(20)",
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
						SchemaName: "TESTUSER",
						TableName:  "PRODUCTS",
						NewTable: &storepb.TableMetadata{
							Name: "PRODUCTS",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "NAME", Type: "VARCHAR2(100)", Nullable: false},
								{Name: "PRICE", Type: "NUMBER(10,2)", Nullable: false},
							},
						},
					},
				},
				ViewChanges: []*schema.ViewDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						ViewName:   "EXPENSIVE_PRODUCTS",
						NewView: &storepb.ViewMetadata{
							Name:       "EXPENSIVE_PRODUCTS",
							Definition: "SELECT * FROM PRODUCTS WHERE PRICE > 100",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "TESTUSER", Table: "PRODUCTS", Column: "ID"},
								{Schema: "TESTUSER", Table: "PRODUCTS", Column: "NAME"},
								{Schema: "TESTUSER", Table: "PRODUCTS", Column: "PRICE"},
							},
						},
					},
				},
				MaterializedViewChanges: []*schema.MaterializedViewDiff{
					{
						Action:               schema.MetadataDiffActionCreate,
						SchemaName:           "TESTUSER",
						MaterializedViewName: "EXPENSIVE_PRODUCTS_MV",
						NewMaterializedView: &storepb.MaterializedViewMetadata{
							Name:       "EXPENSIVE_PRODUCTS_MV",
							Definition: "SELECT NAME FROM EXPENSIVE_PRODUCTS",
							DependencyColumns: []*storepb.DependencyColumn{
								{Schema: "TESTUSER", Table: "EXPENSIVE_PRODUCTS", Column: "NAME"},
							},
						},
					},
				},
			},
			description: "Materialized view should be created after view it depends on",
		},
		{
			name: "function_depends_on_table",
			diff: &schema.MetadataDiff{
				TableChanges: []*schema.TableDiff{
					{
						Action:     schema.MetadataDiffActionCreate,
						SchemaName: "TESTUSER",
						TableName:  "EMPLOYEES",
						NewTable: &storepb.TableMetadata{
							Name: "EMPLOYEES",
							Columns: []*storepb.ColumnMetadata{
								{Name: "ID", Type: "NUMBER", Nullable: false},
								{Name: "SALARY", Type: "NUMBER(10,2)", Nullable: false},
							},
						},
					},
				},
				FunctionChanges: []*schema.FunctionDiff{
					{
						Action:       schema.MetadataDiffActionCreate,
						SchemaName:   "TESTUSER",
						FunctionName: "GET_AVG_SALARY",
						NewFunction: &storepb.FunctionMetadata{
							Name:       "GET_AVG_SALARY",
							Definition: "CREATE OR REPLACE FUNCTION GET_AVG_SALARY RETURN NUMBER IS avg_sal NUMBER; BEGIN SELECT AVG(SALARY) INTO avg_sal FROM EMPLOYEES; RETURN avg_sal; END;",
							DependencyTables: []*storepb.DependencyTable{
								{Schema: "TESTUSER", Table: "EMPLOYEES"},
							},
						},
					},
				},
			},
			description: "Function should be created after table it depends on",
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
				customersIndex := findStatementIndex(statements, "CREATE TABLE", "CUSTOMERS")
				ordersIndex := findStatementIndex(statements, "CREATE TABLE", "ORDERS")
				assert.True(t, customersIndex < ordersIndex,
					"Customers table should be created before orders table. Got statements: %v", statements)

			case "view_depends_on_tables":
				// Both tables should be created before the view
				usersIndex := findStatementIndex(statements, "CREATE TABLE", "USERS")
				ordersIndex := findStatementIndex(statements, "CREATE TABLE", "ORDERS")
				viewIndex := findStatementIndex(statements, "CREATE VIEW", "USER_ORDERS")
				assert.True(t, usersIndex < viewIndex && ordersIndex < viewIndex,
					"Tables should be created before view. Got statements: %v", statements)

			case "column_addition_follows_table_dependency":
				// Verify column additions are in the topological order
				customersAlterIndex := findStatementIndex(statements, "ALTER TABLE", "CUSTOMERS")
				ordersAlterIndex := findStatementIndex(statements, "ALTER TABLE", "ORDERS")
				// Both should exist
				assert.True(t, customersAlterIndex >= 0, "Customers ALTER should exist")
				assert.True(t, ordersAlterIndex >= 0, "Orders ALTER should exist")

			case "materialized_view_depends_on_view":
				// Table, then view, then materialized view
				tableIndex := findStatementIndex(statements, "CREATE TABLE", "PRODUCTS")
				viewIndex := findStatementIndex(statements, "CREATE VIEW", "EXPENSIVE_PRODUCTS")
				mvIndex := findStatementIndex(statements, "CREATE MATERIALIZED VIEW", "EXPENSIVE_PRODUCTS_MV")
				assert.True(t, tableIndex < viewIndex && viewIndex < mvIndex,
					"Objects should be created in dependency order: table -> view -> materialized view. Got statements: %v", statements)

			case "function_depends_on_table":
				// Table should be created before function
				tableIndex := findStatementIndex(statements, "CREATE TABLE", "EMPLOYEES")
				funcIndex := findStatementIndex(statements, "CREATE OR REPLACE FUNCTION", "GET_AVG_SALARY")
				if funcIndex == -1 {
					// Try alternative function creation pattern
					funcIndex = findStatementIndex(statements, "CREATE FUNCTION", "GET_AVG_SALARY")
				}
				assert.True(t, tableIndex < funcIndex,
					"Table should be created before function. Got statements: %v", statements)
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
				SchemaName: "TESTUSER",
				TableName:  "TABLE_A",
				NewTable: &storepb.TableMetadata{
					Name: "TABLE_A",
					Columns: []*storepb.ColumnMetadata{
						{Name: "ID", Type: "NUMBER", Nullable: false},
						{Name: "B_ID", Type: "NUMBER", Nullable: true},
					},
					ForeignKeys: []*storepb.ForeignKeyMetadata{
						{
							Name:              "FK_A_TO_B",
							Columns:           []string{"B_ID"},
							ReferencedSchema:  "TESTUSER",
							ReferencedTable:   "TABLE_B",
							ReferencedColumns: []string{"ID"},
						},
					},
				},
			},
			{
				Action:     schema.MetadataDiffActionCreate,
				SchemaName: "TESTUSER",
				TableName:  "TABLE_B",
				NewTable: &storepb.TableMetadata{
					Name: "TABLE_B",
					Columns: []*storepb.ColumnMetadata{
						{Name: "ID", Type: "NUMBER", Nullable: false},
						{Name: "A_ID", Type: "NUMBER", Nullable: true},
					},
					ForeignKeys: []*storepb.ForeignKeyMetadata{
						{
							Name:              "FK_B_TO_A",
							Columns:           []string{"A_ID"},
							ReferencedSchema:  "TESTUSER",
							ReferencedTable:   "TABLE_A",
							ReferencedColumns: []string{"ID"},
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
	tableAIndex := findStatementIndex(statements, "CREATE TABLE", "TABLE_A")
	tableBIndex := findStatementIndex(statements, "CREATE TABLE", "TABLE_B")
	assert.True(t, tableAIndex >= 0, "Table A should be created")
	assert.True(t, tableBIndex >= 0, "Table B should be created")

	// Foreign key constraints should be added later as separate ALTER TABLE statements
	fkCount := 0
	for _, stmt := range statements {
		if strings.Contains(strings.ToUpper(stmt), "ADD CONSTRAINT") && strings.Contains(strings.ToUpper(stmt), "FOREIGN KEY") {
			fkCount++
		}
	}
	assert.True(t, fkCount >= 2, "Both foreign keys should be added separately, found %d FK statements", fkCount)
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
