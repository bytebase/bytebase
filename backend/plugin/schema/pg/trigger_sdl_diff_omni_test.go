package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniTriggerDiff_CreateTrigger(t *testing.T) {
	sql := omniSDLMigration(t, `
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
	`, `
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
	`)
	require.Contains(t, sql, "CREATE")
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "audit_trigger")
}

func TestOmniTriggerDiff_SimpleTrigger(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_MultipleTriggersSameTable(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		CREATE TRIGGER t2 BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Contains(t, sql, "t1")
	require.Contains(t, sql, "t2")
}

func TestOmniTriggerDiff_SchemaQualifiedTable(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE SCHEMA app;
		CREATE TABLE app.users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION app.f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON app.users FOR EACH ROW EXECUTE FUNCTION app.f();
	`)
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_TriggerComment(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Audit log trigger';
	`)
	require.Contains(t, sql, "COMMENT ON TRIGGER")
	require.Contains(t, sql, "audit_trigger")
	require.Contains(t, sql, "Audit log trigger")
}

func TestOmniTriggerDiff_DropTrigger(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
	`)
	require.Contains(t, sql, "DROP TRIGGER")
	require.Contains(t, sql, "audit_trigger")
	require.Contains(t, sql, "users")
}

func TestOmniTriggerDiff_CreateWithDependencyOrder(t *testing.T) {
	sql := omniSDLMigration(t, "", `
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
	`)

	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "audit_trigger")

	tableIdx := strings.Index(sql, "CREATE TABLE")
	functionIdx := strings.Index(sql, "CREATE FUNCTION")
	// The omni engine uses CREATE TRIGGER (not CREATE OR REPLACE TRIGGER)
	triggerIdx := strings.Index(sql, "CREATE TRIGGER")

	require.NotEqual(t, -1, tableIdx, "Table must be created")
	require.NotEqual(t, -1, functionIdx, "Function must be created")
	require.NotEqual(t, -1, triggerIdx, "Trigger must be created")

	require.Less(t, tableIdx, triggerIdx, "Table must be created before trigger")
	require.Less(t, functionIdx, triggerIdx, "Function must be created before trigger")
}

func TestOmniTriggerDiff_StandaloneCreate(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_StandaloneDrop(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
	`)
	require.Contains(t, sql, "DROP TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_StandaloneModify(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_StandaloneNoChange(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Empty(t, sql)
}

func TestOmniTriggerDiff_MultipleTriggersIntegration(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE orders (id SERIAL PRIMARY KEY, status VARCHAR(20));
		CREATE FUNCTION log_insert() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE FUNCTION log_update() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;

		CREATE TRIGGER insert_trigger AFTER INSERT ON orders FOR EACH ROW EXECUTE FUNCTION log_insert();
		CREATE TRIGGER update_trigger BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION log_update();
	`)
	require.Contains(t, sql, "insert_trigger")
	require.Contains(t, sql, "update_trigger")
}

func TestOmniTriggerDiff_SchemaQualifiedIntegration(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE SCHEMA app;
		CREATE TABLE app.events (id SERIAL PRIMARY KEY);
		CREATE FUNCTION app.notify() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER event_trigger AFTER INSERT ON app.events FOR EACH ROW EXECUTE FUNCTION app.notify();
	`)
	require.Contains(t, sql, "event_trigger")
	require.Contains(t, sql, "app")
}

func TestOmniTriggerDiff_WhenCondition(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE products (id SERIAL PRIMARY KEY, price NUMERIC);
		CREATE FUNCTION check_price() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER price_check
		BEFORE UPDATE ON products
		FOR EACH ROW
		WHEN (NEW.price > 1000)
		EXECUTE FUNCTION check_price();
	`)
	require.Contains(t, sql, "WHEN")
	require.Contains(t, sql, "price")
}

func TestOmniTriggerDiff_ModifyPreservesOther(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		CREATE TRIGGER t2 AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
		CREATE TRIGGER t2 AFTER UPDATE ON users FOR EACH ROW EXECUTE FUNCTION f();
	`)
	require.Contains(t, sql, "t1")
	require.NotContains(t, sql, "t2")
}

func TestOmniTriggerDiff_MultipleEvents(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE audit_log (id SERIAL PRIMARY KEY);
		CREATE FUNCTION log_changes() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER multi_event_trigger
		AFTER INSERT OR UPDATE OR DELETE ON audit_log
		FOR EACH ROW EXECUTE FUNCTION log_changes();
	`)
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "INSERT OR UPDATE OR DELETE")
}

func TestOmniTriggerDiff_ForEachStatement(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE statements (id SERIAL PRIMARY KEY);
		CREATE FUNCTION stmt_trigger() RETURNS TRIGGER AS $$ BEGIN RETURN NULL; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER statement_level_trigger
		AFTER INSERT ON statements
		FOR EACH STATEMENT EXECUTE FUNCTION stmt_trigger();
	`)
	require.Contains(t, sql, "FOR EACH STATEMENT")
}

func TestOmniTriggerDiff_MultipleComments(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE events (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON events FOR EACH ROW EXECUTE FUNCTION f();
		CREATE TRIGGER t2 AFTER UPDATE ON events FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER t1 ON events IS 'Insert trigger';
		COMMENT ON TRIGGER t2 ON events IS 'Update trigger';
	`)
	require.Contains(t, sql, "Insert trigger")
	require.Contains(t, sql, "Update trigger")
}

func TestOmniTriggerDiff_DropTriggerWithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER t1 AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION f();
		COMMENT ON TRIGGER t1 ON users IS 'My trigger';
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION f() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
	`)
	require.Contains(t, sql, "DROP TRIGGER")
	require.Contains(t, sql, "t1")
}

func TestOmniTriggerDiff_ReferencingColumns(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE inventory (id SERIAL PRIMARY KEY, stock INT);
		CREATE FUNCTION check_stock() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER stock_check
		BEFORE UPDATE OF stock ON inventory
		FOR EACH ROW EXECUTE FUNCTION check_stock();
	`)
	require.Contains(t, sql, "UPDATE OF")
	require.Contains(t, sql, "stock_check")
	require.Contains(t, sql, "TRIGGER")
	require.Contains(t, sql, "inventory")
}

func TestOmniTriggerDiff_ModifyCommentOnly(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Original comment';
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Updated comment';
	`)
	require.Contains(t, sql, "COMMENT ON TRIGGER")
	require.Contains(t, sql, "Updated comment")
}

func TestOmniTriggerDiff_AddCommentToExisting(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
		COMMENT ON TRIGGER audit_trigger ON users IS 'New comment';
	`)
	require.Contains(t, sql, "COMMENT ON TRIGGER")
	require.Contains(t, sql, "New comment")
}

func TestOmniTriggerDiff_RemoveComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
		COMMENT ON TRIGGER audit_trigger ON users IS 'Old comment';
	`, `
		CREATE TABLE users (id SERIAL PRIMARY KEY);
		CREATE FUNCTION audit() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER audit_trigger
		AFTER INSERT ON users
		FOR EACH ROW EXECUTE FUNCTION audit();
	`)
	require.Contains(t, sql, "COMMENT ON TRIGGER")
	require.Contains(t, sql, "NULL")
}
