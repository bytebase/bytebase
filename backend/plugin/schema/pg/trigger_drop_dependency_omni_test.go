package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniDropTriggerDependencyOrder(t *testing.T) {
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

	sql := omniSDLMigration(t, previousSDL, "")

	t.Logf("Migration:\n%s", sql)

	dropFunctionIdx := strings.Index(sql, "DROP FUNCTION")
	dropTableIdx := strings.Index(sql, "DROP TABLE")

	require.NotEqual(t, -1, dropFunctionIdx, "Should have DROP FUNCTION statements")
	require.NotEqual(t, -1, dropTableIdx, "Should have DROP TABLE statements")

	// The omni engine uses DROP TABLE CASCADE which handles triggers implicitly.
	// If DROP TRIGGER is present, it must come before DROP FUNCTION.
	dropTriggerIdx := strings.Index(sql, "DROP TRIGGER")
	if dropTriggerIdx != -1 {
		require.Less(t, dropTriggerIdx, dropFunctionIdx, "DROP TRIGGER must come before DROP FUNCTION")
	}

	// Tables are dropped with CASCADE before functions (CASCADE handles trigger deps)
	require.Less(t, dropTableIdx, dropFunctionIdx, "DROP TABLE CASCADE must come before DROP FUNCTION")
}

func TestOmniCreateTriggerDependencyOrder(t *testing.T) {
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

	sql := omniSDLMigration(t, "", currentSDL)

	t.Logf("Migration:\n%s", sql)

	createTableIdx := strings.Index(sql, "CREATE TABLE")
	createFunctionIdx := strings.Index(sql, "CREATE FUNCTION")
	createTriggerIdx := strings.Index(sql, "CREATE TRIGGER")

	require.NotEqual(t, -1, createTableIdx, "Should have CREATE TABLE statements")
	require.NotEqual(t, -1, createFunctionIdx, "Should have CREATE FUNCTION statements")
	require.NotEqual(t, -1, createTriggerIdx, "Should have CREATE TRIGGER statements")

	// Critical: tables must be created before triggers
	require.Less(t, createTableIdx, createTriggerIdx, "CREATE TABLE must come before CREATE TRIGGER")

	// Critical: functions must be created before triggers
	require.Less(t, createFunctionIdx, createTriggerIdx, "CREATE FUNCTION must come before CREATE TRIGGER")

	// Count total triggers
	triggerCount := strings.Count(sql, "CREATE TRIGGER")
	require.Equal(t, 7, triggerCount, "Should have exactly 7 triggers")
}

func TestOmniSameTriggerNameOnDifferentTables(t *testing.T) {
	sql := omniSDLMigration(t, "", `
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

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON users
		FOR EACH ROW EXECUTE FUNCTION update_timestamp();

		CREATE TRIGGER update_timestamp_trigger
		BEFORE UPDATE ON orders
		FOR EACH ROW EXECUTE FUNCTION update_timestamp();
	`)

	t.Logf("Migration:\n%s", sql)

	require.Contains(t, sql, "update_timestamp_trigger")
	require.Contains(t, sql, "users")
	require.Contains(t, sql, "orders")
}
