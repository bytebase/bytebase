package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTableSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create new table",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			expectedMigration: `CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);

`,
		},
		{
			name: "Drop table",
			previousSDL: `
				CREATE TABLE old_table (
					id INTEGER PRIMARY KEY,
					data TEXT
				);
			`,
			currentSDL: ``,
			expectedMigration: `DROP TABLE IF EXISTS "public"."old_table";
`,
		},
		{
			name:        "Create table with constraints",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) CHECK (price > 0),
					category_id INTEGER,
					CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
				);
			`,
			expectedMigration: `CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) CHECK (price > 0),
					category_id INTEGER,
					CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
				);

`,
		},
		{
			name:        "Create schema-qualified table",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE test_schema.items (
					id BIGSERIAL PRIMARY KEY,
					description TEXT
				);
			`,
			expectedMigration: `CREATE TABLE test_schema.items (
					id BIGSERIAL PRIMARY KEY,
					description TEXT
				);

`,
		},
		{
			name: "Multiple tables with different operations",
			previousSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_b (id INTEGER PRIMARY KEY);
			`,
			currentSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_c (id INTEGER PRIMARY KEY);
			`,
			expectedMigration: `DROP TABLE IF EXISTS "public"."table_b";

CREATE TABLE table_c (id INTEGER PRIMARY KEY);

`,
		},
		{
			name:        "Create table with various data types",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE data_types_test (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					title VARCHAR(100),
					content TEXT,
					count INTEGER,
					amount DECIMAL(15,2),
					is_active BOOLEAN DEFAULT false,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					metadata JSONB
				);
			`,
			expectedMigration: `CREATE TABLE data_types_test (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					title VARCHAR(100),
					content TEXT,
					count INTEGER,
					amount DECIMAL(15,2),
					is_active BOOLEAN DEFAULT false,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					metadata JSONB
				);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains table changes with AST nodes
			if len(diff.TableChanges) > 0 {
				for _, tableDiff := range diff.TableChanges {
					switch tableDiff.Action {
					case schema.MetadataDiffActionCreate:
						assert.NotNil(t, tableDiff.NewASTNode,
							"Create action should have NewASTNode")
						assert.Nil(t, tableDiff.OldASTNode,
							"Create action should not have OldASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, tableDiff.NewTable,
							"AST-only mode should not have metadata")
					case schema.MetadataDiffActionDrop:
						assert.NotNil(t, tableDiff.OldASTNode,
							"Drop action should have OldASTNode")
						assert.Nil(t, tableDiff.NewASTNode,
							"Drop action should not have NewASTNode")
						// Verify that no metadata was extracted (AST-only mode)
						assert.Nil(t, tableDiff.OldTable,
							"AST-only mode should not have metadata")
					default:
						// Other actions like ALTER
						t.Logf("Encountered table action: %v", tableDiff.Action)
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected keywords for tables
			if tt.expectedMigration != "" {
				if containsTableString(tt.expectedMigration, "CREATE TABLE") {
					assert.Contains(t, migrationSQL, "CREATE TABLE",
						"Migration should contain CREATE TABLE statement")
				}
				if containsTableString(tt.expectedMigration, "DROP TABLE") {
					assert.Contains(t, migrationSQL, "DROP TABLE",
						"Migration should contain DROP TABLE statement")
				}
			}
		})
	}
}

func TestTableMigrationASTOnlyModeValidation(t *testing.T) {
	// Test that ensures AST-only mode works correctly without any metadata extraction
	previousSDL := ``
	currentSDL := `
		CREATE TABLE test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) DEFAULT 'active'
		);
	`

	// Get diff without metadata extraction
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly one table change
	assert.Len(t, diff.TableChanges, 1)
	tableDiff := diff.TableChanges[0]

	// Verify AST-only mode properties
	assert.Equal(t, schema.MetadataDiffActionCreate, tableDiff.Action)
	assert.Equal(t, "public", tableDiff.SchemaName)
	assert.Equal(t, "test_table", tableDiff.TableName)

	// Critical assertion: No metadata should be present (AST-only mode)
	assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
	assert.Nil(t, tableDiff.OldTable, "AST-only mode should not extract metadata")

	// But AST nodes should be present
	assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present for CREATE action")
	assert.Nil(t, tableDiff.OldASTNode, "No old AST node for CREATE action")

	// Generate migration should work with AST nodes only
	migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	assert.Contains(t, migrationSQL, "CREATE TABLE test_table")
	assert.Contains(t, migrationSQL, "id SERIAL PRIMARY KEY")
	assert.Contains(t, migrationSQL, "name VARCHAR(255) NOT NULL")
	assert.Contains(t, migrationSQL, "status VARCHAR(50) DEFAULT 'active'")
}

func TestMultipleTablesHandling(t *testing.T) {
	// Test that verifies multiple tables are handled correctly
	previousSDL := `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
	`
	currentSDL := `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
		CREATE TABLE table2 (id SERIAL PRIMARY KEY, name TEXT);
		CREATE TABLE table3 (id UUID PRIMARY KEY, data JSONB);
	`

	// Get diff without metadata extraction (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have exactly two table changes (CREATE actions for table2 and table3)
	assert.Len(t, diff.TableChanges, 2)

	// Verify both tables are CREATE actions
	createCount := 0
	tableNames := make(map[string]bool)
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.Action == schema.MetadataDiffActionCreate {
			createCount++
			// Verify AST-only mode properties
			assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present for CREATE action")
			assert.Nil(t, tableDiff.OldASTNode, "No old AST node for CREATE action")
			assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")

			// Collect table names to verify they're different
			tableNames[tableDiff.TableName] = true
		}
	}
	assert.Equal(t, 2, createCount, "Should have two CREATE actions")

	// Verify the table names are correct
	expectedTables := map[string]bool{
		"table2": true,
		"table3": true,
	}
	assert.Equal(t, expectedTables, tableNames, "Should have correct table names")

	// Generate migration SQL - this should work with both tables
	migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration contains both tables
	assert.Contains(t, migrationSQL, "CREATE TABLE table2")
	assert.Contains(t, migrationSQL, "CREATE TABLE table3")

	// Verify both tables are created
	tableCount := strings.Count(migrationSQL, "CREATE TABLE")
	assert.Equal(t, 2, tableCount, "Should create both tables")
}

func TestComplexTableWithConstraintsAndIndexes(t *testing.T) {
	// Test with complex table definition including constraints and indexes
	previousSDL := ``
	currentSDL := `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			order_date DATE DEFAULT CURRENT_DATE,
			total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0),
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
		);
	`

	// Get diff in AST-only mode
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have one table creation
	assert.Len(t, diff.TableChanges, 1)
	tableDiff := diff.TableChanges[0]

	// Verify properties
	assert.Equal(t, schema.MetadataDiffActionCreate, tableDiff.Action)
	assert.Equal(t, "public", tableDiff.SchemaName)
	assert.Equal(t, "orders", tableDiff.TableName)
	assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
	assert.NotNil(t, tableDiff.NewASTNode, "AST node should be present")

	// Generate migration
	migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)

	// Verify the migration contains the complex table definition
	assert.Contains(t, migrationSQL, "CREATE TABLE orders")
	assert.Contains(t, migrationSQL, "id SERIAL PRIMARY KEY")
	assert.Contains(t, migrationSQL, "total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0)")
	assert.Contains(t, migrationSQL, "CONSTRAINT fk_customer FOREIGN KEY")
	assert.Contains(t, migrationSQL, "CONSTRAINT unique_customer_date UNIQUE")
}

func TestTableConstraintSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name: "Add CHECK constraint",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" ADD CONSTRAINT "chk_price" CHECK (price > 0);`,
		},
		{
			name: "Add FOREIGN KEY constraint",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."orders" ADD CONSTRAINT "fk_customer" FOREIGN KEY (customer_id) REFERENCES customers(id);`,
		},
		{
			name: "Add UNIQUE constraint",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(255) NOT NULL,
					CONSTRAINT unique_email UNIQUE (email)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ADD CONSTRAINT "unique_email" UNIQUE (email);`,
		},
		{
			name: "Drop CHECK constraint",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" DROP CONSTRAINT IF EXISTS "chk_price";`,
		},
		{
			name: "Drop FOREIGN KEY constraint",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."orders" DROP CONSTRAINT IF EXISTS "fk_customer";`,
		},
		{
			name: "Modify CHECK constraint",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price >= 0)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" DROP CONSTRAINT IF EXISTS "chk_price";

ALTER TABLE "public"."products" ADD CONSTRAINT "chk_price" CHECK (price >= 0);`,
		},
		{
			name: "Add PRIMARY KEY constraint",
			previousSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL,
					CONSTRAINT pk_users PRIMARY KEY (id)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ADD CONSTRAINT "pk_users" PRIMARY KEY (id);`,
		},
		{
			name: "Drop PRIMARY KEY constraint",
			previousSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL,
					CONSTRAINT pk_users PRIMARY KEY (id)
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" DROP CONSTRAINT IF EXISTS "pk_users";`,
		},
		{
			name: "Add multiple constraints",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending',
					CONSTRAINT chk_total_amount CHECK (total_amount >= 0),
					CONSTRAINT chk_status CHECK (status IN ('pending', 'confirmed', 'shipped')),
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
					CONSTRAINT unique_customer_status UNIQUE (customer_id, status)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."orders" ADD CONSTRAINT "chk_total_amount" CHECK (total_amount >= 0);
ALTER TABLE "public"."orders" ADD CONSTRAINT "chk_status" CHECK (status IN ('pending', 'confirmed', 'shipped'));
ALTER TABLE "public"."orders" ADD CONSTRAINT "unique_customer_status" UNIQUE (customer_id, status);

ALTER TABLE "public"."orders" ADD CONSTRAINT "fk_customer" FOREIGN KEY (customer_id) REFERENCES customers(id);`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Get diff in AST-only mode (nil schemas for AST-only)
			diff, err := GetSDLDiff(test.currentSDL, test.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Should have one table change
			require.Len(t, diff.TableChanges, 1)
			tableDiff := diff.TableChanges[0]

			// Verify it's an ALTER action for constraint changes
			assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)

			// Verify AST-only mode properties
			assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
			assert.Nil(t, tableDiff.OldTable, "AST-only mode should not extract metadata")
			assert.NotNil(t, tableDiff.NewASTNode, "New AST node should be present")
			assert.NotNil(t, tableDiff.OldASTNode, "Old AST node should be present")

			// Verify constraint changes are detected
			hasConstraintChanges := len(tableDiff.CheckConstraintChanges) > 0 ||
				len(tableDiff.ForeignKeyChanges) > 0 ||
				len(tableDiff.UniqueConstraintChanges) > 0 ||
				len(tableDiff.PrimaryKeyChanges) > 0

			assert.True(t, hasConstraintChanges, "Should detect constraint changes")

			// Generate migration SQL
			migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err)

			// Normalize whitespace for comparison
			normalizedMigration := strings.TrimSpace(migrationSQL)
			normalizedExpected := strings.TrimSpace(test.expectedMigration)

			assert.Equal(t, normalizedExpected, normalizedMigration,
				"Migration SQL should match expected output")
		})
	}
}

func TestTableConstraintComplexChanges(t *testing.T) {
	// Test a complex scenario with constraint additions, modifications, and drops
	previousSDL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			age INTEGER CHECK (age >= 18),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`

	currentSDL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			age INTEGER CHECK (age >= 16),
			phone VARCHAR(20) UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
			status VARCHAR(20) DEFAULT 'pending',
			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT unique_user_amount UNIQUE (user_id, amount)
		);
	`

	// Get diff in AST-only mode
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have changes for both tables
	assert.Len(t, diff.TableChanges, 2)

	// Check that each table has the expected constraint changes
	for _, tableDiff := range diff.TableChanges {
		assert.Equal(t, schema.MetadataDiffActionAlter, tableDiff.Action)

		// Verify AST-only mode properties
		assert.Nil(t, tableDiff.NewTable, "AST-only mode should not extract metadata")
		assert.Nil(t, tableDiff.OldTable, "AST-only mode should not extract metadata")
		assert.NotNil(t, tableDiff.NewASTNode, "New AST node should be present")
		assert.NotNil(t, tableDiff.OldASTNode, "Old AST node should be present")

		// Verify constraint or column changes are detected
		hasConstraintChanges := len(tableDiff.CheckConstraintChanges) > 0 ||
			len(tableDiff.ForeignKeyChanges) > 0 ||
			len(tableDiff.UniqueConstraintChanges) > 0 ||
			len(tableDiff.PrimaryKeyChanges) > 0 ||
			len(tableDiff.ColumnChanges) > 0

		if tableDiff.TableName == "users" || tableDiff.TableName == "orders" {
			assert.True(t, hasConstraintChanges, "Should detect constraint or column changes for %s", tableDiff.TableName)
		}
	}

	// Generate migration SQL
	migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
	require.NoError(t, err)
	assert.NotEmpty(t, migrationSQL, "Migration SQL should not be empty")

	// Verify the migration handles both tables
	assert.Contains(t, migrationSQL, "users")
	assert.Contains(t, migrationSQL, "orders")
}

func TestCreateTableWithTableConstraintsIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name:        "Create table with CHECK constraint",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			expectedMigration: `CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);

