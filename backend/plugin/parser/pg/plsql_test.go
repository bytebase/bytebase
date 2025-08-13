package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPlSQLBlock(t *testing.T) {
	tests := []struct {
		name     string
		stmt     string
		expected bool
	}{
		// DO blocks - should return true
		{
			name: "simple DO block with dollar quotes",
			stmt: `DO $$
BEGIN
    RAISE NOTICE 'Hello World';
END
$$;`,
			expected: true,
		},
		{
			name: "DO block with single quotes",
			stmt: `DO '
BEGIN
    RAISE NOTICE ''Hello World'';
END
';`,
			expected: true,
		},
		{
			name: "DO block with language specification",
			stmt: `DO $$
BEGIN
    RAISE NOTICE 'Hello World';
END
$$ LANGUAGE plpgsql;`,
			expected: true,
		},
		{
			name: "DO block with complex logic",
			stmt: `DO $$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT table_name FROM information_schema.tables
    WHERE table_schema = 'public'
    LOOP
        RAISE NOTICE 'Table: %', r.table_name;
    END LOOP;
END
$$;`,
			expected: true,
		},
		{
			name: "DO block with nested dollar quotes",
			stmt: `DO $outer$
BEGIN
    EXECUTE $inner$
        SELECT 1;
    $inner$;
END
$outer$;`,
			expected: true,
		},
		{
			name:     "DO block in single line",
			stmt:     `DO $$ BEGIN RAISE NOTICE 'test'; END $$;`,
			expected: true,
		},
		{
			name:     "DO block with whitespace",
			stmt:     `  DO  $$  BEGIN  RAISE NOTICE 'test';  END  $$  ;  `,
			expected: true,
		},
		{
			name: "DO block with exception handling",
			stmt: `DO $$
BEGIN
    INSERT INTO test_table VALUES (1);
EXCEPTION
    WHEN unique_violation THEN
        RAISE NOTICE 'Duplicate key';
END
$$;`,
			expected: true,
		},

		// Non-DO blocks - should return false
		{
			name:     "SELECT statement",
			stmt:     `SELECT * FROM users;`,
			expected: false,
		},
		{
			name: "CREATE FUNCTION statement",
			stmt: `CREATE FUNCTION test_func() RETURNS void AS $$
BEGIN
    RAISE NOTICE 'Hello';
END
$$ LANGUAGE plpgsql;`,
			expected: false,
		},
		{
			name:     "INSERT statement",
			stmt:     `INSERT INTO users (name) VALUES ('John');`,
			expected: false,
		},
		{
			name:     "UPDATE statement",
			stmt:     `UPDATE users SET name = 'Jane' WHERE id = 1;`,
			expected: false,
		},
		{
			name:     "DELETE statement",
			stmt:     `DELETE FROM users WHERE id = 1;`,
			expected: false,
		},
		{
			name: "CREATE PROCEDURE statement",
			stmt: `CREATE PROCEDURE test_proc() AS $$
BEGIN
    RAISE NOTICE 'Hello';
END
$$ LANGUAGE plpgsql;`,
			expected: false,
		},
		{
			name:     "BEGIN transaction block",
			stmt:     `BEGIN; SELECT 1; COMMIT;`,
			expected: false,
		},
		{
			name:     "DECLARE CURSOR statement",
			stmt:     `DECLARE my_cursor CURSOR FOR SELECT * FROM users;`,
			expected: false,
		},
		{
			name:     "Anonymous block in string (not DO)",
			stmt:     `SELECT 'DO $$ BEGIN RAISE NOTICE ''test''; END $$;' AS query;`,
			expected: false,
		},
		{
			name:     "Comment containing DO",
			stmt:     `-- DO something here\nSELECT 1;`,
			expected: false,
		},
		{
			name:     "Table named 'do'",
			stmt:     `SELECT * FROM do WHERE id = 1;`,
			expected: false,
		},
		{
			name:     "Column named 'do'",
			stmt:     `SELECT do FROM tasks;`,
			expected: false,
		},

		// Edge cases
		{
			name:     "Empty statement",
			stmt:     ``,
			expected: false,
		},
		{
			name:     "Only whitespace",
			stmt:     `   `,
			expected: false,
		},
		{
			name:     "Only DO keyword",
			stmt:     `DO`,
			expected: false,
		},
		{
			name:     "DO without body",
			stmt:     `DO;`,
			expected: false,
		},
		{
			name:     "Malformed DO block",
			stmt:     `DO $$ BEGIN`,
			expected: false,
		},
		{
			name: "Multiple statements with DO block",
			stmt: `SELECT 1;
DO $$ BEGIN RAISE NOTICE 'test'; END $$;`,
			expected: false, // Should be false because it's not a single statement
		},
		{
			name:     "DO in uppercase",
			stmt:     `DO $$ BEGIN RAISE NOTICE 'test'; END $$;`,
			expected: true,
		},
		{
			name:     "do in lowercase",
			stmt:     `do $$ BEGIN RAISE NOTICE 'test'; END $$;`,
			expected: true,
		},
		{
			name:     "Do in mixed case",
			stmt:     `Do $$ BEGIN RAISE NOTICE 'test'; END $$;`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPlSQLBlock(tt.stmt)
			require.Equal(t, tt.expected, result, "Statement: %s", tt.stmt)
		})
	}
}
