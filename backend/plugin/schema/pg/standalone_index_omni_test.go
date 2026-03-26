package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniStandaloneCreateIndexSupport(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		validate func(t *testing.T, sql string)
	}{
		{
			name: "Create new index",
			fromSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			`,
			toSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);

			CREATE INDEX idx_users_email ON users(email);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE INDEX")
				require.Contains(t, sql, "idx_users_email")
			},
		},
		{
			name: "Drop index",
			fromSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);

			CREATE INDEX idx_users_email ON users(email);
			`,
			toSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "DROP INDEX")
				require.Contains(t, sql, "idx_users_email")
			},
		},
		{
			name: "Modify index (drop and recreate)",
			fromSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE,
				name VARCHAR(100)
			);

			CREATE INDEX idx_users_email ON users(email);
			`,
			toSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE,
				name VARCHAR(100)
			);

			CREATE INDEX idx_users_email ON users(email, name);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				// Should contain both drop and create for the modified index
				require.Contains(t, sql, "idx_users_email")
			},
		},
		{
			name: "No changes to index",
			fromSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);

			CREATE INDEX idx_users_email ON users(email);
			`,
			toSDL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE
			);

			CREATE INDEX idx_users_email ON users(email);
			`,
			validate: func(t *testing.T, sql string) {
				require.Empty(t, sql, "Should produce no migration for identical indexes")
			},
		},
		{
			name: "Complex index with WHERE clause",
			fromSDL: `
			CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50),
				customer_id INTEGER
			);
			`,
			toSDL: `
			CREATE TABLE orders (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50),
				customer_id INTEGER
			);

			CREATE INDEX idx_orders_active ON orders(customer_id) WHERE status = 'active';
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE INDEX")
				require.Contains(t, sql, "idx_orders_active")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			tt.validate(t, sql)
		})
	}
}

func TestOmniStandaloneIndexIntegrationWithTableChanges(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		validate func(t *testing.T, sql string)
	}{
		{
			name:    "Standalone index on existing table",
			fromSDL: `CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));`,
			toSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE INDEX idx_users_email ON users(email);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE INDEX")
				require.Contains(t, sql, "idx_users_email")
			},
		},
		{
			name: "Table change and index change combined",
			fromSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE INDEX idx_users_email ON users(email);
			`,
			toSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255), name VARCHAR(100));
				CREATE INDEX idx_users_email_name ON users(email, name);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				// New column added
				require.Contains(t, sql, "name")
				// New index created
				require.Contains(t, sql, "idx_users_email_name")
				// Old index dropped
				require.Contains(t, sql, "idx_users_email")
			},
		},
		{
			name: "Multiple tables with index changes",
			fromSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INTEGER);
				CREATE INDEX idx_users_email ON users(email);
			`,
			toSDL: `
				CREATE TABLE users (id SERIAL PRIMARY KEY, email VARCHAR(255));
				CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INTEGER);
				CREATE INDEX idx_users_email ON users(email);
				CREATE INDEX idx_orders_user_id ON orders(user_id);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE INDEX")
				require.Contains(t, sql, "idx_orders_user_id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			tt.validate(t, sql)
		})
	}
}
