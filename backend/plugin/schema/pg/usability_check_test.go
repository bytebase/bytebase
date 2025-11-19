package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestUsabilityCheck_TableScenarios tests all usability check scenarios for tables
func TestUsabilityCheck_TableScenarios(t *testing.T) {
	tests := []struct {
		name              string
		currentSDL        string
		previousSDL       string
		currentSchema     *model.DatabaseMetadata
		expectTableDiff   bool
		expectCommentDiff bool
		description       string
	}{
		{
			name: "scenario_1_same_structure_same_comment_should_skip_both",
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
    "name" text NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'User table description';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "User information table",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   false,
			expectCommentDiff: false,
			description:       "Structure matches database, comment matches database → skip both diffs",
		},
		{
			name: "scenario_2_same_structure_different_comment_should_skip_structure_only",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Updated user table description';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text NOT NULL
);

COMMENT ON TABLE "public"."users" IS 'Old user table description';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "User information table",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   false,
			expectCommentDiff: true,
			description:       "Structure matches database → skip structure diff, comment differs → generate comment diff",
		},
		{
			name: "scenario_3_different_structure_same_comment_should_skip_comment_only",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text,
    "email" varchar(255)
);

COMMENT ON TABLE "public"."users" IS 'User information table';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Old user table';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "User information table",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   true,
			expectCommentDiff: false,
			description:       "Structure differs → generate structure diff, comment matches database → skip comment diff",
		},
		{
			name: "scenario_4_different_structure_different_comment_should_generate_both",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text,
    "email" varchar(255)
);

COMMENT ON TABLE "public"."users" IS 'Updated user information';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Old user table';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "User information table",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   true,
			expectCommentDiff: true,
			description:       "Structure differs → generate structure diff, comment differs → generate comment diff",
		},
		{
			name: "add_comment_to_existing_table",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

COMMENT ON TABLE "public"."users" IS 'Newly added comment';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   false,
			expectCommentDiff: true,
			description:       "Adding comment to table without comment in database → skip structure, generate comment diff",
		},
		{
			name: "remove_comment_from_table",
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

COMMENT ON TABLE "public"."users" IS 'To be removed';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name:    "users",
									Comment: "User table",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectTableDiff:   false,
			expectCommentDiff: true,
			description:       "Removing comment while structure matches database → skip structure, generate comment diff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)

			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check table diff
			hasTableDiff := len(diff.TableChanges) > 0
			require.Equal(t, tt.expectTableDiff, hasTableDiff,
				"Expected table diff=%v but got=%v. TableChanges count: %d",
				tt.expectTableDiff, hasTableDiff, len(diff.TableChanges))

			// Check comment diff
			hasCommentDiff := false
			for _, commentDiff := range diff.CommentChanges {
				if commentDiff.ObjectType == schema.CommentObjectTypeTable &&
					commentDiff.ObjectName == "users" {
					hasCommentDiff = true
					break
				}
			}
			require.Equal(t, tt.expectCommentDiff, hasCommentDiff,
				"Expected comment diff=%v but got=%v. CommentChanges count: %d",
				tt.expectCommentDiff, hasCommentDiff, len(diff.CommentChanges))

			if t.Failed() {
				t.Log("Full diff result:")
				t.Logf("  TableChanges: %d", len(diff.TableChanges))
				for i, tc := range diff.TableChanges {
					t.Logf("    [%d] %s.%s (Action: %s)", i, tc.SchemaName, tc.TableName, tc.Action)
				}
				t.Logf("  CommentChanges: %d", len(diff.CommentChanges))
				for i, cc := range diff.CommentChanges {
					t.Logf("    [%d] %s.%s (Type: %s, Action: %s)",
						i, cc.SchemaName, cc.ObjectName, cc.ObjectType, cc.Action)
				}
			}
		})
	}
}

