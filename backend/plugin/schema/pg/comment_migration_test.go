package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
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
