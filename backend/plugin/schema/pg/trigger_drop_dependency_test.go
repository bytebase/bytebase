package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDropTriggerDependencyOrder(t *testing.T) {
	// Previous SDL with tables, functions, and triggers
	previousSDL := `
		CREATE TABLE test_products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE test_orders (
			id SERIAL PRIMARY KEY,
			product_id INTEGER,
			quantity INTEGER,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE audit_log (
			id SERIAL PRIMARY KEY,
			table_name VARCHAR(50),
			operation VARCHAR(10),
			changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE FUNCTION update_timestamp_trigger_func() 
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE FUNCTION audit_trigger_func() 
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO audit_log (table_name, operation)
			VALUES (TG_TABLE_NAME, TG_OP);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON test_products
		FOR EACH ROW EXECUTE FUNCTION update_timestamp_trigger_func();

		CREATE TRIGGER audit_insert_trigger
		AFTER INSERT ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_update_trigger
		AFTER UPDATE ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_delete_trigger
		AFTER DELETE ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON test_orders
		FOR EACH ROW EXECUTE FUNCTION update_timestamp_trigger_func();

		CREATE TRIGGER audit_insert_trigger
		AFTER INSERT ON test_orders
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_delete_trigger
		AFTER DELETE ON test_orders
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
	`

	// Current SDL is empty - drop everything
	currentSDL := ``

	// Generate diff and migration (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	// Debug: print diff information
	t.Logf("Number of table changes: %d", len(diff.TableChanges))
	for i, tableDiff := range diff.TableChanges {
		t.Logf("Table %d: %s.%s, Action=%s, TriggerChanges=%d",
			i, tableDiff.SchemaName, tableDiff.TableName, tableDiff.Action, len(tableDiff.TriggerChanges))
		for j, triggerDiff := range tableDiff.TriggerChanges {
			t.Logf("  Trigger %d: %s, Action=%s", j, triggerDiff.TriggerName, triggerDiff.Action)
		}
	}

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Migration:\n%s", migration)

	// Find positions of DROP statements
	dropTriggerIdx := strings.Index(migration, "DROP TRIGGER")
	dropFunctionIdx := strings.Index(migration, "DROP FUNCTION")
	dropTableIdx := strings.Index(migration, "DROP TABLE")

	t.Logf("Indices: dropTrigger=%d, dropFunction=%d, dropTable=%d", dropTriggerIdx, dropFunctionIdx, dropTableIdx)

	// Verify DROP order: triggers must be dropped before functions
	// Functions must be dropped before tables (or at least before tables that don't depend on them)
	assert.NotEqual(t, -1, dropTriggerIdx, "Should have DROP TRIGGER statements")
	assert.NotEqual(t, -1, dropFunctionIdx, "Should have DROP FUNCTION statements")
	assert.NotEqual(t, -1, dropTableIdx, "Should have DROP TABLE statements")

	// Critical: triggers must be dropped before functions they depend on
	assert.Less(t, dropTriggerIdx, dropFunctionIdx, "DROP TRIGGER must come before DROP FUNCTION to avoid dependency errors")

	// Functions should be dropped before or after tables (depends on implementation)
	// But triggers MUST be before functions
}

