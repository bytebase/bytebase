package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniColumnSDLDiff(t *testing.T) {
	testCases := []struct {
		name     string
		fromSDL  string
		toSDL    string
		validate func(t *testing.T, sql string)
	}{
		{
			name: "No column changes - identical tables",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			validate: func(t *testing.T, sql string) {
				require.Empty(t, sql, "Should produce no migration for identical tables")
			},
		},
		{
			name: "Add new column",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(100)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "ALTER TABLE")
				require.Contains(t, sql, "ADD COLUMN")
				require.Contains(t, sql, "email")
			},
		},
		{
			name: "Drop column",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(100)
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "ALTER TABLE")
				require.Contains(t, sql, "DROP COLUMN")
				require.Contains(t, sql, "email")
			},
		},
		{
			name: "Alter column type",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "ALTER TABLE")
				require.Contains(t, sql, "name")
				require.Contains(t, sql, "TYPE")
			},
		},
		{
			name: "Multiple column changes",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				phone VARCHAR(20),
				updated_at TIMESTAMP
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				email VARCHAR(200),
				created_at TIMESTAMP
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				// Added columns
				require.Contains(t, sql, "email")
				require.Contains(t, sql, "created_at")
				// Dropped columns
				require.Contains(t, sql, "phone")
				require.Contains(t, sql, "updated_at")
				// Altered column type
				require.Contains(t, sql, "name")
			},
		},
		{
			name: "Column constraint changes - drop NOT NULL",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "ALTER TABLE")
				require.Contains(t, sql, "name")
			},
		},
		{
			name: "Column default value changes",
			fromSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50) DEFAULT 'active'
			);`,
			toSDL: `CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				status VARCHAR(50) DEFAULT 'inactive'
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "ALTER TABLE")
				require.Contains(t, sql, "status")
				require.Contains(t, sql, "inactive")
			},
		},
		{
			name: "Add foreign key constraint via column test",
			fromSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER
			);`,
			toSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER,
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "fk_orders_customer")
				require.Contains(t, sql, "FOREIGN KEY")
			},
		},
		{
			name: "Add check constraint via column test",
			fromSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2)
			);`,
			toSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_positive_price CHECK (price > 0)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "chk_positive_price")
			},
		},
		{
			name: "Add unique constraint via column test",
			fromSDL: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				email VARCHAR(255)
			);`,
			toSDL: `CREATE TABLE users (
				id INTEGER PRIMARY KEY,
				email VARCHAR(255),
				CONSTRAINT uk_users_email UNIQUE (email)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "uk_users_email")
				require.Contains(t, sql, "UNIQUE")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tc.fromSDL, tc.toSDL)
			tc.validate(t, sql)
		})
	}
}
