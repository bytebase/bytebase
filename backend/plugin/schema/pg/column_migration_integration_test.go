package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestColumnSDLDiffAndMigrationIntegration(t *testing.T) {
	tests := []struct {
		name              string
		previousSDL       string
		currentSDL        string
		expectedMigration string
	}{
		{
			name: "Add new column",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(320) UNIQUE
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ADD COLUMN email VARCHAR(320) UNIQUE;

`,
		},
		{
			name: "Drop column",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					phone VARCHAR(20)
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" DROP COLUMN IF EXISTS "phone";

`,
		},
		{
			name: "Modify column (ALTER TYPE)",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(15,3) NOT NULL DEFAULT 0.00
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" ALTER COLUMN "price" TYPE DECIMAL(15,3);
ALTER TABLE "public"."products" ALTER COLUMN "price" SET DEFAULT 0.00;

`,
		},
		{
			name: "Modify column nullable",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320)
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ALTER COLUMN "email" SET NOT NULL;

`,
		},
		{
			name: "Drop column nullable constraint",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320)
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ALTER COLUMN "email" DROP NOT NULL;

`,
		},
		{
			name: "Add default value to existing column",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL DEFAULT 9.99
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" ALTER COLUMN "price" SET DEFAULT 9.99;

`,
		},
		{
			name: "Drop default value from column",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL DEFAULT 9.99
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" ALTER COLUMN "price" DROP DEFAULT;

`,
		},
		{
			name: "Change column type only",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ALTER COLUMN "name" TYPE VARCHAR(255);

`,
		},
		{
			name: "Multiple column changes in same table",
			previousSDL: `
				CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100),
					email VARCHAR(255) NOT NULL,
					salary DECIMAL(8,2)
				);
			`,
			currentSDL: `
				CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					name VARCHAR(200) NOT NULL,
					email VARCHAR(320) NOT NULL,
					salary DECIMAL(10,2) DEFAULT 50000.00
				);
			`,
			expectedMigration: `ALTER TABLE "public"."employees" ALTER COLUMN "name" TYPE VARCHAR(200);
ALTER TABLE "public"."employees" ALTER COLUMN "name" SET NOT NULL;
ALTER TABLE "public"."employees" ALTER COLUMN "email" TYPE VARCHAR(320);
ALTER TABLE "public"."employees" ALTER COLUMN "salary" TYPE DECIMAL(10,2);
ALTER TABLE "public"."employees" ALTER COLUMN "salary" SET DEFAULT 50000.00;

`,
		},
		{
			name: "Add column with complex constraints",
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
					status VARCHAR(20) NOT NULL DEFAULT 'pending'
				);
			`,
			expectedMigration: `ALTER TABLE "public"."orders" ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending';

`,
		},
		{
			name: "Change column from nullable to not null with default",
			previousSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					category VARCHAR(50)
				);
			`,
			currentSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					category VARCHAR(50) NOT NULL DEFAULT 'general'
				);
			`,
			expectedMigration: `ALTER TABLE "public"."products" ALTER COLUMN "category" SET NOT NULL;
ALTER TABLE "public"."products" ALTER COLUMN "category" SET DEFAULT 'general';

`,
		},
		{
			name: "Complex type change with precision",
			previousSDL: `
				CREATE TABLE measurements (
					id SERIAL PRIMARY KEY,
					value DECIMAL(5,2)
				);
			`,
			currentSDL: `
				CREATE TABLE measurements (
					id SERIAL PRIMARY KEY,
					value DECIMAL(10,4) NOT NULL DEFAULT 0.0000
				);
			`,
			expectedMigration: `ALTER TABLE "public"."measurements" ALTER COLUMN "value" TYPE DECIMAL(10,4);
ALTER TABLE "public"."measurements" ALTER COLUMN "value" SET NOT NULL;
ALTER TABLE "public"."measurements" ALTER COLUMN "value" SET DEFAULT 0.0000;

`,
		},
		{
			name: "Add multiple columns",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(320) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			expectedMigration: `ALTER TABLE "public"."users" ADD COLUMN email VARCHAR(320) UNIQUE;
ALTER TABLE "public"."users" ADD COLUMN created_at TIMESTAMP DEFAULT NOW();

`,
		},
		{
			name: "Drop multiple columns",
			previousSDL: `
				CREATE TABLE temp_data (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					temp_field1 VARCHAR(100),
					temp_field2 INTEGER,
					description TEXT
				);
			`,
			currentSDL: `
				CREATE TABLE temp_data (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					description TEXT
				);
			`,
			expectedMigration: `ALTER TABLE "public"."temp_data" DROP COLUMN IF EXISTS "temp_field1";
ALTER TABLE "public"."temp_data" DROP COLUMN IF EXISTS "temp_field2";

`,
		},
		{
			name: "Mixed operations: add, drop, modify",
			previousSDL: `
				CREATE TABLE inventory (
					id SERIAL PRIMARY KEY,
					product_name VARCHAR(100) NOT NULL,
					old_field VARCHAR(50),
					quantity INTEGER DEFAULT 0
				);
			`,
			currentSDL: `
				CREATE TABLE inventory (
					id SERIAL PRIMARY KEY,
					product_name VARCHAR(200) NOT NULL,
					quantity INTEGER DEFAULT 1,
					location VARCHAR(100) NOT NULL DEFAULT 'warehouse'
				);
			`,
			expectedMigration: `ALTER TABLE "public"."inventory" DROP COLUMN IF EXISTS "old_field";