`,
		},
		{
			name:        "Create table with FOREIGN KEY constraint",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					order_date DATE DEFAULT CURRENT_DATE,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
				);
			`,
			expectedMigration: `CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					order_date DATE DEFAULT CURRENT_DATE,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
				);

`,
		},
		{
			name:        "Create table with UNIQUE constraint",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL,
					username VARCHAR(50) NOT NULL,
					CONSTRAINT unique_email UNIQUE (email),
					CONSTRAINT unique_username UNIQUE (username)
				);
			`,
			expectedMigration: `CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL,
					username VARCHAR(50) NOT NULL,
					CONSTRAINT unique_email UNIQUE (email),
					CONSTRAINT unique_username UNIQUE (username)
				);

`,
		},
		{
			name:        "Create table with PRIMARY KEY constraint",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE composite_key_table (
					tenant_id INTEGER NOT NULL,
					user_id INTEGER NOT NULL,
					data TEXT,
					CONSTRAINT pk_composite PRIMARY KEY (tenant_id, user_id)
				);
			`,
			expectedMigration: `CREATE TABLE composite_key_table (
					tenant_id INTEGER NOT NULL,
					user_id INTEGER NOT NULL,
					data TEXT,
					CONSTRAINT pk_composite PRIMARY KEY (tenant_id, user_id)
				);

`,
		},
		{
			name:        "Create table with multiple constraints",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL,
					customer_id INTEGER NOT NULL,
					order_date DATE DEFAULT CURRENT_DATE,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending',
					created_at TIMESTAMP DEFAULT NOW(),
					CONSTRAINT pk_orders PRIMARY KEY (id),
					CONSTRAINT chk_total_amount CHECK (total_amount >= 0),
					CONSTRAINT chk_status CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
					CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
				);
			`,
			expectedMigration: `CREATE TABLE orders (
					id SERIAL,
					customer_id INTEGER NOT NULL,
					order_date DATE DEFAULT CURRENT_DATE,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending',
					created_at TIMESTAMP DEFAULT NOW(),
					CONSTRAINT pk_orders PRIMARY KEY (id),
					CONSTRAINT chk_total_amount CHECK (total_amount >= 0),
					CONSTRAINT chk_status CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
					CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
				);

