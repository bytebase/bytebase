package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniImplicitSchemaCreation_AddTableInNewSchema(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY,
			name VARCHAR(100)
		);
	`, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY,
			name VARCHAR(100)
		);

		CREATE SCHEMA new_schema;

		CREATE TABLE new_schema.t(
			id INT PRIMARY KEY,
			value VARCHAR(50)
		);
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "new_schema")
	require.Contains(t, sql, "CREATE TABLE")
}

func TestOmniImplicitSchemaCreation_MultipleTablesInNewSchema(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);
	`, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);

		CREATE SCHEMA new_schema;

		CREATE TABLE new_schema.users(
			id INT PRIMARY KEY,
			name VARCHAR(100)
		);

		CREATE TABLE new_schema.orders(
			id INT PRIMARY KEY,
			user_id INT REFERENCES new_schema.users(id)
		);
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "new_schema")
	require.Contains(t, sql, "users")
	require.Contains(t, sql, "orders")
}

func TestOmniImplicitSchemaCreation_ExplicitCreateSchema(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY,
			name VARCHAR(100)
		);
	`, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY,
			name VARCHAR(100)
		);

		CREATE SCHEMA new_schema;

		CREATE TABLE new_schema.t(
			id INT PRIMARY KEY,
			value VARCHAR(50)
		);
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "new_schema")
	require.Contains(t, sql, "CREATE TABLE")
}

func TestOmniImplicitSchemaCreation_ExplicitSchemaWithoutObjects(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);
	`, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);

		CREATE SCHEMA new_schema;
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "new_schema")
}

func TestOmniImplicitSchemaCreation_MixedSchemas(t *testing.T) {
	sql := omniSDLMigration(t, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);
	`, `
		CREATE TABLE public.existing_table(
			id INT PRIMARY KEY
		);

		CREATE SCHEMA schema_a;

		CREATE TABLE schema_a.table_a(
			id INT PRIMARY KEY
		);

		CREATE SCHEMA schema_b;

		CREATE TABLE schema_b.table_b(
			id INT PRIMARY KEY
		);
	`)
	require.Contains(t, sql, "CREATE SCHEMA")
	require.Contains(t, sql, "schema_a")
	require.Contains(t, sql, "schema_b")
	require.Contains(t, sql, "table_a")
	require.Contains(t, sql, "table_b")
}