ALTER TABLE "public"."inventory" ADD COLUMN location VARCHAR(100) NOT NULL DEFAULT 'warehouse';
ALTER TABLE "public"."inventory" ALTER COLUMN "product_name" TYPE VARCHAR(200);
ALTER TABLE "public"."inventory" ALTER COLUMN "quantity" SET DEFAULT 1;

`,
		},
		{
			name: "Integer type variations",
			previousSDL: `
				CREATE TABLE counters (
					id SERIAL PRIMARY KEY,
					small_count SMALLINT,
					big_count BIGINT
				);
			`,
			currentSDL: `
				CREATE TABLE counters (
					id SERIAL PRIMARY KEY,
					small_count INTEGER NOT NULL DEFAULT 0,
					big_count BIGINT NOT NULL
				);
			`,
			expectedMigration: `ALTER TABLE "public"."counters" ALTER COLUMN "small_count" TYPE INTEGER;
ALTER TABLE "public"."counters" ALTER COLUMN "small_count" SET NOT NULL;
ALTER TABLE "public"."counters" ALTER COLUMN "small_count" SET DEFAULT 0;
ALTER TABLE "public"."counters" ALTER COLUMN "big_count" SET NOT NULL;

`,
		},
		{
			name: "Text and character types",
			previousSDL: `
				CREATE TABLE documents (
					id SERIAL PRIMARY KEY,
					title CHAR(50),
					content TEXT
				);
			`,
			currentSDL: `
				CREATE TABLE documents (
					id SERIAL PRIMARY KEY,
					title VARCHAR(100) NOT NULL,
					content TEXT DEFAULT ''
				);
			`,
			expectedMigration: `ALTER TABLE "public"."documents" ALTER COLUMN "title" TYPE VARCHAR(100);
ALTER TABLE "public"."documents" ALTER COLUMN "title" SET NOT NULL;
ALTER TABLE "public"."documents" ALTER COLUMN "content" SET DEFAULT '';

`,
		},
		{
			name: "Boolean column with default",
			previousSDL: `
				CREATE TABLE features (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL
				);
			`,
			currentSDL: `
				CREATE TABLE features (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT false
				);
			`,
			expectedMigration: `ALTER TABLE "public"."features" ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT false;

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Get SDL diff using AST-only mode (no metadata extraction)
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Step 2: Verify that diff contains column changes with AST nodes
			if len(diff.TableChanges) > 0 {
				for _, tableDiff := range diff.TableChanges {
					for _, colDiff := range tableDiff.ColumnChanges {
						switch colDiff.Action {
						case schema.MetadataDiffActionCreate:
							assert.NotNil(t, colDiff.NewASTNode,
								"Create action should have NewASTNode")
							assert.Nil(t, colDiff.OldASTNode,
								"Create action should not have OldASTNode")
							// Verify that no metadata was extracted (AST-only mode)
							assert.Nil(t, colDiff.NewColumn,
								"AST-only mode should not have metadata")
						case schema.MetadataDiffActionDrop:
							assert.NotNil(t, colDiff.OldASTNode,
								"Drop action should have OldASTNode")
							assert.Nil(t, colDiff.NewASTNode,
								"Drop action should not have NewASTNode")
							// Verify that no metadata was extracted (AST-only mode)
							assert.Nil(t, colDiff.OldColumn,
								"AST-only mode should not have metadata")
						case schema.MetadataDiffActionAlter:
							assert.NotNil(t, colDiff.OldASTNode,
								"Alter action should have OldASTNode")
							assert.NotNil(t, colDiff.NewASTNode,
								"Alter action should have NewASTNode")
							// Verify that no metadata was extracted (AST-only mode)
							assert.Nil(t, colDiff.OldColumn,
								"AST-only mode should not have metadata")
							assert.Nil(t, colDiff.NewColumn,
								"AST-only mode should not have metadata")
						default:
							// Other actions
							t.Logf("Encountered column action: %v", colDiff.Action)
						}
					}
				}
			}

			// Step 3: Generate migration SQL using AST nodes
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// Step 4: Verify the generated migration matches expectations
			assert.Equal(t, tt.expectedMigration, migrationSQL,
				"Generated migration SQL should match expected output")

			// Step 5: Verify the migration contains expected keywords for columns
			if tt.expectedMigration != "" {
				if containsColumnString(tt.expectedMigration, "ADD COLUMN") {
					assert.Contains(t, migrationSQL, "ADD COLUMN",
						"Migration should contain ADD COLUMN statement")
				}
				if containsColumnString(tt.expectedMigration, "DROP COLUMN") {
					assert.Contains(t, migrationSQL, "DROP COLUMN",
						"Migration should contain DROP COLUMN statement")
				}
				if containsColumnString(tt.expectedMigration, "ALTER COLUMN") {
					assert.Contains(t, migrationSQL, "ALTER COLUMN",
						"Migration should contain ALTER COLUMN statement")
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsColumnString(s, substr string) bool {
	return strings.Contains(s, substr)
}
