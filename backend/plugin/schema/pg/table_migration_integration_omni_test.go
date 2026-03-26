package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniTableSDLDiffAndMigration(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name:    "Create new table",
			fromSDL: ``,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(255) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			contains: []string{"CREATE TABLE", "users"},
		},
		{
			name: "Drop table",
			fromSDL: `
				CREATE TABLE old_table (
					id INTEGER PRIMARY KEY,
					data TEXT
				);
			`,
			toSDL:    ``,
			contains: []string{"DROP TABLE", "old_table"},
		},
		{
			name:    "Create table with constraints",
			fromSDL: ``,
			toSDL: `
				CREATE TABLE categories (
					id SERIAL PRIMARY KEY
				);
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					price DECIMAL(10,2) CHECK (price > 0),
					category_id INTEGER,
					CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
				);
			`,
			contains: []string{"CREATE TABLE", "products", "FOREIGN KEY", "fk_category"},
		},
		{
			name:    "Create schema-qualified table",
			fromSDL: ``,
			toSDL: `
				CREATE SCHEMA test_schema;
				CREATE TABLE test_schema.items (
					id BIGSERIAL PRIMARY KEY,
					description TEXT
				);
			`,
			contains: []string{"CREATE TABLE", "items"},
		},
		{
			name: "Multiple tables with different operations",
			fromSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_b (id INTEGER PRIMARY KEY);
			`,
			toSDL: `
				CREATE TABLE table_a (id INTEGER PRIMARY KEY);
				CREATE TABLE table_c (id INTEGER PRIMARY KEY);
			`,
			contains: []string{"DROP TABLE", "table_b", "CREATE TABLE", "table_c"},
		},
		{
			name:    "Create table with various data types",
			fromSDL: ``,
			toSDL: `
				CREATE TABLE data_types_test (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					title VARCHAR(100),
					content TEXT,
					count INTEGER,
					amount DECIMAL(15,2),
					is_active BOOLEAN DEFAULT false,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					metadata JSONB
				);
			`,
			contains: []string{"CREATE TABLE", "data_types_test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniTableConstraintChanges(t *testing.T) {
	tests := []struct {
		name        string
		fromSDL     string
		toSDL       string
		contains    []string
		notContains []string
	}{
		{
			name: "Add CHECK constraint",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			contains: []string{"chk_price", "CHECK"},
		},
		{
			name: "Add FOREIGN KEY constraint",
			fromSDL: `
				CREATE TABLE customers (
					id SERIAL PRIMARY KEY
				);
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE customers (
					id SERIAL PRIMARY KEY
				);
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
				);
			`,
			contains: []string{"fk_customer", "FOREIGN KEY"},
		},
		{
			name: "Add UNIQUE constraint",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(255) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(255) NOT NULL,
					CONSTRAINT unique_email UNIQUE (email)
				);
			`,
			contains: []string{"unique_email", "UNIQUE"},
		},
		{
			name: "Drop CHECK constraint",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL,
					CONSTRAINT chk_price CHECK (price > 0)
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			contains: []string{"chk_price", "DROP"},
		},
		{
			name: "Drop FOREIGN KEY constraint",
			fromSDL: `
				CREATE TABLE customers (
					id SERIAL PRIMARY KEY
				);
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
				);
			`,
			toSDL: `
				CREATE TABLE customers (
					id SERIAL PRIMARY KEY
				);
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL
				);
			`,
			contains: []string{"fk_customer", "DROP"},
		},
		{
			name: "Add PRIMARY KEY constraint",
			fromSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id INTEGER NOT NULL,
					name VARCHAR(255) NOT NULL,
					CONSTRAINT pk_users PRIMARY KEY (id)
				);
			`,
			contains: []string{"pk_users", "PRIMARY KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
			for _, s := range tt.notContains {
				require.NotContains(t, sql, s)
			}
		})
	}
}

func TestOmniTableIndexChanges(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "Create standalone indexes",
			fromSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);
			`,
			toSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer_id ON orders (customer_id);
				CREATE UNIQUE INDEX idx_orders_status_unique ON orders (status);
			`,
			contains: []string{"CREATE INDEX", "idx_orders_customer_id", "idx_orders_status_unique"},
		},
		{
			name: "Drop standalone indexes",
			fromSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_customer_id ON orders (customer_id);
				CREATE UNIQUE INDEX idx_orders_status_unique ON orders (status);
			`,
			toSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);
			`,
			contains: []string{"DROP INDEX", "idx_orders_customer_id", "idx_orders_status_unique"},
		},
		{
			name: "Replace indexes (drop old, create new)",
			fromSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_old ON orders (customer_id);
			`,
			toSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					total_amount DECIMAL(12,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending'
				);

				CREATE INDEX idx_orders_new ON orders (customer_id, status);
			`,
			contains: []string{"DROP INDEX", "idx_orders_old", "CREATE INDEX", "idx_orders_new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.contains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniComplexTableWithConstraintsAndIndexes(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE customers (
			id SERIAL PRIMARY KEY
		);
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			customer_id INTEGER NOT NULL,
			order_date DATE DEFAULT CURRENT_DATE,
			total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0),
			status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
		);
	`)
	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "orders")
	require.Contains(t, sql, "fk_customer")
	require.Contains(t, sql, "FOREIGN KEY")
	require.Contains(t, sql, "unique_customer_date")
	require.Contains(t, sql, "UNIQUE")
}

