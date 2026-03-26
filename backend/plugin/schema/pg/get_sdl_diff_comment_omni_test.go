package pg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/pg/catalog"
)

func TestOmniCommentDiff_TableComment(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name: "add table comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User information table';`,
			wantContains: []string{"COMMENT ON TABLE", "users", "User information table"},
		},
		{
			name: "modify table comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Old comment';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`,
			wantContains: []string{"COMMENT ON TABLE", "New comment"},
		},
		{
			name: "remove table comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS NULL;`,
			wantContains: []string{"COMMENT ON TABLE", "IS NULL"},
		},
		{
			name: "no comment change",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Same comment';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Same comment';`,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)

			if tt.wantEmpty {
				require.Empty(t, sql)
				return
			}
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_ColumnComment(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
	}{
		{
			name: "add column comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User full name';`,
			wantContains: []string{"COMMENT ON COLUMN", "name", "User full name"},
		},
		{
			name: "modify column comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'Old name';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'New name';`,
			wantContains: []string{"COMMENT ON COLUMN", "New name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_MultipleColumnComments(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`, `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'Full name';
COMMENT ON COLUMN users.email IS 'Email address';`)

	require.Contains(t, sql, "COMMENT ON COLUMN")
	require.Contains(t, sql, "Full name")
	require.Contains(t, sql, "Email address")

	// Also verify via diff counts.
	from, err := catalog.LoadSDL(strings.TrimSpace(`
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`))
	require.NoError(t, err)
	to, err := catalog.LoadSDL(strings.TrimSpace(`
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT,
    email TEXT
);

COMMENT ON COLUMN users.name IS 'Full name';
COMMENT ON COLUMN users.email IS 'Email address';`))
	require.NoError(t, err)
	diff := catalog.Diff(from, to)
	require.GreaterOrEqual(t, len(diff.Comments), 2, "expected at least 2 comment changes")
}

func TestOmniCommentDiff_ViewComment(t *testing.T) {
	sql := omniSDLMigration(t,
		`CREATE VIEW user_view AS SELECT 1;`,
		`CREATE VIEW user_view AS SELECT 1;

COMMENT ON VIEW user_view IS 'User view description';`)

	require.Contains(t, sql, "COMMENT ON VIEW")
	require.Contains(t, sql, "user_view")
	require.Contains(t, sql, "User view description")
}

func TestOmniCommentDiff_SequenceComment(t *testing.T) {
	sql := omniSDLMigration(t,
		`CREATE SEQUENCE user_id_seq;`,
		`CREATE SEQUENCE user_id_seq;

COMMENT ON SEQUENCE user_id_seq IS 'User ID sequence';`)

	require.Contains(t, sql, "COMMENT ON SEQUENCE")
	require.Contains(t, sql, "User ID sequence")
}

func TestOmniCommentDiff_FunctionComment(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE FUNCTION add_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;`, `
CREATE FUNCTION add_numbers(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION add_numbers(INT, INT) IS 'Adds two numbers';`)

	require.Contains(t, sql, "COMMENT ON FUNCTION")
	require.Contains(t, sql, "Adds two numbers")
}

func TestOmniCommentDiff_ObjectCreateWithComment(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
	}{
		{
			name:    "create table with comment",
			fromSDL: ``,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			wantContains: []string{"CREATE TABLE", "COMMENT ON TABLE", "User table"},
		},
		{
			name:    "create table with column comment",
			fromSDL: ``,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`,
			wantContains: []string{"CREATE TABLE", "COMMENT ON COLUMN", "User name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_ObjectDropWithComment(t *testing.T) {
	tests := []struct {
		name            string
		fromSDL         string
		toSDL           string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "drop table with comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'User table';`,
			toSDL:        ``,
			wantContains: []string{"DROP TABLE"},
		},
		{
			name: "drop table with column comment",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'User name';`,
			toSDL:        ``,
			wantContains: []string{"DROP TABLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_CommentOnlyChange(t *testing.T) {
	tests := []struct {
		name            string
		fromSDL         string
		toSDL           string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "only table comment changes",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'Old comment';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`,
			wantContains:    []string{"COMMENT ON TABLE", "New comment"},
			wantNotContains: []string{"ALTER TABLE", "DROP TABLE", "CREATE TABLE"},
		},
		{
			name: "only column comment changes",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'Old name';`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN users.name IS 'New name';`,
			wantContains:    []string{"COMMENT ON COLUMN", "New name"},
			wantNotContains: []string{"ALTER TABLE", "DROP TABLE", "CREATE TABLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
			for _, s := range tt.wantNotContains {
				require.NotContains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_TableChangeWithCommentChange(t *testing.T) {
	sql := omniSDLMigration(t, `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'Old comment';`, `
CREATE TABLE users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON TABLE users IS 'New comment';`)

	require.Contains(t, sql, "ALTER TABLE")
	require.Contains(t, sql, "COMMENT ON TABLE")
	require.Contains(t, sql, "New comment")
}

func TestOmniCommentDiff_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
	}{
		{
			name: "comment with single quote",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'User''s table';`,
			wantContains: []string{"COMMENT ON TABLE", "User"},
		},
		{
			name: "comment with newlines",
			fromSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);`,
			toSDL: `
CREATE TABLE users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE users IS 'Line 1
Line 2';`,
			wantContains: []string{"COMMENT ON TABLE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}

func TestOmniCommentDiff_SchemaQualified(t *testing.T) {
	tests := []struct {
		name         string
		fromSDL      string
		toSDL        string
		wantContains []string
	}{
		{
			name: "schema qualified table comment",
			fromSDL: `
CREATE SCHEMA myschema;
CREATE TABLE myschema.users (
    id INT PRIMARY KEY
);`,
			toSDL: `
CREATE SCHEMA myschema;
CREATE TABLE myschema.users (
    id INT PRIMARY KEY
);

COMMENT ON TABLE myschema.users IS 'Users table';`,
			wantContains: []string{"COMMENT ON TABLE", "myschema", "users", "Users table"},
		},
		{
			name: "schema qualified column comment",
			fromSDL: `
CREATE SCHEMA myschema;
CREATE TABLE myschema.users (
    id INT PRIMARY KEY,
    name TEXT
);`,
			toSDL: `
CREATE SCHEMA myschema;
CREATE TABLE myschema.users (
    id INT PRIMARY KEY,
    name TEXT
);

COMMENT ON COLUMN myschema.users.name IS 'User name';`,
			wantContains: []string{"COMMENT ON COLUMN", "myschema", "User name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := omniSDLMigration(t, tt.fromSDL, tt.toSDL)
			for _, s := range tt.wantContains {
				require.Contains(t, sql, s)
			}
		})
	}
}
