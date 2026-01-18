package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestViewSDLDiff(t *testing.T) {
	tests := []struct {
		name                string
		previousSDL         string
		currentSDL          string
		expectedViewChanges int
		expectedActions     []schema.MetadataDiffAction
	}{
		{
			name:        "Create new view",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW active_users AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			expectedViewChanges: 1,
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop view",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW active_users AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedViewChanges: 1,
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify view (drop and recreate)",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);
				
				CREATE VIEW active_users AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);
				
				CREATE VIEW active_users AS
				SELECT id, name, 'active' as status
				FROM users
				WHERE active = true;
			`,
			expectedViewChanges: 2, // Drop + Create
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionDrop, schema.MetadataDiffActionCreate},
		},
		{
			name: "No changes to view",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW active_users AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW active_users AS
				SELECT id, name
				FROM users
				WHERE active = true;
			`,
			expectedViewChanges: 0,
			expectedActions:     []schema.MetadataDiffAction{},
		},
		{
			name: "Multiple views with different changes",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW all_users AS
				SELECT * FROM users;
				
				CREATE VIEW active_users AS
				SELECT id, name FROM users WHERE active = true;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE VIEW all_users AS
				SELECT * FROM users;
				
				CREATE VIEW admin_users AS
				SELECT id, name FROM users WHERE role = 'admin';
			`,
			expectedViewChanges: 2, // Drop active_users + Create admin_users
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionCreate, schema.MetadataDiffActionDrop},
		},
		{
			name: "Schema-qualified view names",
			previousSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
			`,
			currentSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255)
				);
				
				CREATE VIEW test_schema.product_summary AS
				SELECT id, name FROM test_schema.products;
			`,
			expectedViewChanges: 1,
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			assert.Equal(t, tt.expectedViewChanges, len(diff.ViewChanges),
				"Expected %d view changes, got %d", tt.expectedViewChanges, len(diff.ViewChanges))

			// Check that the actions match expectations
			var actualActions []schema.MetadataDiffAction
			for _, viewDiff := range diff.ViewChanges {
				actualActions = append(actualActions, viewDiff.Action)
			}

			// Handle nil vs empty slice comparison
			if len(tt.expectedActions) == 0 && len(actualActions) == 0 {
				// Both are effectively empty - test passes
				t.Log("Both expected and actual actions are empty")
			} else {
				assert.ElementsMatch(t, tt.expectedActions, actualActions,
					"Expected actions %v, got %v", tt.expectedActions, actualActions)
			}

			// Verify AST nodes are properly set
			for i, viewDiff := range diff.ViewChanges {
				switch viewDiff.Action {
				case schema.MetadataDiffActionCreate:
					assert.NotNil(t, viewDiff.NewASTNode,
						"View diff %d should have NewASTNode for CREATE action", i)
					assert.Nil(t, viewDiff.OldASTNode,
						"View diff %d should not have OldASTNode for CREATE action", i)
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, viewDiff.OldASTNode,
						"View diff %d should have OldASTNode for DROP action", i)
					assert.Nil(t, viewDiff.NewASTNode,
						"View diff %d should not have NewASTNode for DROP action", i)
				default:
					t.Errorf("Unexpected action %v for view diff %d", viewDiff.Action, i)
				}
			}
		})
	}
}

func TestIdentifierParsing(t *testing.T) {
	tests := []struct {
		name           string
		identifier     string
		expectedSchema string
		expectedView   string
	}{
		{
			name:           "Simple view name",
			identifier:     "my_view",
			expectedSchema: "public",
			expectedView:   "my_view",
		},
		{
			name:           "Schema-qualified view name",
			identifier:     "test_schema.my_view",
			expectedSchema: "test_schema",
			expectedView:   "my_view",
		},
		{
			name:           "Public schema explicit",
			identifier:     "public.user_summary",
			expectedSchema: "public",
			expectedView:   "user_summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, view := parseIdentifier(tt.identifier)
			assert.Equal(t, tt.expectedSchema, schema)
			assert.Equal(t, tt.expectedView, view)
		})
	}
}
