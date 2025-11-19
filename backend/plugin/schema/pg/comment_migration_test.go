package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestCommentMigration_TableComment(t *testing.T) {
	tests := []struct {
		name        string
		currentSDL  string
		previousSDL string
		wantSQL     string
	}{
		{
			name: "add table comment",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			wantSQL: `COMMENT ON TABLE "public"."users" IS 'User information table';
`,
		},
		{
			name: "modify table comment",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Updated user table';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			wantSQL: `COMMENT ON TABLE "public"."users" IS 'Updated user table';
`,
		},
		{
			name: "remove table comment",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			wantSQL: `COMMENT ON TABLE "public"."users" IS NULL;
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)

			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			require.Equal(t, tt.wantSQL, migrationSQL)
		})
	}
}

func TestCommentMigration_ColumnComment(t *testing.T) {
	tests := []struct {
		name        string
		currentSDL  string
		previousSDL string
		wantSQL     string
	}{
		{
			name: "add column comment",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			wantSQL: `COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
`,
		},
		{
			name: "modify column comment",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'User unique identifier';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
`,
			wantSQL: `COMMENT ON COLUMN "public"."users"."id" IS 'User unique identifier';
`,
		},
		{
			name: "multiple column comments",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
COMMENT ON COLUMN "public"."users"."name" IS 'User full name';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			// Note: order may vary depending on map iteration
			wantSQL: `COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';
COMMENT ON COLUMN "public"."users"."name" IS 'User full name';
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)

			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			// For multiple column comments, check that both statements exist (order may vary)
			if tt.name == "multiple column comments" {
				require.Contains(t, migrationSQL, `COMMENT ON COLUMN "public"."users"."id" IS 'Primary key';`)
				require.Contains(t, migrationSQL, `COMMENT ON COLUMN "public"."users"."name" IS 'User full name';`)
			} else {
				require.Equal(t, tt.wantSQL, migrationSQL)
			}
		})
	}
}

func TestCommentMigration_ViewComment(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users;

COMMENT ON VIEW "public"."active_users" IS 'View of active users';
`

	previousSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users;
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	expectedSQL := `COMMENT ON VIEW "public"."active_users" IS 'View of active users';
`
	require.Equal(t, expectedSQL, migrationSQL)
}

func TestCommentMigration_SequenceComment(t *testing.T) {
	currentSDL := `
CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;

COMMENT ON SEQUENCE "public"."user_id_seq" IS 'Sequence for user IDs';
`

	previousSDL := `
CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	expectedSQL := `COMMENT ON SEQUENCE "public"."user_id_seq" IS 'Sequence for user IDs';
`
	require.Equal(t, expectedSQL, migrationSQL)
}

func TestCommentMigration_FunctionComment(t *testing.T) {
	currentSDL := `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_numbers"(integer, integer) IS 'Adds two numbers';
`

	previousSDL := `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$;
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	// Function signature includes parameter names
	expectedSQL := `COMMENT ON FUNCTION "public".add_numbers(a integer, b integer) IS 'Adds two numbers';
`
	require.Equal(t, expectedSQL, migrationSQL)
}

func TestCommentMigration_SchemaComment(t *testing.T) {
	currentSDL := `
CREATE SCHEMA "app";

COMMENT ON SCHEMA "app" IS 'Application schema';
`

	previousSDL := `
CREATE SCHEMA "app";
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	expectedSQL := `COMMENT ON SCHEMA "app" IS 'Application schema';
`
	require.Equal(t, expectedSQL, migrationSQL)
}

