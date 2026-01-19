package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestCommentDiff_TableComment tests table-level comment changes
func TestCommentDiff_TableComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantCommentDiffCount int
		wantAction           schema.MetadataDiffAction
		wantObjectType       schema.CommentObjectType
		wantOldComment       string
		wantNewComment       string
	}{
		{
			name: "add table comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User information table';`,
			wantCommentDiffCount: 1,
			wantAction:           schema.MetadataDiffActionCreate,
			wantObjectType:       schema.CommentObjectTypeTable,
			wantOldComment:       "",
			wantNewComment:       "User information table",
		},
		{
			name: "modify table comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Old comment';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`,
			wantCommentDiffCount: 1,
			wantAction:           schema.MetadataDiffActionAlter,
			wantObjectType:       schema.CommentObjectTypeTable,
			wantOldComment:       "Old comment",
			wantNewComment:       "New comment",
		},
		{
			name: "remove table comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS NULL;`,
			wantCommentDiffCount: 1,
			wantAction:           schema.MetadataDiffActionAlter,
			wantObjectType:       schema.CommentObjectTypeTable,
			wantOldComment:       "User table",
			wantNewComment:       "",
		},
		{
			name: "no comment change",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Same comment';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Same comment';`,
			wantCommentDiffCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantCommentDiffCount > 0 {
				commentDiff := diff.CommentChanges[0]
				require.Equal(t, tt.wantAction, commentDiff.Action, "unexpected action")
				require.Equal(t, tt.wantObjectType, commentDiff.ObjectType, "unexpected object type")
				require.Equal(t, tt.wantOldComment, commentDiff.OldComment, "unexpected old comment")
				require.Equal(t, tt.wantNewComment, commentDiff.NewComment, "unexpected new comment")
				require.Equal(t, "public", commentDiff.SchemaName, "unexpected schema name")
				require.Equal(t, "users", commentDiff.ObjectName, "unexpected object name")
			}
		})
	}
}

// TestCommentDiff_ColumnComment tests column-level comment changes
func TestCommentDiff_ColumnComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantCommentDiffCount int
		wantAction           schema.MetadataDiffAction
		wantColumnName       string
		wantOldComment       string
		wantNewComment       string
	}{
		{
			name: "add column comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User full name';`,
			wantCommentDiffCount: 1,
			wantAction:           schema.MetadataDiffActionCreate,
			wantColumnName:       "name",
			wantOldComment:       "",
			wantNewComment:       "User full name",
		},
		{
			name: "modify column comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'Old name';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'New name';`,
			wantCommentDiffCount: 1,
			wantAction:           schema.MetadataDiffActionAlter,
			wantColumnName:       "name",
			wantOldComment:       "Old name",
			wantNewComment:       "New name",
		},
		{
			name: "multiple column comments",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'Full name';
COMMENT ON COLUMN users.email IS 'Email address';`,
			wantCommentDiffCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantCommentDiffCount == 1 {
				commentDiff := diff.CommentChanges[0]
				require.Equal(t, tt.wantAction, commentDiff.Action, "unexpected action")
				require.Equal(t, schema.CommentObjectTypeColumn, commentDiff.ObjectType, "unexpected object type")
				require.Equal(t, tt.wantColumnName, commentDiff.ColumnName, "unexpected column name")
				require.Equal(t, tt.wantOldComment, commentDiff.OldComment, "unexpected old comment")
				require.Equal(t, tt.wantNewComment, commentDiff.NewComment, "unexpected new comment")
			}
		})
	}
}

// TestCommentDiff_ViewComment tests view-level comment changes
func TestCommentDiff_ViewComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantCommentDiffCount int
		wantNewComment       string
	}{
		{
			name: "add view comment",
			previousSDL: `
CREATE VIEW user_view AS SELECT * FROM users;`,
			currentSDL: `
CREATE VIEW user_view AS SELECT * FROM users;

COMMENT ON VIEW user_view IS 'User view description';`,
			wantCommentDiffCount: 1,
			wantNewComment:       "User view description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantCommentDiffCount > 0 {
				commentDiff := diff.CommentChanges[0]
				require.Equal(t, schema.CommentObjectTypeView, commentDiff.ObjectType, "unexpected object type")
				require.Equal(t, tt.wantNewComment, commentDiff.NewComment, "unexpected new comment")
			}
		})
	}
}

// TestCommentDiff_SequenceComment tests sequence-level comment changes
func TestCommentDiff_SequenceComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantCommentDiffCount int
		wantNewComment       string
	}{
		{
			name: "add sequence comment",
			previousSDL: `
CREATE SEQUENCE user_id_seq;`,
			currentSDL: `
CREATE SEQUENCE user_id_seq;

COMMENT ON SEQUENCE user_id_seq IS 'User ID sequence';`,
			wantCommentDiffCount: 1,
			wantNewComment:       "User ID sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantCommentDiffCount > 0 {
				commentDiff := diff.CommentChanges[0]
				require.Equal(t, schema.CommentObjectTypeSequence, commentDiff.ObjectType, "unexpected object type")
				require.Equal(t, tt.wantNewComment, commentDiff.NewComment, "unexpected new comment")
			}
		})
	}
}

// TestCommentDiff_FunctionComment tests function-level comment changes
func TestCommentDiff_FunctionComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantCommentDiffCount int
		wantNewComment       string
	}{
		{
			name: "add function comment",
			previousSDL: `
CREATE FUNCTION add_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;`,
			currentSDL: `
CREATE FUNCTION add_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION add_numbers(INT, INT) IS 'Adds two numbers';`,
			wantCommentDiffCount: 1,
			wantNewComment:       "Adds two numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantCommentDiffCount > 0 {
				commentDiff := diff.CommentChanges[0]
				require.Equal(t, schema.CommentObjectTypeFunction, commentDiff.ObjectType, "unexpected object type")
				require.Equal(t, tt.wantNewComment, commentDiff.NewComment, "unexpected new comment")
			}
		})
	}
}

// TestCommentDiff_ObjectCreateWithComment tests that creating an object with comment generates CommentDiff
func TestCommentDiff_ObjectCreateWithComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantTableDiffCount   int
		wantCommentDiffCount int
	}{
		{
			name:        "create table with comment",
			previousSDL: ``,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			wantTableDiffCount:   1,
			wantCommentDiffCount: 1, // CommentDiff is generated for the table comment
		},
		{
			name:        "create table with column comment",
			previousSDL: ``,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`,
			wantTableDiffCount:   1,
			wantCommentDiffCount: 1, // CommentDiff is generated for the column comment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantTableDiffCount, len(diff.TableChanges), "unexpected number of table diffs")
			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantTableDiffCount > 0 {
				require.Equal(t, schema.MetadataDiffActionCreate, diff.TableChanges[0].Action, "expected CREATE action")
			}
		})
	}
}

