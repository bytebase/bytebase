package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTableDiffScenarios(t *testing.T) {
	testCases := []struct {
		name                string
		currentSDL          string
		previousSDL         string
		expectedCreateCount int
		expectedAlterCount  int
		expectedDropCount   int
		expectedTableNames  []string
	}{
		{
			name: "Create new table",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			previousSDL:         "",
			expectedCreateCount: 1,
			expectedAlterCount:  0,
			expectedDropCount:   0,
			expectedTableNames:  []string{"users"},
		},
		{
			name:       "Drop existing table",
			currentSDL: "",
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedCreateCount: 0,
			expectedAlterCount:  0,
			expectedDropCount:   1,
			expectedTableNames:  []string{"users"},
		},
		{
			name: "Modify existing table",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255)
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			expectedCreateCount: 0,
			expectedAlterCount:  1,
			expectedDropCount:   0,
			expectedTableNames:  []string{"users"},
		},
		{
			name: "Mixed operations",
			currentSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255)
			);
			
			CREATE TABLE products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			previousSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,  
				name VARCHAR(255) NOT NULL
			);
			
			CREATE TABLE old_table (
				id SERIAL PRIMARY KEY
			);`,
			expectedCreateCount: 1, // products
			expectedAlterCount:  1, // users
			expectedDropCount:   1, // old_table
			expectedTableNames:  []string{"users", "products", "old_table"},
		},
		{
			name: "Schema qualified tables",
			currentSDL: `CREATE TABLE public.users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);
			
			CREATE TABLE admin.settings (
				key VARCHAR(255) PRIMARY KEY,
				value TEXT
			);`,
			previousSDL: `CREATE TABLE public.users (
				id SERIAL PRIMARY KEY
			);`,
			expectedCreateCount: 1, // admin.settings
			expectedAlterCount:  1, // public.users
			expectedDropCount:   0,
			expectedTableNames:  []string{"users", "settings"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tc.currentSDL, tc.previousSDL, nil)
			require.NoError(t, err)

			// Count different action types
			createCount := 0
			alterCount := 0
			dropCount := 0

			for _, change := range diff.TableChanges {
				switch change.Action {
				case schema.MetadataDiffActionCreate:
					createCount++
				case schema.MetadataDiffActionAlter:
					alterCount++
				case schema.MetadataDiffActionDrop:
					dropCount++
				default:
					// Should not reach here with valid actions
				}
			}

			assert.Equal(t, tc.expectedCreateCount, createCount, "CREATE count mismatch")
			assert.Equal(t, tc.expectedAlterCount, alterCount, "ALTER count mismatch")
			assert.Equal(t, tc.expectedDropCount, dropCount, "DROP count mismatch")

			// Verify table names
			actualTableNames := make([]string, len(diff.TableChanges))
			for i, change := range diff.TableChanges {
				actualTableNames[i] = change.TableName
			}

			assert.ElementsMatch(t, tc.expectedTableNames, actualTableNames, "Table names mismatch")

			// Verify schema names are set correctly
			for _, change := range diff.TableChanges {
				assert.NotEmpty(t, change.SchemaName, "Schema name should not be empty")
			}

			// Verify AST nodes are populated correctly
			for _, change := range diff.TableChanges {
				switch change.Action {
				case schema.MetadataDiffActionCreate:
					assert.Nil(t, change.OldASTNode, "CREATE action should have nil OldASTNode")
					assert.NotNil(t, change.NewASTNode, "CREATE action should have NewASTNode")
				case schema.MetadataDiffActionAlter:
					assert.NotNil(t, change.OldASTNode, "ALTER action should have OldASTNode")
					assert.NotNil(t, change.NewASTNode, "ALTER action should have NewASTNode")
					assert.NotEqual(t, change.OldASTNode, change.NewASTNode, "ALTER action should have different OldASTNode and NewASTNode")
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, change.OldASTNode, "DROP action should have OldASTNode")
					assert.Nil(t, change.NewASTNode, "DROP action should have nil NewASTNode")
				default:
					// Should not reach here with valid actions
				}
			}
		})
	}
}