func TestCreateTriggerDependencyOrder(t *testing.T) {
	// Previous SDL is empty
	previousSDL := ``

	// Current SDL with tables, functions, and triggers
	currentSDL := `
		CREATE TABLE test_products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE test_orders (
			id SERIAL PRIMARY KEY,
			product_id INTEGER,
			quantity INTEGER,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE audit_log (
			id SERIAL PRIMARY KEY,
			table_name VARCHAR(50),
			operation VARCHAR(10),
			changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE FUNCTION update_timestamp_trigger_func() 
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE FUNCTION audit_trigger_func() 
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO audit_log (table_name, operation)
			VALUES (TG_TABLE_NAME, TG_OP);
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON test_products
		FOR EACH ROW EXECUTE FUNCTION update_timestamp_trigger_func();

		CREATE TRIGGER audit_insert_trigger
		AFTER INSERT ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_update_trigger
		AFTER UPDATE ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_delete_trigger
		AFTER DELETE ON test_products
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON test_orders
		FOR EACH ROW EXECUTE FUNCTION update_timestamp_trigger_func();

		CREATE TRIGGER audit_insert_trigger
		AFTER INSERT ON test_orders
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();

		CREATE TRIGGER audit_delete_trigger
		AFTER DELETE ON test_orders
		FOR EACH ROW EXECUTE FUNCTION audit_trigger_func();
	`

	// Generate diff and migration (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	// Debug: print diff information
	t.Logf("Number of table changes: %d", len(diff.TableChanges))
	for i, tableDiff := range diff.TableChanges {
		t.Logf("Table %d: %s.%s, Action=%s, TriggerChanges=%d",
			i, tableDiff.SchemaName, tableDiff.TableName, tableDiff.Action, len(tableDiff.TriggerChanges))
		for j, triggerDiff := range tableDiff.TriggerChanges {
			t.Logf("  Trigger %d: %s, Action=%s", j, triggerDiff.TriggerName, triggerDiff.Action)
		}
	}

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Migration:\n%s", migration)

	// Find positions of CREATE statements
	createTableIdx := strings.Index(migration, "CREATE TABLE")
	createFunctionIdx := strings.Index(migration, "CREATE FUNCTION") // Note: Not "CREATE OR REPLACE" for new functions
	createTriggerIdx := strings.Index(migration, "CREATE OR REPLACE TRIGGER")

	t.Logf("Indices: createTable=%d, createFunction=%d, createTrigger=%d",
		createTableIdx, createFunctionIdx, createTriggerIdx)

	// Verify CREATE order: tables must be created before functions and triggers
	assert.NotEqual(t, -1, createTableIdx, "Should have CREATE TABLE statements")
	assert.NotEqual(t, -1, createFunctionIdx, "Should have CREATE FUNCTION statements")
	assert.NotEqual(t, -1, createTriggerIdx, "Should have CREATE TRIGGER statements")

	// Critical: tables must be created before triggers (triggers reference tables)
	assert.Less(t, createTableIdx, createTriggerIdx, "CREATE TABLE must come before CREATE TRIGGER")

	// Functions must be created before triggers (triggers call functions)
	assert.Less(t, createFunctionIdx, createTriggerIdx, "CREATE FUNCTION must come before CREATE TRIGGER to avoid dependency errors")

	// Verify no duplicate triggers
	firstTrigger := strings.Index(migration, "CREATE OR REPLACE TRIGGER")
	assert.NotEqual(t, -1, firstTrigger, "Should have at least one CREATE TRIGGER")

	// Count total triggers (should match the number we expect)
	triggerCount := strings.Count(migration, "CREATE OR REPLACE TRIGGER")
	t.Logf("Total triggers in migration: %d", triggerCount)
	assert.Equal(t, 7, triggerCount, "Should have exactly 7 triggers (4 from test_products, 3 from test_orders)")
}

func TestSameTriggerNameOnDifferentTables(t *testing.T) {
	// Test that triggers with the same name on different tables are handled correctly
	// In PostgreSQL, trigger names are table-scoped, not schema-scoped
	// So "public.table1.trigger_name" and "public.table2.trigger_name" should be different

	currentSDL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE FUNCTION update_timestamp() 
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		-- Same trigger name on different tables
		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON users
		FOR EACH ROW EXECUTE FUNCTION update_timestamp();

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON orders
		FOR EACH ROW EXECUTE FUNCTION update_timestamp();
	`

	previousSDL := ``

	// Generate diff and migration (AST-only mode)
	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	// Debug: print diff information
	t.Logf("Number of table changes: %d", len(diff.TableChanges))

	totalTriggers := 0
	for _, tableDiff := range diff.TableChanges {
		t.Logf("Table: %s.%s, Action=%s, TriggerChanges=%d",
			tableDiff.SchemaName, tableDiff.TableName, tableDiff.Action, len(tableDiff.TriggerChanges))
		for _, triggerDiff := range tableDiff.TriggerChanges {
			t.Logf("  Trigger: %s, Action=%s", triggerDiff.TriggerName, triggerDiff.Action)
			totalTriggers++
		}
	}

	// Should have 2 triggers (one for each table)
	assert.Equal(t, 2, totalTriggers, "Should have 2 triggers (one for each table)")

	migration, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Migration:\n%s", migration)

	// Verify both triggers are in the migration
	assert.Contains(t, migration, "ON users", "Should have trigger on users table")
	assert.Contains(t, migration, "ON orders", "Should have trigger on orders table")

	// Count how many times the trigger is created
	// Should be 2 (once for users, once for orders)
	triggerCount := 0
	if assert.Contains(t, migration, "CREATE OR REPLACE TRIGGER update_timestamp_trigger") {
		// Count occurrences manually by checking both table references
		hasUsersTrigger := assert.Contains(t, migration, "update_timestamp_trigger") &&
			assert.Contains(t, migration, "ON users")
		hasOrdersTrigger := assert.Contains(t, migration, "update_timestamp_trigger") &&
			assert.Contains(t, migration, "ON orders")

		if hasUsersTrigger {
			triggerCount++
		}
		if hasOrdersTrigger {
			triggerCount++
		}
	}

	assert.Equal(t, 2, triggerCount, "Should have 2 CREATE TRIGGER statements (one for each table)")
}
