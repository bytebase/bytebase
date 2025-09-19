package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestColumnSDLDiff provides comprehensive testing for column-related SDL differences
func TestColumnSDLDiff(t *testing.T) {
	testCases := []struct {
		name                  string
		currentSDL            string
		previousSDL           string
		expectedTableChanges  int
		expectedColumnChanges int
		expectedCreateColumns []string
		expectedAlterColumns  []string
		expectedDropColumns   []string
		validateColumnDiffs   func(t *testing.T, columnDiffs []*schema.ColumnDiff)
		validateTableDiffs    func(t *testing.T, tableDiffs []*schema.TableDiff)
	}{
		{
			name: "No column changes - identical tables",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedTableChanges:  0,
			expectedColumnChanges: 0,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				assert.Len(t, columnDiffs, 0, "Should have no column changes for identical tables")
			},
		},
		{
			name: "Add new column",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(100)
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{"email"},
			expectedAlterColumns:  []string{},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, columnDiff.Action, "Should be CREATE action")
				assert.Nil(t, columnDiff.OldColumn, "CREATE should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "CREATE should have nil NewColumn in AST-only mode")

				// Verify AST nodes
				assert.Nil(t, columnDiff.OldASTNode, "CREATE should have nil OldASTNode")
				assert.NotNil(t, columnDiff.NewASTNode, "CREATE should have NewASTNode")

				// Extract column name from AST for verification
				columnName := extractColumnNameFromAST(columnDiff.NewASTNode)
				assert.Equal(t, "email", columnName, "Column name should match")
			},
		},
		{
			name: "Drop column",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(100)
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{},
			expectedDropColumns:   []string{"email"},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionDrop, columnDiff.Action, "Should be DROP action")
				assert.Nil(t, columnDiff.OldColumn, "DROP should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "DROP should have nil NewColumn")

				// Verify AST nodes
				assert.NotNil(t, columnDiff.OldASTNode, "DROP should have OldASTNode")
				assert.Nil(t, columnDiff.NewASTNode, "DROP should have nil NewASTNode")

				// Extract column name from AST for verification
				columnName := extractColumnNameFromAST(columnDiff.OldASTNode)
				assert.Equal(t, "email", columnName, "Column name should match")
			},
		},
		{
			name: "Alter column type",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{"name"},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, columnDiff.Action, "Should be ALTER action")
				assert.Nil(t, columnDiff.OldColumn, "ALTER should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "ALTER should have nil NewColumn in AST-only mode")

				// Verify AST nodes
				assert.NotNil(t, columnDiff.OldASTNode, "ALTER should have OldASTNode")
				assert.NotNil(t, columnDiff.NewASTNode, "ALTER should have NewASTNode")

				// Extract column names from AST for verification
				oldColumnName := extractColumnNameFromAST(columnDiff.OldASTNode)
				newColumnName := extractColumnNameFromAST(columnDiff.NewASTNode)
				assert.Equal(t, "name", oldColumnName, "Old column name should match")
				assert.Equal(t, "name", newColumnName, "New column name should match")

				// Verify AST text extraction
				oldText := getColumnText(columnDiff.OldASTNode)
				newText := getColumnText(columnDiff.NewASTNode)
				assert.Contains(t, oldText, "VARCHAR(255)", "Old AST should contain VARCHAR(255)")
				assert.Contains(t, newText, "TEXT", "New AST should contain TEXT")
			},
		},
		{
			name: "Multiple column changes",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				email VARCHAR(200),
				created_at TIMESTAMP
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				phone VARCHAR(20),
				updated_at TIMESTAMP
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 5,
			expectedCreateColumns: []string{"email", "created_at"},
			expectedAlterColumns:  []string{"name"},
			expectedDropColumns:   []string{"phone", "updated_at"},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 5, "Should have exactly 5 column changes")

				// Count each action type
				createCount := 0
				alterCount := 0
				dropCount := 0

				createColumns := []string{}
				alterColumns := []string{}
				dropColumns := []string{}

				for _, columnDiff := range columnDiffs {
					switch columnDiff.Action {
					case schema.MetadataDiffActionCreate:
						createCount++
						columnName := extractColumnNameFromAST(columnDiff.NewASTNode)
						createColumns = append(createColumns, columnName)
						assert.Nil(t, columnDiff.OldColumn, "CREATE should have nil OldColumn in AST-only mode")
						assert.Nil(t, columnDiff.NewColumn, "CREATE should have nil NewColumn in AST-only mode")
						assert.Nil(t, columnDiff.OldASTNode, "CREATE should have nil OldASTNode")
						assert.NotNil(t, columnDiff.NewASTNode, "CREATE should have NewASTNode")
					case schema.MetadataDiffActionAlter:
						alterCount++
						columnName := extractColumnNameFromAST(columnDiff.NewASTNode)
						alterColumns = append(alterColumns, columnName)
						assert.Nil(t, columnDiff.OldColumn, "ALTER should have nil OldColumn in AST-only mode")
						assert.Nil(t, columnDiff.NewColumn, "ALTER should have nil NewColumn in AST-only mode")
						assert.NotNil(t, columnDiff.OldASTNode, "ALTER should have OldASTNode")
						assert.NotNil(t, columnDiff.NewASTNode, "ALTER should have NewASTNode")
					case schema.MetadataDiffActionDrop:
						dropCount++
						columnName := extractColumnNameFromAST(columnDiff.OldASTNode)
						dropColumns = append(dropColumns, columnName)
						assert.Nil(t, columnDiff.OldColumn, "DROP should have nil OldColumn in AST-only mode")
						assert.Nil(t, columnDiff.NewColumn, "DROP should have nil NewColumn")
						assert.NotNil(t, columnDiff.OldASTNode, "DROP should have OldASTNode")
						assert.Nil(t, columnDiff.NewASTNode, "DROP should have nil NewASTNode")
					default:
						t.Fatalf("Unexpected column diff action: %v", columnDiff.Action)
					}
				}

				assert.Equal(t, 2, createCount, "Should have 2 CREATE actions")
				assert.Equal(t, 1, alterCount, "Should have 1 ALTER action")
				assert.Equal(t, 2, dropCount, "Should have 2 DROP actions")

				assert.ElementsMatch(t, []string{"email", "created_at"}, createColumns, "CREATE columns should match")
				assert.ElementsMatch(t, []string{"name"}, alterColumns, "ALTER columns should match")
				assert.ElementsMatch(t, []string{"phone", "updated_at"}, dropColumns, "DROP columns should match")
			},
		},
		{
			name: "Column constraint changes",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255)
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{"name"},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, columnDiff.Action, "Should be ALTER action")
				assert.Nil(t, columnDiff.OldColumn, "Should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "Should have nil NewColumn in AST-only mode")

				// Verify AST nodes are available
				assert.NotNil(t, columnDiff.OldASTNode, "Should have OldASTNode")
				assert.NotNil(t, columnDiff.NewASTNode, "Should have NewASTNode")

				// Check constraint differences via AST text
				oldText := getColumnText(columnDiff.OldASTNode)
				newText := getColumnText(columnDiff.NewASTNode)
				assert.Contains(t, oldText, "NOT NULL", "Old AST should contain NOT NULL")
				assert.NotContains(t, newText, "NOT NULL", "New AST should not contain NOT NULL")
			},
		},
		{
			name: "Column default value changes",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50) DEFAULT 'inactive'
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50) DEFAULT 'active'
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{"status"},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, columnDiff.Action, "Should be ALTER action")
				assert.Nil(t, columnDiff.OldColumn, "Should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "Should have nil NewColumn in AST-only mode")

				// Verify AST nodes are available
				assert.NotNil(t, columnDiff.OldASTNode, "Should have OldASTNode")
				assert.NotNil(t, columnDiff.NewASTNode, "Should have NewASTNode")

				// Check default value differences via AST text
				oldText := getColumnText(columnDiff.OldASTNode)
				newText := getColumnText(columnDiff.NewASTNode)
				assert.Contains(t, oldText, "'active'", "Old AST should contain 'active'")
				assert.Contains(t, newText, "'inactive'", "New AST should contain 'inactive'")
			},
		},
		{
			name: "Column collation changes",
			currentSDL: `CREATE TABLE users (
				name VARCHAR(255) COLLATE "en_US"
			);`,
			previousSDL: `CREATE TABLE users (
				name VARCHAR(255) COLLATE "C"
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{"name"},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should have exactly one column change")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, columnDiff.Action, "Should be ALTER action")
				assert.Nil(t, columnDiff.OldColumn, "Should have nil OldColumn in AST-only mode")
				assert.Nil(t, columnDiff.NewColumn, "Should have nil NewColumn in AST-only mode")

				// Verify AST nodes are available
				assert.NotNil(t, columnDiff.OldASTNode, "Should have OldASTNode")
				assert.NotNil(t, columnDiff.NewASTNode, "Should have NewASTNode")

				// Check collation differences via AST text
				oldText := getColumnText(columnDiff.OldASTNode)
				newText := getColumnText(columnDiff.NewASTNode)
				assert.Contains(t, oldText, `"C"`, "Old AST should contain 'C' collation")
				assert.Contains(t, newText, `"en_US"`, "New AST should contain 'en_US' collation")
			},
		},
		{
			name: "Whitespace differences detected by text comparison",
			currentSDL: `CREATE TABLE users (
				id    SERIAL    PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedTableChanges:  1,
			expectedColumnChanges: 1,
			expectedCreateColumns: []string{},
			expectedAlterColumns:  []string{"id"},
			expectedDropColumns:   []string{},
			validateColumnDiffs: func(t *testing.T, columnDiffs []*schema.ColumnDiff) {
				require.Len(t, columnDiffs, 1, "Should detect whitespace difference")

				columnDiff := columnDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, columnDiff.Action, "Should be ALTER action")
				columnName := extractColumnNameFromAST(columnDiff.NewASTNode)
				assert.Equal(t, "id", columnName, "Column name should match")

				// Verify that text comparison detects the difference
				oldText := getColumnText(columnDiff.OldASTNode)
				newText := getColumnText(columnDiff.NewASTNode)
				assert.NotEqual(t, oldText, newText, "AST text should be different due to whitespace")
			},
		},
		{
			name: "Add foreign key constraint",
			currentSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER,
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
			);`,
			previousSDL: `CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER
			);`,
			expectedTableChanges: 1,
			validateTableDiffs: func(t *testing.T, tableDiffs []*schema.TableDiff) {
				assert.Len(t, tableDiffs, 1, "Should have 1 table diff")
				tableDiff := tableDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				assert.Len(t, tableDiff.ForeignKeyChanges, 1, "Should have 1 FK change")
				fkChange := tableDiff.ForeignKeyChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, fkChange.Action)
				// AST node should be present for created FK
				assert.NotNil(t, fkChange.NewASTNode)
				assert.Nil(t, fkChange.OldASTNode)
			},
		},
		{
			name: "Add check constraint",
			currentSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_positive_price CHECK (price > 0)
			);`,
			previousSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2)
			);`,
			expectedTableChanges: 1,
			validateTableDiffs: func(t *testing.T, tableDiffs []*schema.TableDiff) {
				assert.Len(t, tableDiffs, 1, "Should have 1 table diff")
				tableDiff := tableDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				assert.Len(t, tableDiff.CheckConstraintChanges, 1, "Should have 1 check constraint change")
				checkChange := tableDiff.CheckConstraintChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, checkChange.Action)
				// AST node should be present for created check constraint
				assert.NotNil(t, checkChange.NewASTNode)
				assert.Nil(t, checkChange.OldASTNode)
			},
		},
		{
			name: "Add unique constraint",
			currentSDL: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				email VARCHAR(255),
				CONSTRAINT uk_users_email UNIQUE (email)
			);`,
			previousSDL: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				email VARCHAR(255)
			);`,
			expectedTableChanges: 1,
			validateTableDiffs: func(t *testing.T, tableDiffs []*schema.TableDiff) {
				assert.Len(t, tableDiffs, 1, "Should have 1 table diff")
				tableDiff := tableDiffs[0]
				assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)
				assert.Len(t, tableDiff.UniqueConstraintChanges, 1, "Should have 1 unique constraint change")
				uniqueChange := tableDiff.UniqueConstraintChanges[0]
				assert.Equal(t, schema.MetadataDiffActionCreate, uniqueChange.Action)
				// AST node should be present for created unique constraint
				assert.NotNil(t, uniqueChange.NewASTNode)
				assert.Nil(t, uniqueChange.OldASTNode)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call GetSDLDiff
			diff, err := GetSDLDiff(tc.currentSDL, tc.previousSDL, nil, nil)
			require.NoError(t, err, "GetSDLDiff should not return error")

			// Validate table changes count
			assert.Len(t, diff.TableChanges, tc.expectedTableChanges,
				"Expected %d table changes, got %d", tc.expectedTableChanges, len(diff.TableChanges))

			if tc.expectedTableChanges == 0 {
				return // No further validation needed
			}

			// Should have exactly one table change
			require.Len(t, diff.TableChanges, 1, "Should have exactly one table change")
			tableDiff := diff.TableChanges[0]

			// Validate column changes count
			assert.Len(t, tableDiff.ColumnChanges, tc.expectedColumnChanges,
				"Expected %d column changes, got %d", tc.expectedColumnChanges, len(tableDiff.ColumnChanges))

			// Validate column changes by action type
			if tc.expectedColumnChanges > 0 {
				createColumns := []string{}
				alterColumns := []string{}
				dropColumns := []string{}

				for _, columnChange := range tableDiff.ColumnChanges {
					switch columnChange.Action {
					case schema.MetadataDiffActionCreate:
						if columnChange.NewColumn != nil {
							createColumns = append(createColumns, columnChange.NewColumn.Name)
						} else if columnChange.NewASTNode != nil {
							// AST-only mode: extract column name from AST
							columnName := extractColumnNameFromAST(columnChange.NewASTNode)
							createColumns = append(createColumns, columnName)
						}
					case schema.MetadataDiffActionAlter:
						if columnChange.NewColumn != nil {
							alterColumns = append(alterColumns, columnChange.NewColumn.Name)
						} else if columnChange.NewASTNode != nil {
							// AST-only mode: extract column name from AST
							columnName := extractColumnNameFromAST(columnChange.NewASTNode)
							alterColumns = append(alterColumns, columnName)
						}
					case schema.MetadataDiffActionDrop:
						if columnChange.OldColumn != nil {
							dropColumns = append(dropColumns, columnChange.OldColumn.Name)
						} else if columnChange.OldASTNode != nil {
							// AST-only mode: extract column name from AST
							columnName := extractColumnNameFromAST(columnChange.OldASTNode)
							dropColumns = append(dropColumns, columnName)
						}
					default:
						t.Fatalf("Unexpected column change action: %v", columnChange.Action)
					}
				}

				// Verify expected columns
				assert.ElementsMatch(t, tc.expectedCreateColumns, createColumns, "CREATE columns mismatch")
				assert.ElementsMatch(t, tc.expectedAlterColumns, alterColumns, "ALTER columns mismatch")
				assert.ElementsMatch(t, tc.expectedDropColumns, dropColumns, "DROP columns mismatch")

				// Run custom validation
				if tc.validateColumnDiffs != nil {
					tc.validateColumnDiffs(t, tableDiff.ColumnChanges)
				}
			}

			// Run table-level validation
			if tc.validateTableDiffs != nil {
				tc.validateTableDiffs(t, diff.TableChanges)
			}
		})
	}
}

// TestColumnMetadataExtraction tests all column metadata fields extraction
func TestColumnMetadataExtraction(t *testing.T) {
	testCases := []struct {
		name              string
		sdlText           string
		expectedName      string
		expectedType      string
		expectedNullable  bool
		expectedDefault   string
		expectedCollation string
		expectedComment   string
	}{
		{
			name: "Simple INTEGER column",
			sdlText: `CREATE TABLE test (
				id INTEGER
			);`,
			expectedName:      "id",
			expectedType:      "INTEGER",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "VARCHAR with length and nullable",
			sdlText: `CREATE TABLE test (
				name VARCHAR(255)
			);`,
			expectedName:      "name",
			expectedType:      "VARCHAR(255)",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "NOT NULL SERIAL column",
			sdlText: `CREATE TABLE test (
				id SERIAL NOT NULL
			);`,
			expectedName:      "id",
			expectedType:      "SERIAL",
			expectedNullable:  false,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "PRIMARY KEY column (implicitly NOT NULL)",
			sdlText: `CREATE TABLE test (
				id INTEGER PRIMARY KEY
			);`,
			expectedName:      "id",
			expectedType:      "INTEGER",
			expectedNullable:  false,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "DECIMAL with precision",
			sdlText: `CREATE TABLE test (
				price DECIMAL(10,2)
			);`,
			expectedName:      "price",
			expectedType:      "DECIMAL(10,2)",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "Column with string default value",
			sdlText: `CREATE TABLE test (
				status VARCHAR(50) DEFAULT 'active'
			);`,
			expectedName:      "status",
			expectedType:      "VARCHAR(50)",
			expectedNullable:  true,
			expectedDefault:   "'active'",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "Column with numeric default value",
			sdlText: `CREATE TABLE test (
				count INTEGER DEFAULT 0
			);`,
			expectedName:      "count",
			expectedType:      "INTEGER",
			expectedNullable:  true,
			expectedDefault:   "0",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "Column with function default value",
			sdlText: `CREATE TABLE test (
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);`,
			expectedName:      "created_at",
			expectedType:      "TIMESTAMP",
			expectedNullable:  true,
			expectedDefault:   "CURRENT_TIMESTAMP",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "NOT NULL column with default",
			sdlText: `CREATE TABLE test (
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);`,
			expectedName:      "updated_at",
			expectedType:      "TIMESTAMP",
			expectedNullable:  false,
			expectedDefault:   "CURRENT_TIMESTAMP",
			expectedCollation: "",
			expectedComment:   "",
		},
		{
			name: "Column with collation",
			sdlText: `CREATE TABLE test (
				name VARCHAR(255) COLLATE "C"
			);`,
			expectedName:      "name",
			expectedType:      "VARCHAR(255)",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: `"C"`,
			expectedComment:   "",
		},
		{
			name: "Column with unquoted collation",
			sdlText: `CREATE TABLE test (
				name VARCHAR(255) COLLATE C
			);`,
			expectedName:      "name",
			expectedType:      "VARCHAR(255)",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: "C",
			expectedComment:   "",
		},
		{
			name: "Column with schema-qualified collation",
			sdlText: `CREATE TABLE test (
				name VARCHAR(255) COLLATE pg_catalog."POSIX"
			);`,
			expectedName:      "name",
			expectedType:      "VARCHAR(255)",
			expectedNullable:  true,
			expectedDefault:   "",
			expectedCollation: `pg_catalog."POSIX"`,
			expectedComment:   "",
		},
		{
			name: "Complex column with multiple attributes",
			sdlText: `CREATE TABLE test (
				name VARCHAR(255) NOT NULL COLLATE "en_US" DEFAULT 'unknown'
			);`,
			expectedName:      "name",
			expectedType:      "VARCHAR(255)",
			expectedNullable:  false,
			expectedDefault:   "'unknown'",
			expectedCollation: `"en_US"`,
			expectedComment:   "",
		},
		{
			name: "TEXT column with collation and default",
			sdlText: `CREATE TABLE test (
				description TEXT COLLATE "en_US.UTF-8" DEFAULT 'No description'
			);`,
			expectedName:      "description",
			expectedType:      "TEXT",
			expectedNullable:  true,
			expectedDefault:   "'No description'",
			expectedCollation: `"en_US.UTF-8"`,
			expectedComment:   "",
		},
		{
			name: "BIGSERIAL PRIMARY KEY column",
			sdlText: `CREATE TABLE test (
				id BIGSERIAL PRIMARY KEY
			);`,
			expectedName:      "id",
			expectedType:      "BIGSERIAL",
			expectedNullable:  false,
			expectedDefault:   "",
			expectedCollation: "",
			expectedComment:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := ChunkSDLText(tc.sdlText)
			require.NoError(t, err)
			require.Contains(t, chunks.Tables, "test")

			testChunk := chunks.Tables["test"]
			createStmt, ok := testChunk.ASTNode.(*parser.CreatestmtContext)
			require.True(t, ok, "Should be a CreatestmtContext")

			// Extract the first column definition directly
			require.NotNil(t, createStmt.Opttableelementlist(), "Should have table element list")
			require.NotNil(t, createStmt.Opttableelementlist().Tableelementlist(), "Should have table element list")

			elements := createStmt.Opttableelementlist().Tableelementlist().AllTableelement()
			require.Len(t, elements, 1, "Should have exactly one table element")

			element := elements[0]
			require.NotNil(t, element.ColumnDef(), "Should have column definition")

			columnDef := element.ColumnDef()
			column := extractColumnMetadata(columnDef)

			require.NotNil(t, column, "Should extract column metadata")

			// Verify all column fields
			assert.Equal(t, tc.expectedName, column.Name, "Column name should match")
			assert.Equal(t, tc.expectedType, column.Type, "Column type should match")
			assert.Equal(t, tc.expectedNullable, column.Nullable, "Column nullable should match")
			assert.Equal(t, tc.expectedDefault, column.Default, "Column default should match")
			assert.Equal(t, tc.expectedCollation, column.Collation, "Column collation should match")
			assert.Equal(t, tc.expectedComment, column.Comment, "Column comment should match")
		})
	}
}
