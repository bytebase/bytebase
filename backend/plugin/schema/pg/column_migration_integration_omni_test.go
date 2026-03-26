package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniColumnSDLDiffAndMigration(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "Add new column",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(320) UNIQUE
				);
			`,
			contains: []string{"ALTER TABLE", "email"},
		},
		{
			name: "Drop column",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					phone VARCHAR(20)
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			contains: []string{"ALTER TABLE", "DROP", "phone"},
		},
		{
			name: "Modify column type",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(15,3) NOT NULL DEFAULT 0.00
				);
			`,
			contains: []string{"ALTER TABLE", "price"},
		},
		{
			name: "Set NOT NULL",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320)
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL
				);
			`,
			contains: []string{"ALTER TABLE", "NOT NULL", "email"},
		},
		{
			name: "Drop NOT NULL",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(320)
				);
			`,
			contains: []string{"ALTER TABLE", "DROP NOT NULL", "email"},
		},
		{
			name: "Add default value",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL DEFAULT 9.99
				);
			`,
			contains: []string{"ALTER TABLE", "DEFAULT", "price"},
		},
		{
			name: "Drop default value",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL DEFAULT 9.99
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					price DECIMAL(10,2) NOT NULL
				);
			`,
			contains: []string{"ALTER TABLE", "DROP DEFAULT", "price"},
		},
		{
			name: "Change column type only",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			contains: []string{"ALTER TABLE", "TYPE", "name"},
		},
		{
			name: "Multiple column changes in same table",
			fromSDL: `
				CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100),
					email VARCHAR(255) NOT NULL,
					salary DECIMAL(8,2)
				);
			`,
			toSDL: `
				CREATE TABLE employees (
					id SERIAL PRIMARY KEY,
					name VARCHAR(200) NOT NULL,
					email VARCHAR(320) NOT NULL,
					salary DECIMAL(10,2) DEFAULT 50000.00
				);
			`,
			contains: []string{"ALTER TABLE", "name", "email", "salary"},
		},
		{
			name: "Add column with complex constraints",
			fromSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE orders (
					id SERIAL PRIMARY KEY,
					customer_id INTEGER NOT NULL,
					status VARCHAR(20) NOT NULL DEFAULT 'pending'
				);
			`,
			contains: []string{"ALTER TABLE", "status"},
		},
		{
			name: "Change column from nullable to not null with default",
			fromSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					category VARCHAR(50)
				);
			`,
			toSDL: `
				CREATE TABLE products (
					id SERIAL PRIMARY KEY,
					category VARCHAR(50) NOT NULL DEFAULT 'general'
				);
			`,
			contains: []string{"ALTER TABLE", "NOT NULL", "DEFAULT", "category"},
		},
		{
			name: "Add multiple columns",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					email VARCHAR(320) UNIQUE,
					created_at TIMESTAMP DEFAULT NOW()
				);
			`,
			contains: []string{"ALTER TABLE", "email", "created_at"},
		},
		{
			name: "Drop multiple columns",
			fromSDL: `
				CREATE TABLE temp_data (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					temp_field1 VARCHAR(100),
					temp_field2 INTEGER,
					description TEXT
				);
			`,
			toSDL: `
				CREATE TABLE temp_data (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					description TEXT
				);
			`,
			contains: []string{"ALTER TABLE", "DROP", "temp_field1", "temp_field2"},
		},
		{
			name: "Mixed operations: add, drop, modify",
			fromSDL: `
				CREATE TABLE inventory (
					id SERIAL PRIMARY KEY,
					product_name VARCHAR(100) NOT NULL,
					old_field VARCHAR(50),
					quantity INTEGER DEFAULT 0
				);
			`,
			toSDL: `
				CREATE TABLE inventory (
					id SERIAL PRIMARY KEY,
					product_name VARCHAR(200) NOT NULL,
					quantity INTEGER DEFAULT 1,
					location VARCHAR(100) NOT NULL DEFAULT 'warehouse'
				);
			`,
			contains: []string{"ALTER TABLE", "old_field", "location", "product_name"},
		},
		{
			name: "Integer type variations",
			fromSDL: `
				CREATE TABLE counters (
					id SERIAL PRIMARY KEY,
					small_count SMALLINT,
					big_count BIGINT
				);
			`,
			toSDL: `
				CREATE TABLE counters (
					id SERIAL PRIMARY KEY,
					small_count INTEGER NOT NULL DEFAULT 0,
					big_count BIGINT NOT NULL
				);
			`,
			contains: []string{"ALTER TABLE", "small_count", "big_count"},
		},
		{
			name: "Text and character types",
			fromSDL: `
				CREATE TABLE documents (
					id SERIAL PRIMARY KEY,
					title CHAR(50),
					content TEXT
				);
			`,
			toSDL: `
				CREATE TABLE documents (
					id SERIAL PRIMARY KEY,
					title VARCHAR(100) NOT NULL,
					content TEXT DEFAULT ''
				);
			`,
			contains: []string{"ALTER TABLE", "title", "content"},
		},
		{
			name: "Boolean column with default",
			fromSDL: `
				CREATE TABLE features (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL
				);
			`,
			toSDL: `
				CREATE TABLE features (
					id SERIAL PRIMARY KEY,
					name VARCHAR(100) NOT NULL,
					enabled BOOLEAN NOT NULL DEFAULT false
				);
			`,
			contains: []string{"ALTER TABLE", "enabled"},
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
