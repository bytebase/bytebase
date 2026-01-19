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
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
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

// TestSameNameFunctionsWithDifferentSignatures tests BYT-7994
// PostgreSQL supports function overloading - same name with different arguments
// The SDL diff should recognize functions by (name + signature), not just name
func TestSameNameFunctionsWithDifferentSignatures(t *testing.T) {
	tests := []struct {
		name                    string
		previousSDL             string
		currentSDL              string
		expectedFunctionChanges int
		expectedCreateCount     int
		expectedDropCount       int
		expectedAlterCount      int
		description             string
	}{
		{
			name: "Add second function with same name but different signature - BYT-7994",
			previousSDL: `
CREATE FUNCTION "public"."my_procedure"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with one parameter: %', p_id;
END;
$$;
`,
			currentSDL: `
CREATE FUNCTION "public"."my_procedure"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with one parameter: %', p_id;
END;
$$;

CREATE FUNCTION "public"."my_procedure"(p_id integer, p_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Function with two parameters: % - %', p_id, p_name;
END;
$$;
`,
			expectedFunctionChanges: 1,
			expectedCreateCount:     1,
			expectedDropCount:       0,
			expectedAlterCount:      0,
			description:             "Should add the new function without dropping the old one",
		},
		{
			name: "Three overloaded functions with same name",
			previousSDL: `
CREATE FUNCTION "public"."calculate"(x integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x * 2;
$$;
`,
			currentSDL: `
CREATE FUNCTION "public"."calculate"(x integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x * 2;
$$;

CREATE FUNCTION "public"."calculate"(x integer, y integer) RETURNS integer
    LANGUAGE sql
    AS $$
    SELECT x + y;
$$;

CREATE FUNCTION "public"."calculate"(x numeric, y numeric, z numeric) RETURNS numeric
    LANGUAGE sql
    AS $$
    SELECT x + y + z;
$$;
`,
			expectedFunctionChanges: 2,
			expectedCreateCount:     2,
			expectedDropCount:       0,
			expectedAlterCount:      0,
			description:             "Should add two new overloaded functions",
		},
		{
			name: "Remove one overloaded function, keep others",
			previousSDL: `
CREATE FUNCTION "public"."process"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process one param';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process two params';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text, p_email text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process three params';
END;
$$;
`,
			currentSDL: `
CREATE FUNCTION "public"."process"(p_id integer) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process one param';
END;
$$;

CREATE FUNCTION "public"."process"(p_id integer, p_name text, p_email text) RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE NOTICE 'Process three params';
END;
$$;
`,
			expectedFunctionChanges: 1,
			expectedCreateCount:     0,
			expectedDropCount:       1,
			expectedAlterCount:      0,
			description:             "Should drop only the middle function (2 params), keep others",
		},
		{
			name: "Modify one overloaded function, keep others unchanged",
			previousSDL: `
CREATE FUNCTION "public"."get_info"(p_id integer) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User ID: ' || p_id::text;
$$;

CREATE FUNCTION "public"."get_info"(p_name text) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User Name: ' || p_name;
$$;
`,
			currentSDL: `
CREATE FUNCTION "public"."get_info"(p_id integer) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'Updated User ID: ' || p_id::text;
$$;

CREATE FUNCTION "public"."get_info"(p_name text) RETURNS text
    LANGUAGE sql
    AS $$
    SELECT 'User Name: ' || p_name;
$$;
`,
			expectedFunctionChanges: 1,
			expectedCreateCount:     0,
			expectedDropCount:       0,
			expectedAlterCount:      1,
			description:             "Should modify only the first function, keep second unchanged",
		},
		{
			name: "Procedures with same name but different signatures",
			previousSDL: `
CREATE PROCEDURE "public"."update_status"(p_id integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = 'active' WHERE id = p_id;
END;
$$;
`,
			currentSDL: `
CREATE PROCEDURE "public"."update_status"(p_id integer)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = 'active' WHERE id = p_id;
END;
$$;

CREATE PROCEDURE "public"."update_status"(p_id integer, p_new_status text)
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE users SET status = p_new_status WHERE id = p_id;
END;
$$;
`,
			expectedFunctionChanges: 1,
			expectedCreateCount:     1,
			expectedDropCount:       0,
			expectedAlterCount:      0,
			description:             "Should add the new procedure without dropping the old one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)

			// Count different action types
			createCount := 0
			dropCount := 0
			alterCount := 0

			for _, change := range diff.FunctionChanges {
				signature := ""
				if change.NewFunction != nil {
					signature = change.NewFunction.Signature
				} else if change.OldFunction != nil {
					signature = change.OldFunction.Signature
				}
				t.Logf("Function: %s, Signature: %s, Action: %s",
					change.FunctionName, signature, change.Action)

				switch change.Action {
				case schema.MetadataDiffActionCreate:
					createCount++
				case schema.MetadataDiffActionDrop:
					dropCount++
				case schema.MetadataDiffActionAlter:
					alterCount++
				default:
					t.Errorf("Unexpected action: %v", change.Action)
				}
			}

			t.Logf("Description: %s", tt.description)
			t.Logf("Total changes: %d (CREATE: %d, DROP: %d, ALTER: %d)",
				len(diff.FunctionChanges), createCount, dropCount, alterCount)

			assert.Equal(t, tt.expectedFunctionChanges, len(diff.FunctionChanges),
				"Expected %d function changes, got %d", tt.expectedFunctionChanges, len(diff.FunctionChanges))
			assert.Equal(t, tt.expectedCreateCount, createCount,
				"Expected %d CREATE actions, got %d", tt.expectedCreateCount, createCount)
			assert.Equal(t, tt.expectedDropCount, dropCount,
				"Expected %d DROP actions, got %d", tt.expectedDropCount, dropCount)
			assert.Equal(t, tt.expectedAlterCount, alterCount,
				"Expected %d ALTER actions, got %d", tt.expectedAlterCount, alterCount)

			// If we see unexpected DROP actions for overloaded functions, that's the bug
			if dropCount > tt.expectedDropCount {
				t.Errorf("BUG DETECTED (BYT-7994): Found %d DROP action(s), expected %d. "+
					"Functions with same name but different signatures should be identified separately.",
					dropCount, tt.expectedDropCount)
			}
		})
	}
}
