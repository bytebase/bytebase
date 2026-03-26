package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniProcedureSDLDiff_CreateNew(t *testing.T) {
	sql := omniSDLMigration(t, "", `
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
	`)
	require.Contains(t, sql, "CREATE PROCEDURE")
	require.Contains(t, sql, "log_message")
}

func TestOmniProcedureSDLDiff_Drop(t *testing.T) {
	sql := omniSDLMigration(t, `
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
	`, `
		CREATE TABLE logs (
			id SERIAL PRIMARY KEY,
			message TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.Contains(t, sql, "DROP PROCEDURE")
	require.Contains(t, sql, "log_message")
}

func TestOmniProcedureSDLDiff_Modify(t *testing.T) {
	sql := omniSDLMigration(t, `
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
	`, `
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
	`)
	require.Contains(t, sql, "CREATE OR REPLACE PROCEDURE")
	require.Contains(t, sql, "log_message")
}

func TestOmniProcedureSDLDiff_MixedFunctionsAndProcedures(t *testing.T) {
	sql := omniSDLMigration(t, `
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
	`, `
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
	`)
	require.Contains(t, sql, "DROP PROCEDURE")
	require.Contains(t, sql, "add_user")
	require.Contains(t, sql, "CREATE PROCEDURE")
	require.Contains(t, sql, "update_user_email")
	require.NotContains(t, sql, "get_user_count")
}

func TestOmniProcedureSDLDiff_SchemaQualified(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE SCHEMA admin;
		CREATE TABLE admin.settings (
			key VARCHAR(255) PRIMARY KEY,
			value TEXT
		);
	`, `
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
	`)
	require.Contains(t, sql, "CREATE PROCEDURE")
	require.Contains(t, sql, "update_setting")
}

func TestOmniProcedureSDLDiff_CommentOnProcedure_Create(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE PROCEDURE new_procedure()
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'New procedure executed';
		END;
		$$;

		COMMENT ON PROCEDURE new_procedure() IS 'A new procedure that raises a notice';
	`)
	require.Contains(t, sql, "CREATE PROCEDURE")
	require.Contains(t, sql, "COMMENT ON PROCEDURE")
	require.Contains(t, sql, "new_procedure")
	require.Contains(t, sql, "A new procedure that raises a notice")
}

func TestOmniProcedureSDLDiff_CommentOnProcedure_EnsureNotFunction(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE logs (
			id SERIAL PRIMARY KEY,
			message TEXT
		);
	`, `
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

		COMMENT ON PROCEDURE log_message(text) IS 'Logs a message to the logs table';
	`)
	require.Contains(t, sql, "CREATE PROCEDURE")
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "log_message")
}

func TestOmniProcedureSDLDiff_MixedComments(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE FUNCTION test_function()
		RETURNS void
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'Test function executed';
		END;
		$$;

		COMMENT ON FUNCTION test_function() IS 'A test function that raises a notice';

		CREATE PROCEDURE test_procedure()
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'Test procedure executed';
		END;
		$$;

		COMMENT ON PROCEDURE test_procedure() IS 'A test procedure that raises a notice';
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "test_function")
	require.Contains(t, sql, "test_procedure")
	require.Contains(t, sql, "A test function that raises a notice")
	require.Contains(t, sql, "A test procedure that raises a notice")
}

func TestOmniProcedureSDLDiff_DropColumnWithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE test_table (
			id serial PRIMARY KEY,
			name character varying(100) NOT NULL,
			test_column integer
		);

		COMMENT ON TABLE test_table IS 'A table for testing purposes';
		COMMENT ON COLUMN test_table.test_column IS 'A test column for various uses';
	`, `
		CREATE TABLE test_table (
			id serial PRIMARY KEY,
			name character varying(100) NOT NULL
		);

		COMMENT ON TABLE test_table IS 'A table for testing purposes';
	`)
	require.Contains(t, sql, "DROP COLUMN")
	require.Contains(t, sql, "test_column")
}

func TestOmniProcedureSDLDiff_DropTableWithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE test_table (
			id serial PRIMARY KEY,
			name character varying(100) NOT NULL
		);

		COMMENT ON TABLE test_table IS 'A table for testing purposes';
	`, "")
	require.Contains(t, sql, "DROP TABLE")
	require.Contains(t, sql, "test_table")
}

func TestOmniProcedureSDLDiff_RemoveProcedureComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE PROCEDURE new_procedure()
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'New procedure executed';
		END;
		$$;

		COMMENT ON PROCEDURE new_procedure() IS 'A new procedure that raises a notice';
	`, `
		CREATE PROCEDURE new_procedure()
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'New procedure executed';
		END;
		$$;
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "new_procedure")
	require.Contains(t, sql, "NULL")
}

func TestOmniProcedureSDLDiff_UpdateProcedureComment(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE PROCEDURE log_message(msg text)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE '%', msg;
		END;
		$$;

		COMMENT ON PROCEDURE log_message(text) IS 'Old comment';
	`, `
		CREATE PROCEDURE log_message(msg text)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE '%', msg;
		END;
		$$;

		COMMENT ON PROCEDURE log_message(text) IS 'New comment';
	`)
	require.Contains(t, sql, "COMMENT ON")
	require.Contains(t, sql, "New comment")
	require.Contains(t, sql, "log_message")
}
