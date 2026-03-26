package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOmniCommentMigration_TableComment(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "add table comment",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			contains: []string{"COMMENT ON TABLE", "users", "User information table"},
		},
		{
			name: "modify table comment",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Updated user table';
`,
			contains: []string{"COMMENT ON TABLE", "Updated user table"},
		},
		{
			name: "remove table comment",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			contains: []string{"COMMENT ON TABLE", "NULL"},
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

func TestOmniCommentMigration_ColumnComment(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "add column comment",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
`,
			contains: []string{"COMMENT ON COLUMN", "Primary key"},
		},
		{
			name: "modify column comment",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'User unique identifier';
`,
			contains: []string{"COMMENT ON COLUMN", "User unique identifier"},
		},
		{
			name: "multiple column comments",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
COMMENT ON COLUMN "public"."users"."name" IS 'User full name';
`,
			contains: []string{"COMMENT ON COLUMN", "Primary key", "User full name"},
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

func TestOmniCommentMigration_ViewComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users;
`, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users;

COMMENT ON VIEW "public"."active_users" IS 'View of active users';
`)
	require.Contains(t, sql, "COMMENT ON VIEW")
	require.Contains(t, sql, "View of active users")
}

func TestOmniCommentMigration_SequenceComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;
`, `
CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;

COMMENT ON SEQUENCE "public"."user_id_seq" IS 'Sequence for user IDs';
`)
	require.Contains(t, sql, "COMMENT ON SEQUENCE")
	require.Contains(t, sql, "Sequence for user IDs")
}

func TestOmniCommentMigration_FunctionComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$;
`, `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_numbers"(integer, integer) IS 'Adds two numbers';
`)
	require.Contains(t, sql, "COMMENT ON FUNCTION")
	require.Contains(t, sql, "Adds two numbers")
}

func TestOmniCommentMigration_SchemaComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE SCHEMA "app";
`, `
CREATE SCHEMA "app";

COMMENT ON SCHEMA "app" IS 'Application schema';
`)
	require.Contains(t, sql, "COMMENT ON SCHEMA")
	require.Contains(t, sql, "Application schema")
}

func TestOmniCommentMigration_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		fromSDL  string
		toSDL    string
		contains []string
	}{
		{
			name: "comment with single quote",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'User''s table';
`,
			contains: []string{"COMMENT ON TABLE", "User"},
		},
		{
			name: "comment with newlines",
			fromSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);
`,
			toSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'This is a
multi-line
comment';
`,
			contains: []string{"COMMENT ON TABLE", "multi-line"},
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

func TestOmniCommentMigration_NoObjectChangeWithCommentChange(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Old comment';
`, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Updated comment';
`)
	require.Contains(t, sql, "COMMENT ON TABLE")
	require.Contains(t, sql, "Updated comment")
	require.NotContains(t, sql, "ALTER TABLE")
}

func TestOmniCommentMigration_CombinedChanges(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Basic user table';
`, `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text,
    "email" text
);

COMMENT ON TABLE "public"."users" IS 'User table with email';
`)
	require.Contains(t, sql, "ALTER TABLE")
	require.Contains(t, sql, "email")
	require.Contains(t, sql, "COMMENT ON TABLE")
	require.Contains(t, sql, "User table with email")
}

func TestOmniCommentMigration_NewFunctionWithComment(t *testing.T) {
	sql := omniSDLMigration(t, "", `
CREATE FUNCTION "public"."calculate_tax"(amount numeric) RETURNS numeric
	LANGUAGE plpgsql
	AS $$
BEGIN
	RETURN amount * 0.1;
END;
$$;

COMMENT ON FUNCTION "public"."calculate_tax"(numeric) IS 'Calculate 10% tax';
`)
	require.Contains(t, sql, "CREATE FUNCTION")
	require.Contains(t, sql, "COMMENT ON FUNCTION")
	require.Contains(t, sql, "calculate_tax")
	require.Contains(t, sql, "Calculate 10% tax")
}

func TestOmniCommentMigration_NewViewWithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"active" boolean
);
`, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"active" boolean
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users WHERE active = true;

COMMENT ON VIEW "public"."active_users" IS 'View of all active users';
`)
	require.Contains(t, sql, "CREATE VIEW")
	require.Contains(t, sql, "COMMENT ON VIEW")
}

func TestOmniCommentMigration_NewTableWithComment(t *testing.T) {
	sql := omniSDLMigration(t, "", `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255)
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`)
	require.Contains(t, sql, "CREATE TABLE")
	require.Contains(t, sql, "COMMENT ON TABLE")
	require.Contains(t, sql, "users")
	require.Contains(t, sql, "User information table")
}

func TestOmniCommentMigration_AddColumnWithComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);
`, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255)
);

COMMENT ON COLUMN "public"."users"."email" IS 'User email address';
`)
	require.Contains(t, sql, "email")
	require.Contains(t, sql, "COMMENT ON COLUMN")
	require.Contains(t, sql, "User email address")
}

func TestOmniCommentMigration_AddMultipleColumnsWithComments(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);
`, `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255),
	"phone" varchar(20),
	"age" integer
);

COMMENT ON COLUMN "public"."users"."email" IS 'User email address';
COMMENT ON COLUMN "public"."users"."phone" IS 'User phone number';
`)
	require.Contains(t, sql, "email")
	require.Contains(t, sql, "phone")
	require.Contains(t, sql, "age")
	require.Contains(t, sql, "COMMENT ON COLUMN")
	require.Contains(t, sql, "User email address")
	require.Contains(t, sql, "User phone number")
}

func TestOmniCommentMigration_IndexComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);
`, `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);

CREATE INDEX "idx_products_name" ON "public"."products" (name);

COMMENT ON INDEX "public"."idx_products_name" IS 'Index for product name lookup';
`)
	require.Contains(t, sql, "CREATE INDEX")
	require.Contains(t, sql, "COMMENT ON INDEX")
	require.Contains(t, sql, "idx_products_name")
	require.Contains(t, sql, "Index for product name lookup")
}

func TestOmniCommentMigration_NewSequenceWithOwner(t *testing.T) {
	// When a sequence has OWNED BY, omni may treat it as part of the table column.
	// Test just that the diff is computed without error.
	sql := omniSDLMigration(t, `
CREATE TABLE users (
	id integer NOT NULL,
	name text
);
`, `
CREATE TABLE users (
	id integer NOT NULL,
	name text
);

CREATE SEQUENCE user_id_seq AS integer START WITH 1 INCREMENT BY 1;

ALTER SEQUENCE user_id_seq OWNED BY users.id;
`)
	// Omni absorbs the owned sequence into the table definition, so no CREATE SEQUENCE is emitted.
	require.NotContains(t, sql, "CREATE SEQUENCE")
}

func TestOmniCommentMigration_NewSequenceWithOwnerAndComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE products (
	id integer NOT NULL,
	name text
);
`, `
CREATE TABLE products (
	id integer NOT NULL,
	name text
);

CREATE SEQUENCE product_id_seq AS bigint START WITH 100 INCREMENT BY 1;

ALTER SEQUENCE product_id_seq OWNED BY products.id;

COMMENT ON SEQUENCE product_id_seq IS 'Product ID sequence';
`)
	// The comment should be generated even if the CREATE SEQUENCE is absorbed
	require.Contains(t, sql, "product_id_seq")
	require.Contains(t, sql, "Product ID sequence")
}