func TestCommentMigration_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name        string
		currentSDL  string
		previousSDL string
		wantSQL     string
	}{
		{
			name: "comment with single quote",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'User''s table';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);
`,
			// Single quotes are already escaped in SDL, and writeCommentOnTable escapes again
			wantSQL: `COMMENT ON TABLE "public"."users" IS 'User''''s table';
`,
		},
		{
			name: "comment with newlines",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'This is a
multi-line
comment';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL
);
`,
			wantSQL: `COMMENT ON TABLE "public"."users" IS 'This is a
multi-line
comment';
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)

			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			require.Equal(t, tt.wantSQL, migrationSQL)
		})
	}
}

func TestCommentMigration_NoObjectChangeWithCommentChange(t *testing.T) {
	// Test that when only comment changes, no table ALTER is generated
	currentSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Updated comment';
`

	previousSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Old comment';
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	// Should have no table changes
	require.Len(t, diff.TableChanges, 0)

	// Should have comment change
	require.Len(t, diff.CommentChanges, 1)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	expectedSQL := `COMMENT ON TABLE "public"."users" IS 'Updated comment';
`
	require.Equal(t, expectedSQL, migrationSQL)
}

func TestCommentMigration_CombinedChanges(t *testing.T) {
	// Test that when both table and comment change, we get both statements
	currentSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text,
    "email" text
);

COMMENT ON TABLE "public"."users" IS 'User table with email';
`

	previousSDL := `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Basic user table';
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	// Should have table change
	require.Len(t, diff.TableChanges, 1)
	require.Equal(t, "ALTER", string(diff.TableChanges[0].Action))

	// Should have comment change
	require.Len(t, diff.CommentChanges, 1)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	// Should contain both ALTER TABLE and COMMENT ON
	require.Contains(t, migrationSQL, "ALTER TABLE")
	require.Contains(t, migrationSQL, "COMMENT ON TABLE")
	require.Contains(t, migrationSQL, "email")
	require.Contains(t, migrationSQL, "User table with email")
}

func TestFunctionMigration_NewFunctionWithComment(t *testing.T) {
	currentSDL := `
CREATE FUNCTION "public"."calculate_tax"(amount numeric) RETURNS numeric
	LANGUAGE plpgsql
	AS $$
BEGIN
	RETURN amount * 0.1;
END;
$$;

COMMENT ON FUNCTION "public"."calculate_tax"(numeric) IS 'Calculate 10% tax';
`

	previousSDL := ``

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE FUNCTION and COMMENT ON FUNCTION
	require.Contains(t, migrationSQL, "CREATE FUNCTION")
	require.Contains(t, migrationSQL, "COMMENT ON FUNCTION")
	require.Contains(t, migrationSQL, "calculate_tax")
	require.Contains(t, migrationSQL, "Calculate 10% tax")
}

func TestViewMigration_NewViewWithComment(t *testing.T) {
	tests := []struct {
		name        string
		currentSDL  string
		previousSDL string
	}{
		{
			name: "new view with comment - table exists in both",
			currentSDL: `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"active" boolean
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users WHERE active = true;

COMMENT ON VIEW "public"."active_users" IS 'View of all active users';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"active" boolean
);
`,
		},
		{
			name: "new view with comment - realistic scenario with JOIN",
			currentSDL: `
CREATE TABLE "public"."test_table" (
	"id" integer NOT NULL,
	"name" text,
	"created_at" timestamp,
	"is_active" boolean,
	"description" text
);

CREATE TABLE "public"."test_r_table" (
	"id" integer NOT NULL,
	"r_name" text,
	"r_value" text
);

CREATE VIEW "public"."test_view" AS
SELECT
	t.id,
	t.name,
	t.created_at,
	t.is_active,
	t.description,
	r.r_name,
	r.r_value
FROM
	"public"."test_table" t
JOIN
	"public"."test_r_table" r ON t.id = r.id;

COMMENT ON VIEW "public"."test_view" IS 'A view that combines test_table and test_r_table for easier';
`,
			previousSDL: `
CREATE TABLE "public"."test_table" (
	"id" integer NOT NULL,
	"name" text,
	"created_at" timestamp,
	"is_active" boolean,
	"description" text
);

CREATE TABLE "public"."test_r_table" (
	"id" integer NOT NULL,
	"r_name" text,
	"r_value" text
);
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)

			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration SQL:\n%s", migrationSQL)

			// Should contain both CREATE VIEW and COMMENT ON VIEW
			require.Contains(t, migrationSQL, "CREATE VIEW", "Migration should contain CREATE VIEW")
			require.Contains(t, migrationSQL, "COMMENT ON VIEW", "Migration should contain COMMENT ON VIEW")
		})
	}
}

func TestViewMigration_NewViewWithComment_DriftScenario(t *testing.T) {
	// This test simulates the drift scenario where schemas are provided
	// This matches what happens in real execution (schema_update_sdl_executor.go)

	previousSDL := `
CREATE TABLE "public"."test_table" (
	"id" integer NOT NULL,
	"name" text
);

CREATE TABLE "public"."test_r_table" (
	"id" integer NOT NULL,
	"r_name" text,
	"r_value" text
);
`

	currentSDL := `
CREATE TABLE "public"."test_table" (
	"id" integer NOT NULL,
	"name" text
);

CREATE TABLE "public"."test_r_table" (
	"id" integer NOT NULL,
	"r_name" text,
	"r_value" text
);

CREATE VIEW "public"."test_view" AS
SELECT
	t.id,
	t.name,
	r.r_name,
	r.r_value
FROM
	"public"."test_table" t
JOIN
	"public"."test_r_table" r ON t.id = r.id;

COMMENT ON VIEW "public"."test_view" IS 'A view that combines test_table and test_r_table';
`

	// Create mock schemas with view metadata
	// Previous schema: no views
	previousSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	// Current schema: includes view with comment
	currentSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Views: []*storepb.ViewMetadata{
						{
							Name: "test_view",
							Definition: `SELECT
	t.id,
	t.name,
	r.r_name,
	r.r_value
FROM
	"public"."test_table" t
JOIN
	"public"."test_r_table" r ON t.id = r.id;`,
							Comment: "A view that combines test_table and test_r_table",
						},
					},
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE VIEW and COMMENT ON VIEW
	require.Contains(t, migrationSQL, "CREATE VIEW", "Migration should contain CREATE VIEW")
	require.Contains(t, migrationSQL, "COMMENT ON VIEW", "Migration should contain COMMENT ON VIEW")
	require.Contains(t, migrationSQL, "test_view", "Migration should reference test_view")
	require.Contains(t, migrationSQL, "A view that combines test_table and test_r_table", "Migration should contain the comment text")
}

func TestIndexMigration_NewIndexWithComment(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);

CREATE INDEX "idx_products_name" ON "public"."products" (name);

COMMENT ON INDEX "public"."idx_products_name" IS 'Index for product name lookup';
`

	previousSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE INDEX and COMMENT ON INDEX
	require.Contains(t, migrationSQL, "CREATE INDEX")
	require.Contains(t, migrationSQL, "COMMENT ON INDEX")
	require.Contains(t, migrationSQL, "idx_products_name")
	require.Contains(t, migrationSQL, "Index for product name lookup")
}

func TestTableMigration_NewTableWithComment(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255)
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`
	previousSDL := ``

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE TABLE and COMMENT ON TABLE
	require.Contains(t, migrationSQL, "CREATE TABLE")
	require.Contains(t, migrationSQL, "COMMENT ON TABLE")
	require.Contains(t, migrationSQL, "users")
	require.Contains(t, migrationSQL, "User information table")
}

func TestColumnMigration_AddColumnWithComment(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255)
);

COMMENT ON COLUMN "public"."users"."email" IS 'User email address';
`
	previousSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both ADD COLUMN and COMMENT ON COLUMN
	require.Contains(t, migrationSQL, "ADD COLUMN")
	require.Contains(t, migrationSQL, "COMMENT ON COLUMN")
	require.Contains(t, migrationSQL, "email")
	require.Contains(t, migrationSQL, "User email address")
}

func TestColumnMigration_AddMultipleColumnsWithComments(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255),
	"phone" varchar(20),
	"age" integer
);

COMMENT ON COLUMN "public"."users"."email" IS 'User email address';
COMMENT ON COLUMN "public"."users"."phone" IS 'User phone number';
`
	previousSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain ADD COLUMN for all new columns
	require.Contains(t, migrationSQL, "ADD COLUMN")
	require.Contains(t, migrationSQL, "email")
	require.Contains(t, migrationSQL, "phone")
	require.Contains(t, migrationSQL, "age")

	// Should contain COMMENT ON COLUMN for columns with comments
	require.Contains(t, migrationSQL, "COMMENT ON COLUMN")
	require.Contains(t, migrationSQL, "User email address")
	require.Contains(t, migrationSQL, "User phone number")
}

func TestColumnMigration_AddColumnAndModifyExistingComment(t *testing.T) {
	currentSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text,
	"email" varchar(255)
);

COMMENT ON COLUMN "public"."users"."name" IS 'Updated: User full name';
COMMENT ON COLUMN "public"."users"."email" IS 'User email address';
`
	previousSDL := `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);

COMMENT ON COLUMN "public"."users"."name" IS 'User name';
`

	diff, err := GetSDLDiff(currentSDL, previousSDL, nil, nil)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain ADD COLUMN for the new column
	require.Contains(t, migrationSQL, "ADD COLUMN")
	require.Contains(t, migrationSQL, "email")

	// Should contain COMMENT ON COLUMN for both new and modified comments
	require.Contains(t, migrationSQL, "COMMENT ON COLUMN")
	require.Contains(t, migrationSQL, "Updated: User full name")
	require.Contains(t, migrationSQL, "User email address")
}

func TestSequenceMigration_NewSequenceWithOwner(t *testing.T) {
	tests := []struct {
		name        string
		currentSDL  string
		previousSDL string
		wantSQL     string
	}{
		{
			name: "add new sequence with owner",
			currentSDL: `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);

CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;

ALTER SEQUENCE "public"."user_id_seq" OWNED BY "public"."users"."id";
`,
			previousSDL: `
CREATE TABLE "public"."users" (
	"id" integer NOT NULL,
	"name" text
);
`,
			wantSQL: `CREATE SEQUENCE "public"."user_id_seq" AS integer START WITH 1 INCREMENT BY 1;

ALTER SEQUENCE "public"."user_id_seq" OWNED BY "public"."users"."id";

`,
		},
		{
			name: "add new sequence with owner and comment",
			currentSDL: `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text
);

CREATE SEQUENCE "public"."product_id_seq" AS bigint START WITH 100 INCREMENT BY 1;

ALTER SEQUENCE "public"."product_id_seq" OWNED BY "public"."products"."id";

COMMENT ON SEQUENCE "public"."product_id_seq" IS 'Product ID sequence';
`,
			previousSDL: `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text
);
`,
			wantSQL: `CREATE SEQUENCE "public"."product_id_seq" AS bigint START WITH 100 INCREMENT BY 1;

ALTER SEQUENCE "public"."product_id_seq" OWNED BY "public"."products"."id";

COMMENT ON SEQUENCE "public"."product_id_seq" IS 'Product ID sequence';
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil, nil)
			require.NoError(t, err)

			migrationSQL, err := generateMigration(diff)
			require.NoError(t, err)

			t.Logf("Generated migration SQL:\n%s", migrationSQL)
			t.Logf("Expected SQL:\n%s", tt.wantSQL)

			require.Equal(t, tt.wantSQL, migrationSQL)
		})
	}
}

func TestIndexMigration_NewIndexWithComment_DriftScenario(t *testing.T) {
	// This test simulates the drift scenario where an index with comment
	// is created from database metadata (not in previous SDL)

	previousSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);
`

	currentSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text,
	"price" numeric
);

CREATE INDEX "idx_products_name" ON "public"."products" (name);

COMMENT ON INDEX "public"."idx_products_name" IS 'Index for product name lookup';
`

	// Previous schema: table with columns but no indexes
	previousSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "products",
							Columns: []*storepb.ColumnMetadata{
								{
									Name:     "id",
									Type:     "integer",
									Nullable: false,
								},
								{
									Name: "name",
									Type: "text",
								},
								{
									Name: "price",
									Type: "numeric",
								},
							},
						},
					},
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	// Current schema: includes index with comment
	currentSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "products",
							Columns: []*storepb.ColumnMetadata{
								{
									Name:     "id",
									Type:     "integer",
									Nullable: false,
								},
								{
									Name: "name",
									Type: "text",
								},
								{
									Name: "price",
									Type: "numeric",
								},
							},
							Indexes: []*storepb.IndexMetadata{
								{
									Name:        "idx_products_name",
									Expressions: []string{"name"},
									Unique:      false,
									Comment:     "Index for product name lookup",
								},
							},
						},
					},
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE INDEX and COMMENT ON INDEX
	require.Contains(t, migrationSQL, "CREATE INDEX", "Migration should contain CREATE INDEX")
	require.Contains(t, migrationSQL, "COMMENT ON INDEX", "Migration should contain COMMENT ON INDEX")
	require.Contains(t, migrationSQL, "idx_products_name", "Migration should reference idx_products_name")
	require.Contains(t, migrationSQL, "Index for product name lookup", "Migration should contain the comment text")
}

func TestSequenceMigration_NewSequenceWithComment_DriftScenario(t *testing.T) {
	// This test simulates the drift scenario where a sequence with comment
	// is created from database metadata (not in previous SDL)

	previousSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text
);
`

	currentSDL := `
CREATE TABLE "public"."products" (
	"id" integer NOT NULL,
	"name" text
);

CREATE SEQUENCE "public"."product_id_seq" AS bigint START WITH 100 INCREMENT BY 1;

COMMENT ON SEQUENCE "public"."product_id_seq" IS 'Product ID sequence';
`

	// Previous schema: table but no sequences
	previousSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "products",
							Columns: []*storepb.ColumnMetadata{
								{
									Name:     "id",
									Type:     "integer",
									Nullable: false,
								},
								{
									Name: "name",
									Type: "text",
								},
							},
						},
					},
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	// Current schema: table and sequence with comment
	currentSchema := model.NewDatabaseMetadata(
		&storepb.DatabaseSchemaMetadata{
			Name: "testdb",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name: "products",
							Columns: []*storepb.ColumnMetadata{
								{
									Name:     "id",
									Type:     "integer",
									Nullable: false,
								},
								{
									Name: "name",
									Type: "text",
								},
							},
						},
					},
					Sequences: []*storepb.SequenceMetadata{
						{
							Name:      "product_id_seq",
							DataType:  "bigint",
							Start:     "100",
							Increment: "1",
							Comment:   "Product ID sequence",
						},
					},
				},
			},
		},
		[]byte{},
		&storepb.DatabaseConfig{},
		storepb.Engine_POSTGRES,
		false,
	)

	diff, err := GetSDLDiff(currentSDL, previousSDL, currentSchema, previousSchema)
	require.NoError(t, err)

	migrationSQL, err := generateMigration(diff)
	require.NoError(t, err)

	t.Logf("Generated migration SQL:\n%s", migrationSQL)

	// Should contain both CREATE SEQUENCE and COMMENT ON SEQUENCE
	require.Contains(t, migrationSQL, "CREATE SEQUENCE", "Migration should contain CREATE SEQUENCE")
	require.Contains(t, migrationSQL, "COMMENT ON SEQUENCE", "Migration should contain COMMENT ON SEQUENCE")
	require.Contains(t, migrationSQL, "product_id_seq", "Migration should reference product_id_seq")
	require.Contains(t, migrationSQL, "Product ID sequence", "Migration should contain the comment text")
}