// TestUsabilityCheck_ViewScenarios tests usability check for views
func TestUsabilityCheck_ViewScenarios(t *testing.T) {
	tests := []struct {
		name              string
		currentSDL        string
		previousSDL       string
		currentSchema     *model.DatabaseMetadata
		expectViewDiff    bool
		expectCommentDiff bool
		description       string
	}{
		{
			name: "view_same_definition_same_comment_skip_both",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users WHERE id > 0;

COMMENT ON VIEW "public"."active_users" IS 'View of active users';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users WHERE id < 0;

COMMENT ON VIEW "public"."active_users" IS 'View of all users';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
							Views: []*storepb.ViewMetadata{
								{
									Name:       "active_users",
									Definition: "SELECT * FROM users WHERE id > 0",
									Comment:    "View of active users",
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectViewDiff:    false,
			expectCommentDiff: false,
			description:       "View definition and comment match database → skip both",
		},
		{
			name: "view_same_definition_different_comment_skip_view_only",
			currentSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users WHERE id > 0;

COMMENT ON VIEW "public"."active_users" IS 'Updated view comment';
`,
			previousSDL: `
CREATE TABLE "public"."users" (
    "id" integer NOT NULL,
    "name" text
);

CREATE VIEW "public"."active_users" AS SELECT * FROM users;

COMMENT ON VIEW "public"."active_users" IS 'Old view comment';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Tables: []*storepb.TableMetadata{
								{
									Name: "users",
									Columns: []*storepb.ColumnMetadata{
										{Name: "id", Type: "integer", Nullable: false},
										{Name: "name", Type: "text", Nullable: true},
									},
								},
							},
							Views: []*storepb.ViewMetadata{
								{
									Name:       "active_users",
									Definition: "SELECT * FROM users WHERE id > 0",
									Comment:    "View of active users",
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectViewDiff:    false,
			expectCommentDiff: true,
			description:       "View definition matches database, comment differs → skip view diff, generate comment diff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)

			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check view diff
			hasViewDiff := len(diff.ViewChanges) > 0
			require.Equal(t, tt.expectViewDiff, hasViewDiff,
				"Expected view diff=%v but got=%v. ViewChanges count: %d",
				tt.expectViewDiff, hasViewDiff, len(diff.ViewChanges))

			// Check comment diff
			hasCommentDiff := false
			for _, commentDiff := range diff.CommentChanges {
				if commentDiff.ObjectType == schema.CommentObjectTypeView &&
					commentDiff.ObjectName == "active_users" {
					hasCommentDiff = true
					break
				}
			}
			require.Equal(t, tt.expectCommentDiff, hasCommentDiff,
				"Expected comment diff=%v but got=%v. CommentChanges count: %d",
				tt.expectCommentDiff, hasCommentDiff, len(diff.CommentChanges))
		})
	}
}

// TestUsabilityCheck_FunctionScenarios tests usability check for functions
func TestUsabilityCheck_FunctionScenarios(t *testing.T) {
	tests := []struct {
		name               string
		currentSDL         string
		previousSDL        string
		currentSchema      *model.DatabaseMetadata
		expectFunctionDiff bool
		expectCommentDiff  bool
		description        string
	}{
		{
			name: "function_same_body_same_comment_skip_both",
			currentSDL: `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$;

COMMENT ON FUNCTION "public"."add_numbers"(a integer, b integer) IS 'Adds two numbers';
`,
			previousSDL: `
CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a - b;
END;
$$;

COMMENT ON FUNCTION "public"."add_numbers"(a integer, b integer) IS 'Old comment';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Functions: []*storepb.FunctionMetadata{
								{
									Name:      "add_numbers",
									Signature: `add_numbers(a integer, b integer)`,
									Definition: `CREATE FUNCTION "public"."add_numbers"(a integer, b integer) RETURNS integer
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN a + b;
END;
$$`,
									Comment: "Adds two numbers",
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectFunctionDiff: false,
			expectCommentDiff:  false,
			description:        "Function body and comment match database → skip both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)

			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check function diff
			hasFunctionDiff := len(diff.FunctionChanges) > 0
			require.Equal(t, tt.expectFunctionDiff, hasFunctionDiff,
				"Expected function diff=%v but got=%v. FunctionChanges count: %d",
				tt.expectFunctionDiff, hasFunctionDiff, len(diff.FunctionChanges))

			// Check comment diff
			hasCommentDiff := false
			for _, commentDiff := range diff.CommentChanges {
				if commentDiff.ObjectType == schema.CommentObjectTypeFunction &&
					commentDiff.ObjectName == "add_numbers" {
					hasCommentDiff = true
					break
				}
			}
			require.Equal(t, tt.expectCommentDiff, hasCommentDiff,
				"Expected comment diff=%v but got=%v",
				tt.expectCommentDiff, hasCommentDiff)
		})
	}
}

// TestUsabilityCheck_SequenceScenarios tests usability check for sequences
func TestUsabilityCheck_SequenceScenarios(t *testing.T) {
	tests := []struct {
		name               string
		currentSDL         string
		previousSDL        string
		currentSchema      *model.DatabaseMetadata
		expectSequenceDiff bool
		expectCommentDiff  bool
		description        string
	}{
		{
			name: "sequence_same_config_same_comment_skip_both",
			currentSDL: `
CREATE SEQUENCE "public"."user_id_seq" START WITH 1 INCREMENT BY 1 NO CYCLE;

COMMENT ON SEQUENCE "public"."user_id_seq" IS 'User ID sequence';
`,
			previousSDL: `
CREATE SEQUENCE "public"."user_id_seq" START WITH 100 INCREMENT BY 2 NO CYCLE;

COMMENT ON SEQUENCE "public"."user_id_seq" IS 'Old sequence comment';
`,
			currentSchema: model.NewDatabaseMetadata(
				&storepb.DatabaseSchemaMetadata{
					Name: "test_db",
					Schemas: []*storepb.SchemaMetadata{
						{
							Name: "public",
							Sequences: []*storepb.SequenceMetadata{
								{
									Name:      "user_id_seq",
									Start:     "1",
									Increment: "1",
									Comment:   "User ID sequence",
								},
							},
						},
					},
				},
				nil, nil, storepb.Engine_POSTGRES, false,
			),
			expectSequenceDiff: false,
			expectCommentDiff:  false,
			description:        "Sequence config and comment match database → skip both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)

			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, tt.currentSchema, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			// Check sequence diff
			hasSequenceDiff := len(diff.SequenceChanges) > 0
			require.Equal(t, tt.expectSequenceDiff, hasSequenceDiff,
				"Expected sequence diff=%v but got=%v",
				tt.expectSequenceDiff, hasSequenceDiff)

			// Check comment diff
			hasCommentDiff := false
			for _, commentDiff := range diff.CommentChanges {
				if commentDiff.ObjectType == schema.CommentObjectTypeSequence &&
					commentDiff.ObjectName == "user_id_seq" {
					hasCommentDiff = true
					break
				}
			}
			require.Equal(t, tt.expectCommentDiff, hasCommentDiff,
				"Expected comment diff=%v but got=%v",
				tt.expectCommentDiff, hasCommentDiff)
		})
	}
}
