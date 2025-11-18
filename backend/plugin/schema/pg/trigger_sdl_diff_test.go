package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestTriggerDiffStructure(t *testing.T) {
	// Test that TriggerDiff has all required fields for SDL processing
	currentSDL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100)
		);

		CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$
		BEGIN
			RAISE NOTICE 'Audit';
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit_log();
	`

	previousSDL := ""

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)
	require.Len(t, diff.TableChanges, 1, "Should have one table change")

	tableDiff := diff.TableChanges[0]
	require.Len(t, tableDiff.TriggerChanges, 1, "Should have one trigger change")

	triggerDiff := tableDiff.TriggerChanges[0]
	assert.Equal(t, schema.MetadataDiffActionCreate, triggerDiff.Action)
	assert.NotNil(t, triggerDiff.NewASTNode, "Should have AST node")
	// These fields should exist after we update the structure
	// assert.Equal(t, "public", triggerDiff.SchemaName)
	// assert.Equal(t, "users", triggerDiff.TableName)
	// assert.Equal(t, "audit_trigger", triggerDiff.TriggerName)
}

func TestCreateTriggerExtraction(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		expectedTriggers int
	}{
		{
			name: "Simple trigger",
			sql: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggers: 1,
		},
		{
			name: "Multiple triggers on same table",
			sql: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
				CREATE TRIGGER t2 BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggers: 2,
		},
		{
			name: "Trigger with schema-qualified table",
			sql: `
				CREATE SCHEMA app;
				CREATE TABLE app.users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION app.f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON app.users FOR EACH ROW EXECUTE FUNCTION app.f();
			`,
			expectedTriggers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.sql, "", nil, nil)
			require.NoError(t, err)

			totalTriggers := 0
			for _, tableDiff := range diff.TableChanges {
				totalTriggers += len(tableDiff.TriggerChanges)
			}

			assert.Equal(t, tt.expectedTriggers, totalTriggers)
		})
	}
}

func TestTriggerCommentExtraction(t *testing.T) {
	sql := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Audit log trigger';
	`

	diff, err := GetSDLDiff(sql, "", nil, nil)
	require.NoError(t, err)

	// Check that comment change exists
	found := false
	for _, commentDiff := range diff.CommentChanges {
		if commentDiff.ObjectType == schema.CommentObjectTypeTrigger &&
			commentDiff.ObjectName == "audit_trigger" &&
			commentDiff.NewComment == "Audit log trigger" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should have trigger comment")
}

func TestDropTriggerMigration(t *testing.T) {
	previousSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`
	currentSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
	`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	assert.Contains(t, migration, "DROP TRIGGER")
	assert.Contains(t, migration, "audit_trigger")
	assert.Contains(t, migration, "ON")
	assert.Contains(t, migration, "users")
}