`,
		},
		{
			name:        "Create table with complex CHECK constraints",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					first_name VARCHAR(50) NOT NULL,
					last_name VARCHAR(50) NOT NULL,
					email VARCHAR(320) NOT NULL,
					salary DECIMAL(10,2),
					hire_date DATE DEFAULT CURRENT_DATE,
					department_id INTEGER,
					CONSTRAINT chk_salary CHECK (salary IS NULL OR salary > 0),
					CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
					CONSTRAINT chk_hire_date CHECK (hire_date >= '2000-01-01'),
					CONSTRAINT chk_name_not_empty CHECK (LENGTH(TRIM(first_name)) > 0 AND LENGTH(TRIM(last_name)) > 0),
					CONSTRAINT unique_email UNIQUE (email),
					CONSTRAINT fk_department FOREIGN KEY (department_id) REFERENCES departments(id)
				);
			`,
			expectedMigration: `CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					first_name VARCHAR(50) NOT NULL,
					last_name VARCHAR(50) NOT NULL,
					email VARCHAR(320) NOT NULL,
					salary DECIMAL(10,2),
					hire_date DATE DEFAULT CURRENT_DATE,
					department_id INTEGER,
					CONSTRAINT chk_salary CHECK (salary IS NULL OR salary > 0),
					CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
					CONSTRAINT chk_hire_date CHECK (hire_date >= '2000-01-01'),
					CONSTRAINT chk_name_not_empty CHECK (LENGTH(TRIM(first_name)) > 0 AND LENGTH(TRIM(last_name)) > 0),
					CONSTRAINT unique_email UNIQUE (email),
					CONSTRAINT fk_department FOREIGN KEY (department_id) REFERENCES departments(id)
				);

`,
		},
		{
			name:        "Create table with FOREIGN KEY and referential actions",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE order_items (
					id SERIAL PRIMARY KEY,
					order_id INTEGER NOT NULL,
					product_id INTEGER NOT NULL,
					quantity INTEGER NOT NULL DEFAULT 1,
					unit_price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_quantity CHECK (quantity > 0),
					CONSTRAINT chk_unit_price CHECK (unit_price > 0),
					CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE,
					CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT ON UPDATE CASCADE,
					CONSTRAINT unique_order_product UNIQUE (order_id, product_id)
				);
			`,
			expectedMigration: `CREATE TABLE order_items (
					id SERIAL PRIMARY KEY,
					order_id INTEGER NOT NULL,
					product_id INTEGER NOT NULL,
					quantity INTEGER NOT NULL DEFAULT 1,
					unit_price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_quantity CHECK (quantity > 0),
					CONSTRAINT chk_unit_price CHECK (unit_price > 0),
					CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE,
					CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT ON UPDATE CASCADE,
					CONSTRAINT unique_order_product UNIQUE (order_id, product_id)
				);

`,
		},
		{
			name:        "Create table with schema-qualified constraint references",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE audit.user_actions (
					id BIGSERIAL PRIMARY KEY,
					user_id INTEGER NOT NULL,
					action_type VARCHAR(50) NOT NULL,
					table_name VARCHAR(100) NOT NULL,
					record_id INTEGER,
					action_timestamp TIMESTAMP DEFAULT NOW(),
					CONSTRAINT chk_action_type CHECK (action_type IN ('INSERT', 'UPDATE', 'DELETE')),
					CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
					CONSTRAINT unique_user_action_time UNIQUE (user_id, action_timestamp)
				);
			`,
			expectedMigration: `CREATE TABLE audit.user_actions (
					id BIGSERIAL PRIMARY KEY,
					user_id INTEGER NOT NULL,
					action_type VARCHAR(50) NOT NULL,
					table_name VARCHAR(100) NOT NULL,
					record_id INTEGER,
					action_timestamp TIMESTAMP DEFAULT NOW(),
					CONSTRAINT chk_action_type CHECK (action_type IN ('INSERT', 'UPDATE', 'DELETE')),
					CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE,
					CONSTRAINT unique_user_action_time UNIQUE (user_id, action_timestamp)
				);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains table creation with AST nodes
			require.Len(t, diff.TableChanges, 1)
			tableDiff := diff.TableChanges[0]

			assert.Equal(t, schema.MetadataDiffActionCreate, tableDiff.Action,
				"Should be a CREATE action")
			assert.NotNil(t, tableDiff.NewASTNode,
				"Create action should have NewASTNode")
			assert.Nil(t, tableDiff.OldASTNode,
				"Create action should not have OldASTNode")
			// Verify that no metadata was extracted (AST-only mode)
			assert.Nil(t, tableDiff.NewTable,
				"AST-only mode should not have metadata")

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected constraint keywords
			if tt.expectedMigration != "" {
				if containsTableString(tt.expectedMigration, "CHECK") {
					assert.Contains(t, migrationSQL, "CHECK",
						"Migration should contain CHECK constraint")
				}
				if containsTableString(tt.expectedMigration, "FOREIGN KEY") {
					assert.Contains(t, migrationSQL, "FOREIGN KEY",
						"Migration should contain FOREIGN KEY constraint")
				}
				if containsTableString(tt.expectedMigration, "UNIQUE") {
					assert.Contains(t, migrationSQL, "UNIQUE",
						"Migration should contain UNIQUE constraint")
				}
				if containsTableString(tt.expectedMigration, "PRIMARY KEY") {
					assert.Contains(t, migrationSQL, "PRIMARY KEY",
						"Migration should contain PRIMARY KEY constraint")
				}
				if containsTableString(tt.expectedMigration, "CONSTRAINT") {
					assert.Contains(t, migrationSQL, "CONSTRAINT",
						"Migration should contain CONSTRAINT keyword")
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring (table tests)
func containsTableString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestIndexSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name: "Create standalone indexes",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer_id ON orders (customer_id);
				CREATE UNIQUE INDEX idx_orders_status_unique ON orders (status);
				CREATE INDEX idx_orders_complex ON orders (customer_id, total_amount DESC);
			`,
			expectedMigration: `CREATE INDEX idx_orders_customer_id ON orders (customer_id);
CREATE UNIQUE INDEX idx_orders_status_unique ON orders (status);
CREATE INDEX idx_orders_complex ON orders (customer_id, total_amount DESC);`,
		},
		{
			name: "Drop standalone indexes",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer_id ON orders (customer_id);
				CREATE UNIQUE INDEX idx_orders_status_unique ON orders (status);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);
			`,
			expectedMigration: `DROP INDEX IF EXISTS "public"."idx_orders_customer_id";
