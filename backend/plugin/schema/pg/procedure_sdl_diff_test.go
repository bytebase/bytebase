package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestProcedureSDLDiff(t *testing.T) {
	tests := []struct {
		name                    string
		previousSDL             string
		currentSDL              string
		expectedFunctionChanges int // Note: procedures are stored as functions in PostgreSQL
		expectedActions         []schema.MetadataDiffAction
	}{
		{
			name:        "Create new procedure",
			previousSDL: ``,
			currentSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT,
					created_at TIMESTAMP DEFAULT NOW()
				);
				
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO logs (message) VALUES (msg);
				END;
				$$;
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop procedure",
			previousSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT,
					created_at TIMESTAMP DEFAULT NOW()
				);
				
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO logs (message) VALUES (msg);
				END;
				$$;
			`,
			currentSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			expectedFunctionChanges: 1,
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify procedure (CREATE OR REPLACE)",
			previousSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT,
					created_at TIMESTAMP DEFAULT NOW()
				);
				
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO logs (message) VALUES (msg);
				END;
				$$;
			`,
			currentSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT,
					created_at TIMESTAMP DEFAULT NOW()
				);
				
				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO logs (message, created_at) VALUES (msg, NOW());
				END;
				$$;
			`,
			expectedFunctionChanges: 1, // ALTER only
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "Mixed functions and procedures",
			previousSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255),
					email VARCHAR(255)
				);
				
				CREATE FUNCTION get_user_count() RETURNS INTEGER
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$;
				
				CREATE PROCEDURE add_user(user_name TEXT, user_email TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO users (name, email) VALUES (user_name, user_email);
				END;
				$$;
			`,
			currentSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255),
					email VARCHAR(255)
				);
				
				CREATE FUNCTION get_user_count() RETURNS INTEGER
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RETURN (SELECT COUNT(*) FROM users);
				END;
				$$;
				
				CREATE PROCEDURE update_user_email(user_id INTEGER, new_email TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					UPDATE users SET email = new_email WHERE id = user_id;
				END;
				$$;
			`,
			expectedFunctionChanges: 2, // Drop add_user + Create update_user_email
			expectedActions:         []schema.MetadataDiffAction{schema.MetadataDiffActionCreate, schema.MetadataDiffActionDrop},
		},
		{
			name: "Schema-qualified procedure",
			previousSDL: `
				CREATE SCHEMA admin;
				CREATE TABLE admin.settings (
					key VARCHAR(255) PRIMARY KEY,
					value TEXT
				);
			`,
			currentSDL: `
				CREATE SCHEMA admin;
				CREATE TABLE admin.settings (
					key VARCHAR(255) PRIMARY KEY,
					value TEXT
				);
				
				CREATE PROCEDURE admin.update_setting(setting_key TEXT, setting_value TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO admin.settings (key, value) VALUES (setting_key, setting_value)
					ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
				END;
				$$;
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

			// Note: In PostgreSQL, procedures are stored as functions
			assert.Equal(t, tt.expectedFunctionChanges, len(diff.FunctionChanges),
				"Expected %d function changes (including procedures), got %d", tt.expectedFunctionChanges, len(diff.FunctionChanges))

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

func TestProcedureSDLDiff_CommentOnProcedure(t *testing.T) {
	tests := []struct {
		name                  string
		previousSDL           string
		currentSDL            string
		expectedSQL           string
		shouldContain         []string
		shouldNotContain      []string
		expectedCommentChange int
	}{
		{
			name:        "Create procedure with COMMENT ON PROCEDURE",
			previousSDL: ``,
			currentSDL: `
				CREATE PROCEDURE new_procedure()
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RAISE NOTICE 'New procedure executed';
				END;
				$$;

				COMMENT ON PROCEDURE "public"."new_procedure"() IS 'A new procedure that raises a notice';
			`,
			shouldContain: []string{
				"CREATE PROCEDURE",
				"COMMENT ON PROCEDURE",
				`"public".new_procedure()`,
			},
			shouldNotContain: []string{
				"COMMENT ON FUNCTION",
			},
			expectedCommentChange: 1,
		},
		{
			name: "Create procedure with COMMENT ON PROCEDURE - ensure FUNCTION is not used",
			previousSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT
				);
			`,
			currentSDL: `
				CREATE TABLE logs (
					id SERIAL PRIMARY KEY,
					message TEXT
				);

				CREATE PROCEDURE log_message(msg TEXT)
				LANGUAGE plpgsql
				AS $$
				BEGIN
					INSERT INTO logs (message) VALUES (msg);
				END;
				$$;

				COMMENT ON PROCEDURE "public"."log_message"(msg text) IS 'Logs a message to the logs table';
			`,
			shouldContain: []string{
				"CREATE PROCEDURE",
				"COMMENT ON PROCEDURE",
				`log_message`,
			},
			shouldNotContain: []string{
				"COMMENT ON FUNCTION",
			},
			expectedCommentChange: 1,
		},
		{
			name:        "Mixed - procedure with comment and function with comment",
			previousSDL: ``,
			currentSDL: `
				CREATE FUNCTION test_function()
				RETURNS void
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RAISE NOTICE 'Test function executed';
				END;
				$$;

				COMMENT ON FUNCTION "public".test_function() IS 'A test function that raises a notice';

				CREATE PROCEDURE test_procedure()
				LANGUAGE plpgsql
				AS $$
				BEGIN
					RAISE NOTICE 'Test procedure executed';
				END;
				$$;

				COMMENT ON PROCEDURE "public"."test_procedure"() IS 'A test procedure that raises a notice';
			`,
			shouldContain: []string{
				"COMMENT ON FUNCTION",
				"test_function",
				"COMMENT ON PROCEDURE",
				"test_procedure",
			},
			shouldNotContain:      []string{},
			expectedCommentChange: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Verify comment changes
			assert.Equal(t, tt.expectedCommentChange, len(diff.CommentChanges),
				"Expected %d comment changes, got %d", tt.expectedCommentChange, len(diff.CommentChanges))

			// Generate migration SQL
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration SQL:\n%s", migrationSQL)

			// Verify expected strings are present
			for _, expected := range tt.shouldContain {
				assert.Contains(t, migrationSQL, expected,
					"Migration SQL should contain %q", expected)
			}

			// Verify unwanted strings are not present
			for _, unwanted := range tt.shouldNotContain {
				assert.NotContains(t, migrationSQL, unwanted,
					"Migration SQL should NOT contain %q", unwanted)
			}
		})
	}
}

func TestDropColumnWithComment_NoCommentGeneration(t *testing.T) {
	tests := []struct {
		name             string
		previousSDL      string
		currentSDL       string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Drop column with comment - should not generate COMMENT ON COLUMN",
			previousSDL: `
				CREATE TABLE "public"."test_table" (
					"id" serial,
					"name" character varying(100) NOT NULL,
					"test_column" integer,
					CONSTRAINT "test_table_pkey" PRIMARY KEY (id)
				);

				COMMENT ON TABLE "public"."test_table" IS 'A table for testing purposes';
				COMMENT ON COLUMN "public"."test_table"."test_column" IS 'A test column for various uses';
			`,
			currentSDL: `
				CREATE TABLE "public"."test_table" (
					"id" serial,
					"name" character varying(100) NOT NULL,
					CONSTRAINT "test_table_pkey" PRIMARY KEY (id)
				);

				COMMENT ON TABLE "public"."test_table" IS 'A table for testing purposes';
			`,
			shouldContain: []string{
				`ALTER TABLE "public"."test_table" DROP COLUMN IF EXISTS "test_column"`,
			},
			shouldNotContain: []string{
				`COMMENT ON COLUMN "public"."test_table"."test_column"`,
			},
		},
		{
			name: "Drop table with comment - should not generate COMMENT ON TABLE",
			previousSDL: `
				CREATE TABLE "public"."test_table" (
					"id" serial,
					"name" character varying(100) NOT NULL,
					CONSTRAINT "test_table_pkey" PRIMARY KEY (id)
				);

				COMMENT ON TABLE "public"."test_table" IS 'A table for testing purposes';
			`,
			currentSDL: ``,
			shouldContain: []string{
				`DROP TABLE IF EXISTS "public"."test_table"`,
			},
			shouldNotContain: []string{
				`COMMENT ON TABLE "public"."test_table"`,
			},
		},
		{
			name: "Drop multiple columns with comments",
			previousSDL: `
				CREATE TABLE "public"."test_table" (
					"id" serial,
					"name" character varying(100) NOT NULL,
					"created_at" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP,
					"is_active" boolean DEFAULT true,
					"description" text,
					"test_column" integer,
					CONSTRAINT "test_table_pkey" PRIMARY KEY (id)
				);

				COMMENT ON TABLE "public"."test_table" IS 'A table for testing purposes';
				COMMENT ON COLUMN "public"."test_table"."test_column" IS 'A test column for various uses';
				COMMENT ON COLUMN "public"."test_table"."description" IS 'Description field';
			`,
			currentSDL: `
				CREATE TABLE "public"."test_table" (
					"id" serial,
					"name" character varying(100) NOT NULL,
					"is_active" boolean DEFAULT true,
					CONSTRAINT "test_table_pkey" PRIMARY KEY (id)
				);

				COMMENT ON TABLE "public"."test_table" IS 'A table for testing purposes';
			`,
			shouldContain: []string{
				`ALTER TABLE "public"."test_table" DROP COLUMN IF EXISTS "test_column"`,
				`ALTER TABLE "public"."test_table" DROP COLUMN IF EXISTS "created_at"`,
				`ALTER TABLE "public"."test_table" DROP COLUMN IF EXISTS "description"`,
			},
			shouldNotContain: []string{
				`COMMENT ON COLUMN "public"."test_table"."test_column"`,
				`COMMENT ON COLUMN "public"."test_table"."description"`,
				`COMMENT ON COLUMN "public"."test_table"."created_at"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Generate migration SQL
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration SQL:\n%s", migrationSQL)

			// Verify expected strings are present
			for _, expected := range tt.shouldContain {
				assert.Contains(t, migrationSQL, expected,
					"Migration SQL should contain %q", expected)
			}

			// Verify unwanted strings are not present
			for _, unwanted := range tt.shouldNotContain {
				assert.NotContains(t, migrationSQL, unwanted,
					"Migration SQL should NOT contain %q", unwanted)
			}
		})
	}
}
func TestRemoveProcedureComment_ShouldUsePROCEDURE(t *testing.T) {
	tests := []struct {
		name             string
		previousSDL      string
		currentSDL       string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "Remove comment from procedure - should use COMMENT ON PROCEDURE, not FUNCTION",
			previousSDL: `
CREATE PROCEDURE "public"."new_procedure"()
LANGUAGE plpgsql
AS $$
BEGIN
	RAISE NOTICE 'New procedure executed';
END;
$$;

COMMENT ON PROCEDURE "public"."new_procedure"() IS 'A new procedure that raises a notice';
`,
			currentSDL: `
CREATE PROCEDURE "public"."new_procedure"()
LANGUAGE plpgsql
AS $$
BEGIN
	RAISE NOTICE 'New procedure executed';
END;
$$;
`,
			shouldContain: []string{
				"COMMENT ON PROCEDURE",
				`"public".new_procedure() IS NULL`,
			},
			shouldNotContain: []string{
				"COMMENT ON FUNCTION",
			},
		},
		{
			name: "Update procedure comment - should use COMMENT ON PROCEDURE",
			previousSDL: `
CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
	INSERT INTO logs (message) VALUES (msg);
END;
$$;

COMMENT ON PROCEDURE "public"."log_message"(msg text) IS 'Old comment';
`,
			currentSDL: `
CREATE PROCEDURE "public"."log_message"(msg text)
LANGUAGE plpgsql
AS $$
BEGIN
	INSERT INTO logs (message) VALUES (msg);
END;
$$;

COMMENT ON PROCEDURE "public"."log_message"(msg text) IS 'New comment';
`,
			shouldContain: []string{
				"COMMENT ON PROCEDURE",
				`"public".log_message(msg text) IS 'New comment'`,
			},
			shouldNotContain: []string{
				"COMMENT ON FUNCTION",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Generate migration SQL
			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration SQL:\n%s", migrationSQL)

			// Verify expected strings are present
			for _, expected := range tt.shouldContain {
				assert.Contains(t, migrationSQL, expected,
					"Migration SQL should contain %q", expected)
			}

			// Verify unwanted strings are not present
			for _, unwanted := range tt.shouldNotContain {
				assert.NotContains(t, migrationSQL, unwanted,
					"Migration SQL should NOT contain %q", unwanted)
			}
		})
	}
}