func TestCreateTriggerMigrationWithDependencyOrder(t *testing.T) {
	currentSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit_log() RETURNS TRIGGER AS $$
		BEGIN
			RAISE NOTICE 'Audit';
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit_log();
	`

	diff, err := GetSDLDiff(currentSDL, "", nil, nil)
	require.NoError(t, err)

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	// Print migration for debugging
	t.Logf("Migration:\n%s", migration)

	// Verify CREATE OR REPLACE TRIGGER is in migration
	assert.Contains(t, migration, "CREATE OR REPLACE TRIGGER")
	assert.Contains(t, migration, "audit_trigger")

	// Verify dependency order: Both TABLE and FUNCTION must be created before TRIGGER
	// The order between TABLE and FUNCTION doesn't matter - either is valid in PostgreSQL
	// since trigger functions don't reference the table until the trigger is created
	tableIdx := strings.Index(migration, "CREATE TABLE")
	functionIdx := strings.Index(migration, "CREATE FUNCTION audit_log")
	triggerIdx := strings.Index(migration, "CREATE OR REPLACE TRIGGER audit_trigger")

	t.Logf("Indices: table=%d, function=%d, trigger=%d", tableIdx, functionIdx, triggerIdx)

	// Both table and function must exist before trigger
	assert.Less(t, tableIdx, triggerIdx, "Table must be created before trigger")
	assert.Less(t, functionIdx, triggerIdx, "Function must be created before trigger")
}

func TestTriggerCommentMigration(t *testing.T) {
	currentSDL := `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Audit log trigger';
	`

	diff, err := GetSDLDiff(currentSDL, "", nil, nil)
	require.NoError(t, err)

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	assert.Contains(t, migration, "COMMENT ON TRIGGER")
	assert.Contains(t, migration, "audit_trigger")
	assert.Contains(t, migration, "Audit log trigger")
}

func TestStandaloneTriggerDiff(t *testing.T) {
	tests := []struct {
		name                string
		previousUserSDL     string
		currentSDL          string
		expectedTriggerDiff int
		expectedActions     []schema.MetadataDiffAction
	}{
		{
			name:            "Create new trigger",
			previousUserSDL: "",
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggerDiff: 1,
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionCreate},
		},
		{
			name: "Drop trigger",
			previousUserSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			`,
			expectedTriggerDiff: 1,
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionDrop},
		},
		{
			name: "Modify trigger (CREATE OR REPLACE)",
			previousUserSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggerDiff: 1, // ALTER (CREATE OR REPLACE)
			expectedActions:     []schema.MetadataDiffAction{schema.MetadataDiffActionAlter},
		},
		{
			name: "No changes to trigger",
			previousUserSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			currentSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY);
				CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
				CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			`,
			expectedTriggerDiff: 0,
			expectedActions:     []schema.MetadataDiffAction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousUserSDL, nil, nil)
			require.NoError(t, err)

			totalTriggers := 0
			var actions []schema.MetadataDiffAction
			for _, tableDiff := range diff.TableChanges {
				for _, triggerDiff := range tableDiff.TriggerChanges {
					totalTriggers++
					actions = append(actions, triggerDiff.Action)
				}
			}

			assert.Equal(t, tt.expectedTriggerDiff, totalTriggers)
			if len(tt.expectedActions) == 0 {
				assert.Empty(t, actions)
			} else {
				assert.Equal(t, tt.expectedActions, actions)
			}
		})
	}
}

func TestTriggerSDLIntegration(t *testing.T) {
	t.Run("Multiple triggers on same table", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE orders (id SERIAL PRIMARY KEY, status VARCHAR(20));
			CREATE FUNCTION log_insert() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE FUNCTION log_update() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;

			CREATE TRIGGER insert_trigger AFTER INSERT ON orders FOR EACH ROW EXECUTE FUNCTION log_insert();
			CREATE TRIGGER update_trigger BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION log_update();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		totalTriggers := 0
		for _, tableDiff := range diff.TableChanges {
			totalTriggers += len(tableDiff.TriggerChanges)
		}
		assert.Equal(t, 2, totalTriggers, "Should have 2 triggers")

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "insert_trigger")
		assert.Contains(t, migration, "update_trigger")
	})

	t.Run("Trigger with schema-qualified table", func(t *testing.T) {
		currentSDL := `
			CREATE SCHEMA app;
			CREATE TABLE app.events (id SERIAL PRIMARY KEY);
			CREATE FUNCTION app.notify() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER event_trigger AFTER INSERT ON app.events FOR EACH ROW EXECUTE FUNCTION app.notify();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		found := false
		for _, tableDiff := range diff.TableChanges {
			if tableDiff.SchemaName == "app" && tableDiff.TableName == "events" {
				assert.Len(t, tableDiff.TriggerChanges, 1)
				found = true
			}
		}
		assert.True(t, found, "Should find trigger on app.events")
	})

	t.Run("Trigger with WHEN condition", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE products (id SERIAL PRIMARY KEY, price NUMERIC);
			CREATE FUNCTION check_price() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER price_check
			BEFORE UPDATE ON products
			FOR EACH ROW
			WHEN (NEW.price > 1000)
			EXECUTE FUNCTION check_price();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "WHEN")
		assert.Contains(t, migration, "NEW.price > 1000")
	})

	t.Run("Trigger modification preserves other triggers", func(t *testing.T) {
		previousSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			CREATE TRIGGER t2 AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
		`
		currentSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER t1 AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
			CREATE TRIGGER t2 AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Should have 1 change for t1 (ALTER using CREATE OR REPLACE), none for t2
		totalChanges := 0
		alterChanges := 0
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				totalChanges++
				if triggerDiff.Action == schema.MetadataDiffActionAlter {
					alterChanges++
				}
			}
		}
		assert.Equal(t, 1, totalChanges, "Should only modify t1 (ALTER)")
		assert.Equal(t, 1, alterChanges, "Should use ALTER action (CREATE OR REPLACE)")
	})

	t.Run("Trigger with multiple events", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE audit_log (id SERIAL PRIMARY KEY);
			CREATE FUNCTION log_changes() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER multi_event_trigger
			AFTER INSERT OR UPDATE OR DELETE ON audit_log
			FOR EACH ROW EXECUTE FUNCTION log_changes();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "CREATE OR REPLACE TRIGGER")
		assert.Contains(t, migration, "INSERT OR UPDATE OR DELETE")
	})

	t.Run("Trigger with FOR EACH STATEMENT", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE statements (id SERIAL PRIMARY KEY);
			CREATE FUNCTION stmt_trigger() RETURNS TRIGGER AS $$ BEGIN RETURN NULL; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER statement_level_trigger
			AFTER INSERT ON statements
			FOR EACH STATEMENT EXECUTE FUNCTION stmt_trigger();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "FOR EACH STATEMENT")
	})

	t.Run("Multiple comments on triggers", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE events (id SERIAL PRIMARY KEY);
			CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER t1 AFTER INSERT ON events FOR EACH ROW EXECUTE FUNCTION f();
			CREATE TRIGGER t2 AFTER UPDATE ON events FOR EACH ROW EXECUTE FUNCTION f();
			COMMENT ON TRIGGER t1 ON events IS 'Insert trigger';
			COMMENT ON TRIGGER t2 ON events IS 'Update trigger';
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		// Check both comments are extracted
		triggerCommentCount := 0
		for _, commentDiff := range diff.CommentChanges {
			if commentDiff.ObjectType == schema.CommentObjectTypeTrigger {
				triggerCommentCount++
			}
		}
		assert.Equal(t, 2, triggerCommentCount, "Should have 2 trigger comments")
	})

	t.Run("Drop trigger removes comment", func(t *testing.T) {
		previousSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
			COMMENT ON TRIGGER t1 ON users IS 'My trigger';
		`
		currentSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Should have trigger drop
		found := false
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionDrop {
					found = true
				}
			}
		}
		assert.True(t, found, "Should have trigger drop action")
	})

	t.Run("Trigger referencing columns", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE inventory (id SERIAL PRIMARY KEY, stock INT);
			CREATE FUNCTION check_stock() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER stock_check
			BEFORE UPDATE OF stock ON inventory
			FOR EACH ROW EXECUTE FUNCTION check_stock();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "UPDATE OF stock")
	})

	t.Run("INSTEAD OF trigger on view", func(t *testing.T) {
		currentSDL := `
			CREATE TABLE base_table (id SERIAL PRIMARY KEY);
			CREATE VIEW my_view AS SELECT * FROM base_table;
			CREATE FUNCTION instead_trigger() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER view_trigger
			INSTEAD OF INSERT ON my_view
			FOR EACH ROW EXECUTE FUNCTION instead_trigger();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)
		assert.Contains(t, migration, "INSTEAD OF INSERT")
	})

	t.Run("CREATE OR REPLACE conversion test", func(t *testing.T) {
		// Test that CREATE TRIGGER is converted to CREATE OR REPLACE TRIGGER
		currentSDL := `
			CREATE TABLE test_table (id SERIAL PRIMARY KEY);
			CREATE FUNCTION test_func() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER test_trigger
			AFTER INSERT ON test_table
			FOR EACH ROW EXECUTE FUNCTION test_func();
		`

		diff, err := GetSDLDiff(currentSDL, "", nil, nil)
		require.NoError(t, err)

		migration, err := generateMigration(diff)
		require.NoError(t, err)

		// Should contain CREATE OR REPLACE TRIGGER, not just CREATE TRIGGER
		assert.Contains(t, migration, "CREATE OR REPLACE TRIGGER")
		assert.NotContains(t, migration, "CREATE TRIGGER test_trigger\n")
	})

	t.Run("Trigger modification uses CREATE OR REPLACE", func(t *testing.T) {
		// Test that modifying a trigger uses CREATE OR REPLACE instead of DROP + CREATE
		previousSDL := `
			CREATE TABLE test_table (id SERIAL PRIMARY KEY);
			CREATE FUNCTION test_func() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER test_trigger
			AFTER INSERT ON test_table
			FOR EACH ROW EXECUTE FUNCTION test_func();
		`

		currentSDL := `
			CREATE TABLE test_table (id SERIAL PRIMARY KEY);
			CREATE FUNCTION test_func() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER test_trigger
			AFTER INSERT OR UPDATE ON test_table
			FOR EACH ROW EXECUTE FUNCTION test_func();
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Should have 1 ALTER action (not DROP + CREATE)
		var alterCount int
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionAlter {
					alterCount++
				}
			}
		}
		assert.Equal(t, 1, alterCount, "Should use ALTER action")

		migration, err := generateMigration(diff)
		require.NoError(t, err)

		// Should use CREATE OR REPLACE
		assert.Contains(t, migration, "CREATE OR REPLACE TRIGGER")
		// Should NOT contain DROP TRIGGER
		assert.NotContains(t, migration, "DROP TRIGGER")
	})

	t.Run("Modify trigger comment only", func(t *testing.T) {
		// Test that modifying only the trigger comment doesn't trigger ALTER on the trigger itself
		previousSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
			COMMENT ON TRIGGER audit_trigger ON users IS 'Original comment';
		`

		currentSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
			COMMENT ON TRIGGER audit_trigger ON users IS 'Updated comment';
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Trigger itself should NOT have ALTER action (definition unchanged)
		triggerAlterCount := 0
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionAlter {
					triggerAlterCount++
				}
			}
		}
		assert.Equal(t, 0, triggerAlterCount, "Trigger should not have ALTER action when only comment changes")

		// Should have comment change
		commentChanged := false
		for _, commentDiff := range diff.CommentChanges {
			if commentDiff.ObjectType == schema.CommentObjectTypeTrigger &&
				commentDiff.ObjectName == "audit_trigger" &&
				commentDiff.OldComment == "Original comment" &&
				commentDiff.NewComment == "Updated comment" {
				commentChanged = true
			}
		}
		assert.True(t, commentChanged, "Should have trigger comment change")

		// Migration should only contain COMMENT statement, not CREATE OR REPLACE TRIGGER
		migration, err := generateMigration(diff)
		require.NoError(t, err)

		assert.Contains(t, migration, "COMMENT ON TRIGGER", "Should have COMMENT ON TRIGGER")
		assert.Contains(t, migration, "Updated comment", "Should have new comment text")
		assert.NotContains(t, migration, "CREATE OR REPLACE TRIGGER", "Should NOT recreate trigger when only comment changes")
	})

	t.Run("Add comment to trigger without comment", func(t *testing.T) {
		// Test adding a comment to a trigger that previously had no comment
		previousSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
		`

		currentSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
			COMMENT ON TRIGGER audit_trigger ON users IS 'New comment';
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Trigger should NOT have ALTER action
		triggerAlterCount := 0
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionAlter {
					triggerAlterCount++
				}
			}
		}
		assert.Equal(t, 0, triggerAlterCount, "Trigger should not have ALTER action when adding comment")

		// Should have comment addition
		commentAdded := false
		for _, commentDiff := range diff.CommentChanges {
			if commentDiff.ObjectType == schema.CommentObjectTypeTrigger &&
				commentDiff.ObjectName == "audit_trigger" &&
				commentDiff.Action == schema.MetadataDiffActionCreate &&
				commentDiff.NewComment == "New comment" {
				commentAdded = true
			}
		}
		assert.True(t, commentAdded, "Should have comment addition")

		migration, err := generateMigration(diff)
		require.NoError(t, err)

		assert.Contains(t, migration, "COMMENT ON TRIGGER", "Should have COMMENT ON TRIGGER")
		assert.NotContains(t, migration, "CREATE OR REPLACE TRIGGER", "Should NOT recreate trigger")
	})

	t.Run("Remove trigger comment", func(t *testing.T) {
		// Test removing a comment from a trigger
		previousSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
			COMMENT ON TRIGGER audit_trigger ON users IS 'Old comment';
		`

		currentSDL := `
			CREATE TABLE users (id SERIAL PRIMARY KEY);
			CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
			CREATE TRIGGER audit_trigger
			AFTER INSERT ON users
			FOR EACH ROW EXECUTE FUNCTION audit();
		`

		diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
		require.NoError(t, err)

		// Trigger should NOT have ALTER action
		triggerAlterCount := 0
		for _, tableDiff := range diff.TableChanges {
			for _, triggerDiff := range tableDiff.TriggerChanges {
				if triggerDiff.Action == schema.MetadataDiffActionAlter {
					triggerAlterCount++
				}
			}
		}
		assert.Equal(t, 0, triggerAlterCount, "Trigger should not have ALTER action when removing comment")

		// Should have comment drop
		commentDropped := false
		for _, commentDiff := range diff.CommentChanges {
			if commentDiff.ObjectType == schema.CommentObjectTypeTrigger &&
				commentDiff.ObjectName == "audit_trigger" &&
				commentDiff.Action == schema.MetadataDiffActionDrop {
				commentDropped = true
			}
		}
		assert.True(t, commentDropped, "Should have comment drop action")

		migration, err := generateMigration(diff)
		require.NoError(t, err)

		assert.Contains(t, migration, "COMMENT ON TRIGGER", "Should have COMMENT ON TRIGGER")
		assert.Contains(t, migration, "NULL", "Should set comment to NULL")
		assert.NotContains(t, migration, "CREATE OR REPLACE TRIGGER", "Should NOT recreate trigger")
	})
}