func TestOmniMultipleTablesHandling(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
	`, `
		CREATE TABLE table1 (id INTEGER PRIMARY KEY);
		CREATE TABLE table2 (id SERIAL PRIMARY KEY, name TEXT);
		CREATE TABLE table3 (id UUID PRIMARY KEY, data JSONB);
	`)
	require.Contains(t, sql, "table2")
	require.Contains(t, sql, "table3")
	tableCount := strings.Count(sql, "CREATE TABLE")
	require.Equal(t, 2, tableCount)
}

func TestOmniTableComplexConstraintChanges(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			age INTEGER CHECK (age >= 18),
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			age INTEGER CHECK (age >= 16),
			phone VARCHAR(20) UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			amount DECIMAL(10,2) NOT NULL CHECK (amount > 0),
			status VARCHAR(20) DEFAULT 'pending',
			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT unique_user_amount UNIQUE (user_id, amount)
		);
	`)
	require.Contains(t, sql, "users")
	require.Contains(t, sql, "orders")
}

func TestOmniCreateTableWithMultipleConstraints(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE customers (
			id SERIAL PRIMARY KEY
		);
		CREATE TABLE orders (
			id SERIAL,
			customer_id INTEGER NOT NULL,
			order_date DATE DEFAULT CURRENT_DATE,
			total_amount DECIMAL(12,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT NOW(),
			CONSTRAINT pk_orders PRIMARY KEY (id),
			CONSTRAINT chk_total_amount CHECK (total_amount >= 0),
			CONSTRAINT chk_status CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered')),
			CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			CONSTRAINT unique_customer_date UNIQUE (customer_id, order_date)
		);
	`)
	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "pk_orders")
	require.Contains(t, sql, "chk_total_amount")
	require.Contains(t, sql, "chk_status")
	require.Contains(t, sql, "fk_customer")
	require.Contains(t, sql, "unique_customer_date")
}

func TestOmniCreateTableWithForeignKeyActions(t *testing.T) {
	sql := omniSDLMigration(t, "", `
		CREATE TABLE orders (
			id SERIAL PRIMARY KEY
		);
		CREATE TABLE products (
			id SERIAL PRIMARY KEY
		);
		CREATE TABLE order_items (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL DEFAULT 1,
			unit_price DECIMAL(10,2) NOT NULL,
			CONSTRAINT chk_quantity CHECK (quantity > 0),
			CONSTRAINT chk_unit_price CHECK (unit_price > 0),
			CONSTRAINT fk_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE ON UPDATE CASCADE,
			CONSTRAINT fk_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT ON UPDATE CASCADE,
			CONSTRAINT unique_order_product UNIQUE (order_id, product_id)
		);
	`)
	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "order_items")
	require.Contains(t, sql, "fk_order")
	require.Contains(t, sql, "fk_product")
	require.Contains(t, sql, "ON DELETE CASCADE")
	require.Contains(t, sql, "ON DELETE RESTRICT")
}