// TestCommentDiff_ObjectDropWithComment tests that dropping an object with comment doesn't generate CommentDiff
func TestCommentDiff_ObjectDropWithComment(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantTableDiffCount   int
		wantCommentDiffCount int
	}{
		{
			name: "drop table with comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			currentSDL:           ``,
			wantTableDiffCount:   1,
			wantCommentDiffCount: 0, // No comment diff because table is dropped
		},
		{
			name: "drop table with column comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`,
			currentSDL:           ``,
			wantTableDiffCount:   1,
			wantCommentDiffCount: 0, // No comment diff because table is dropped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantTableDiffCount, len(diff.TableChanges), "unexpected number of table diffs")
			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")

			if tt.wantTableDiffCount > 0 {
				require.Equal(t, schema.MetadataDiffActionDrop, diff.TableChanges[0].Action, "expected DROP action")
			}
		})
	}
}

// TestCommentDiff_CommentOnlyChange tests that comment-only changes don't trigger object diffs
func TestCommentDiff_CommentOnlyChange(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantTableDiffCount   int
		wantCommentDiffCount int
	}{
		{
			name: "only table comment changes",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Old comment';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`,
			wantTableDiffCount:   0, // No table diff, only comment changed
			wantCommentDiffCount: 1,
		},
		{
			name: "only column comment changes",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'Old name';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'New name';`,
			wantTableDiffCount:   0, // No table diff, only comment changed
			wantCommentDiffCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantTableDiffCount, len(diff.TableChanges), "unexpected number of table diffs")
			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")
		})
	}
}

// TestCommentDiff_TableChangeWithCommentChange tests table changes with comment changes
func TestCommentDiff_TableChangeWithCommentChange(t *testing.T) {
	tests := []struct {
		name                 string
		previousSDL          string
		currentSDL           string
		wantTableDiffCount   int
		wantCommentDiffCount int
	}{
		{
			name: "add column and change comment",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'Old comment';`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`,
			wantTableDiffCount:   1, // Table changed (column added)
			wantCommentDiffCount: 1, // Comment also changed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, tt.wantTableDiffCount, len(diff.TableChanges), "unexpected number of table diffs")
			require.Equal(t, tt.wantCommentDiffCount, len(diff.CommentChanges), "unexpected number of comment diffs")
		})
	}
}

// TestCommentDiff_SpecialCharacters tests comment handling with special characters
func TestCommentDiff_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name           string
		previousSDL    string
		currentSDL     string
		wantNewComment string
	}{
		{
			name: "comment with single quote",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'User''s table';`,
			wantNewComment: "User''s table", // Escaped single quote
		},
		{
			name: "comment with newlines and special chars",
			previousSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);`,
			currentSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'Line 1
Line 2';`,
			wantNewComment: "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, 1, len(diff.CommentChanges), "unexpected number of comment diffs")
			require.Equal(t, tt.wantNewComment, diff.CommentChanges[0].NewComment, "unexpected new comment")
		})
	}
}

// TestCommentDiff_SchemaQualified tests schema-qualified object names
func TestCommentDiff_SchemaQualified(t *testing.T) {
	tests := []struct {
		name           string
		previousSDL    string
		currentSDL     string
		wantSchemaName string
		wantObjectName string
	}{
		{
			name: "schema qualified table",
			previousSDL: `
CREATE TABLE myschema.users (
    id INT PRIMARY KEY
);`,
			currentSDL: `
CREATE TABLE myschema.users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE myschema.users IS 'Users table';`,
			wantSchemaName: "myschema",
			wantObjectName: "users",
		},
		{
			name: "schema qualified column",
			previousSDL: `
CREATE TABLE myschema.users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			currentSDL: `
CREATE TABLE myschema.users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN myschema.users.name IS 'User name';`,
			wantSchemaName: "myschema",
			wantObjectName: "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := GetSDLDiff(tt.currentSDL, tt.previousSDL, nil)
			require.NoError(t, err)
			require.NotNil(t, diff)

			require.Equal(t, 1, len(diff.CommentChanges), "unexpected number of comment diffs")
			require.Equal(t, tt.wantSchemaName, diff.CommentChanges[0].SchemaName, "unexpected schema name")
			require.Equal(t, tt.wantObjectName, diff.CommentChanges[0].ObjectName, "unexpected object name")
		})
	}
}
