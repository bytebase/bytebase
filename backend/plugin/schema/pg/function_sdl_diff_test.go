package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestFunctionSDLDiff(t *testing.T) {
	tests := []struct {
		name                    string
		previousSDL             string
		currentSDL              string
		expectedFunctionChanges int
		expectedActions         []schema.MetadataDiffAction
	}{
		{
			name:        "Create new function",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop function",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify function (CREATE OR REPLACE)",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					active BOOLEAN DEFAULT true
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users WHERE active = true);
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 1, // ALTER only
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "No changes to function",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 0,
			expectedActions:         []schema.MetadataDiffAction{},
		},
		{
			name: "Multiple functions with different changes",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_all_users()
				RETURNS SETOF users AS $$
				BEGIN
					RETURN QUERY SELECT * FROM users;
				END;
				$$ LANGUAGE plpgsql;
				
				CREATE OR REPLACE FUNCTION get_user_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$ LANGUAGE plpgsql;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
				
				CREATE OR REPLACE FUNCTION get_all_users()
				RETURNS SETOF users AS $$
				BEGIN
					RETURN QUERY SELECT * FROM users;
				END;
				$$ LANGUAGE plpgsql;
				
				CREATE OR REPLACE FUNCTION get_admin_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users WHERE role = 'admin');
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 2, // Drop get_user_count + Create get_admin_count
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate, schema.MetadataDiffActionDrop},
		},
		{
			name: "Schema-qualified function names",
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
				
				CREATE OR REPLACE FUNCTION test_schema.get_product_count()
				RETURNS INTEGER AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM test_schema.products);
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Function with complex signature",
			previousSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					amount DECIMAL(10,2),
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			currentSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					amount DECIMAL(10,2),
					created_at TIMESTAMP DEFAULT NOW()
				);
				
				CREATE OR REPLACE FUNCTION calculate_total(
					p_start_date DATE,
					p_end_date DATE,
					p_discount_rate DECIMAL DEFAULT 0.0
				)
				RETURNS DECIMAL(10,2) AS $$
				BEGIN
					RETURN (
						SELECT COALESCE(SUM(amount * (1 - p_discount_rate)), 0)
						FROM orders
						WHERE created_at::DATE BETWEEN p_start_date AND p_end_date
					);
				END;
				$$ LANGUAGE plpgsql;
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			assert.Equal(t, tt.expectedFunctionChanges, len(diff.FunctionChanges),
				"Expected %d function changes, got %d", tt.expectedFunctionChanges, len(diff.FunctionChanges))

			// Check that the actions match expectations
			var actualActions []schema.MetadataDiffAction
			for _, funcDiff := range diff.FunctionChanges {
				actualActions = append(actualActions, funcDiff.Action)
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
			for i, funcDiff := range diff.FunctionChanges {
				switch funcDiff.Action {
				case schema.MetadataDiffActionCreate:
					assert.NotNil(t, funcDiff.NewASTNode,
						"Function diff %d should have NewASTNode for CREATE action", i)
					assert.Nil(t, funcDiff.OldASTNode,
						"Function diff %d should not have OldASTNode for CREATE action", i)
				case schema.MetadataDiffActionDrop:
					assert.NotNil(t, funcDiff.OldASTNode,
						"Function diff %d should have OldASTNode for DROP action", i)
					assert.Nil(t, funcDiff.NewASTNode,
						"Function diff %d should not have NewASTNode for DROP action", i)
				case schema.MetadataDiffActionAlter:
					assert.NotNil(t, funcDiff.OldASTNode,
						"Function diff %d should have OldASTNode for ALTER action", i)
					assert.NotNil(t, funcDiff.NewASTNode,
						"Function diff %d should have NewASTNode for ALTER action", i)
				default:
					t.Errorf("Unexpected action %v for function diff %d", funcDiff.Action, i)
				}
			}
		})
	}
}

func TestFunctionIdentifierParsing(t *testing.T) {
	tests := []struct {
		name             string
		identifier       string
		expectedSchema   string
		expectedFunction string
	}{
		{
			name:             "Simple function name",
			identifier:       "my_function",
			expectedSchema:   "public",
			expectedFunction: "my_function",
		},
		{
			name:             "Schema-qualified function name",
			identifier:       "test_schema.my_function",
			expectedSchema:   "test_schema",
			expectedFunction: "my_function",
		},
		{
			name:             "Public schema explicit",
			identifier:       "public.user_summary_func",
			expectedSchema:   "public",
			expectedFunction: "user_summary_func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, function := parseIdentifier(tt.identifier)
			assert.Equal(t, tt.expectedSchema, schema)
			assert.Equal(t, tt.expectedFunction, function)
		})
	}
}
