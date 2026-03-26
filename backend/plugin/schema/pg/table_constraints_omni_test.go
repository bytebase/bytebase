package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniTableConstraintsSDLDiff(t *testing.T) {
	testCases := []struct {
		name     string
		fromSDL  string
		toSDL    string
		validate func(t *testing.T, sql string)
	}{
		{
			name: "Add primary key constraint",
			fromSDL: `CREATE TABLE users (
				id INTEGER,
				name VARCHAR(255)
			);`,
			toSDL: `CREATE TABLE users (
				id INTEGER,
				name VARCHAR(255),
				CONSTRAINT pk_users PRIMARY KEY (id)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "PRIMARY KEY")
				require.Contains(t, sql, "pk_users")
			},
		},
		{
			name: "Drop foreign key constraint",
			fromSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER,
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
			);`,
			toSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER PRIMARY KEY,
				customer_id INTEGER
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "fk_orders_customer")
				require.Contains(t, sql, "DROP CONSTRAINT")
			},
		},
		{
			name: "Modify check constraint",
			fromSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_price CHECK (price > 0)
			);`,
			toSDL: `CREATE TABLE products (
				id INTEGER PRIMARY KEY,
				price DECIMAL(10,2),
				CONSTRAINT chk_price CHECK (price >= 0)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "chk_price")
			},
		},
		{
			name: "Multiple constraint types in one table",
			fromSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER,
				customer_id INTEGER,
				order_date DATE,
				amount DECIMAL(10,2),
				status VARCHAR(20)
			);`,
			toSDL: `CREATE TABLE customers (
				id INTEGER PRIMARY KEY,
				name VARCHAR(255)
			);
			CREATE TABLE orders (
				id INTEGER,
				customer_id INTEGER,
				order_date DATE,
				amount DECIMAL(10,2),
				status VARCHAR(20),
				CONSTRAINT pk_orders PRIMARY KEY (id),
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
				CONSTRAINT uk_orders_date_customer UNIQUE (order_date, customer_id),
				CONSTRAINT chk_positive_amount CHECK (amount > 0),
				CONSTRAINT chk_valid_status CHECK (status IN ('pending', 'completed', 'cancelled'))
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "PRIMARY KEY")
				require.Contains(t, sql, "FOREIGN KEY")
				require.Contains(t, sql, "UNIQUE")
				require.Contains(t, sql, "chk_positive_amount")
				require.Contains(t, sql, "chk_valid_status")
			},
		},
		{
			name: "Complex foreign key with schema qualification",
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
				CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES public.customers(id)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "fk_orders_customer")
				require.Contains(t, sql, "FOREIGN KEY")
			},
		},
		{
			name: "Composite unique constraint",
			fromSDL: `CREATE TABLE user_sessions (
				user_id INTEGER,
				session_token VARCHAR(255),
				created_at TIMESTAMP
			);`,
			toSDL: `CREATE TABLE user_sessions (
				user_id INTEGER,
				session_token VARCHAR(255),
				created_at TIMESTAMP,
				CONSTRAINT uk_user_sessions UNIQUE (user_id, session_token)
			);`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "uk_user_sessions")
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