DROP INDEX IF EXISTS "public"."idx_orders_status_unique";`,
		},
		{
			name: "Replace indexes (drop old, create new)",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_old ON orders (customer_id);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_new ON orders (customer_id, status);
			`,
			expectedMigration: `DROP INDEX IF EXISTS "public"."idx_orders_old";

CREATE INDEX idx_orders_new ON orders (customer_id, status);`,
		},
		{
			name: "Mixed index operations",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer ON orders (customer_id);
				CREATE INDEX idx_orders_amount ON orders (total_amount);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer ON orders (customer_id);
				CREATE UNIQUE INDEX idx_orders_status ON orders (status);
				CREATE INDEX idx_orders_complex ON orders (customer_id, status, total_amount DESC);
			`,
			expectedMigration: `DROP INDEX IF EXISTS "public"."idx_orders_amount";

CREATE UNIQUE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_complex ON orders (customer_id, status, total_amount DESC);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get diff in AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err, "GetSDLDiff should not return error")
			require.NotNil(t, diff, "diff should not be nil")

			// Generate migration using AST-only mode
			migrationSQL, err := schema.GenerateMigration(storepb.Engine_POSTGRES, diff)
			require.NoError(t, err, "GenerateMigration should not return error")

			// Normalize whitespace for comparison
			normalizedExpected := strings.TrimSpace(tt.expectedMigration)
			normalizedActual := strings.TrimSpace(migrationSQL)

			if normalizedExpected == "" {
				assert.Empty(t, normalizedActual, "Migration should be empty")
			} else {
				// For index tests, check that essential parts are present
				if strings.Contains(tt.expectedMigration, "CREATE INDEX") {
					assert.Contains(t, migrationSQL, "CREATE INDEX",
						"Migration should contain CREATE INDEX statements")
				}
				if strings.Contains(tt.expectedMigration, "CREATE UNIQUE INDEX") {
					assert.Contains(t, migrationSQL, "CREATE UNIQUE INDEX",
						"Migration should contain CREATE UNIQUE INDEX statements")
				}
				if strings.Contains(tt.expectedMigration, "DROP INDEX") {
					assert.Contains(t, migrationSQL, "DROP INDEX IF EXISTS",
						"Migration should contain DROP INDEX statements")
				}

				// Check that specific index names are mentioned
				lines := strings.Split(normalizedExpected, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.Contains(line, "idx_") {
						// Extract index name and check it exists in migration
						if strings.Contains(line, "CREATE") {
							// For CREATE statements, check the full statement structure is preserved
							assert.Contains(t, migrationSQL, line,
								"Migration should contain the expected CREATE INDEX statement")
						} else if strings.Contains(line, "DROP") {
							// For DROP statements, just check the index name is dropped
							indexName := extractIndexNameFromLine(line)
							if indexName != "" {
								assert.Contains(t, migrationSQL, indexName,
									"Migration should drop the expected index")
							}
						}
					}
				}
			}
		})
	}
}

// Helper function to extract index name from DROP INDEX statement
func extractIndexNameFromLine(line string) string {
	if strings.Contains(line, "DROP INDEX") && strings.Contains(line, `"."`) {
		// Extract from DROP INDEX IF EXISTS "schema"."index_name";
		parts := strings.Split(line, `"."`)
		if len(parts) >= 2 {
			indexPart := parts[1]
			// Remove trailing quote and semicolon
			if strings.Contains(indexPart, `"`) {
				return strings.Split(indexPart, `"`)[0]
			}
		}
	}
	return ""
}
