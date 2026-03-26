package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniSequenceSDLDiff(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		validate func(t *testing.T, sql string)
	}{
		{
			name:    "Create new sequence",
			fromSDL: ``,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE SEQUENCE")
				require.Contains(t, sql, "user_seq")
			},
		},
		{
			name: "Drop sequence",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "DROP SEQUENCE")
				require.Contains(t, sql, "user_seq")
			},
		},
		{
			name: "Modify sequence (alter)",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 10
					INCREMENT BY 2
					NO MINVALUE
					NO MAXVALUE
					CACHE 5;
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "user_seq")
			},
		},
		{
			name: "No changes to sequence",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1
					NO MINVALUE
					NO MAXVALUE
					CACHE 1;
			`,
			validate: func(t *testing.T, sql string) {
				// No sequence-related changes expected
				// The SQL might be empty or might not contain sequence statements
				if sql != "" {
					require.NotContains(t, sql, "CREATE SEQUENCE")
					require.NotContains(t, sql, "DROP SEQUENCE")
				}
			},
		},
		{
			name: "Multiple sequences with different changes",
			fromSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1;

				CREATE SEQUENCE order_seq
					START WITH 100
					INCREMENT BY 1;
			`,
			toSDL: `
				CREATE TABLE users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL
				);

				CREATE SEQUENCE user_seq
					START WITH 1
					INCREMENT BY 1;

				CREATE SEQUENCE product_seq
					START WITH 1000
					INCREMENT BY 10;
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				// order_seq should be dropped
				require.Contains(t, sql, "order_seq")
				// product_seq should be created
				require.Contains(t, sql, "product_seq")
			},
		},
		{
			name: "Sequence with complex options",
			fromSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					amount DECIMAL(10,2)
				);
			`,
			toSDL: `
				CREATE TABLE orders (
					id BIGINT PRIMARY KEY,
					amount DECIMAL(10,2)
				);

				CREATE SEQUENCE order_id_seq
					AS BIGINT
					START WITH 1000000
					INCREMENT BY 1
					MINVALUE 1
					MAXVALUE 9223372036854775807
					CACHE 50
					NO CYCLE;
			`,
			validate: func(t *testing.T, sql string) {
				require.NotEmpty(t, sql)
				require.Contains(t, sql, "CREATE SEQUENCE")
				require.Contains(t, sql, "order_id_seq")
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
