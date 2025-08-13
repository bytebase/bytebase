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

func TestIsPlSQLBlock_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		stmt string
	}{
		{
			name: "Very long statement",
			stmt: generateLongStatement(10000),
		},
		{
			name: "Statement with special characters",
			stmt: `DO $$ BEGIN RAISE NOTICE '` + string([]byte{0x00, 0x01, 0x02}) + `'; END $$;`,
		},
		{
			name: "Deeply nested statement",
			stmt: generateDeeplyNestedStatement(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic, just return false on error
			_ = IsPlSQLBlock(tt.stmt)
			// If we reach here, the function handled the edge case
			require.True(t, true, "Function should handle edge cases gracefully")
		})
	}
}

func generateLongStatement(length int) string {
	result := "SELECT '"
	for i := 0; i < length; i++ {
		result += "a"
	}
	result += "';"
	return result
}

func generateDeeplyNestedStatement(depth int) string {
	result := "SELECT "
	for i := 0; i < depth; i++ {
		result += "(SELECT "
	}
	result += "1"
	for i := 0; i < depth; i++ {
		result += ")"
	}
	result += ";"
	return result
}

// BenchmarkIsPlSQLBlock benchmarks the performance of IsPlSQLBlock
func BenchmarkIsPlSQLBlock(b *testing.B) {
	testCases := []struct {
		name string
		stmt string
	}{
		{
			name: "simple_do_block",
			stmt: `DO $$ BEGIN RAISE NOTICE 'test'; END $$;`,
		},
		{
			name: "complex_do_block",
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
		},
		{
			name: "select_statement",
			stmt: `SELECT * FROM users WHERE id = 1;`,
		},
		{
			name: "create_function",
			stmt: `CREATE FUNCTION test_func() RETURNS void AS $$
BEGIN
    RAISE NOTICE 'Hello';
END
$$ LANGUAGE plpgsql;`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = IsPlSQLBlock(tc.stmt)
			}
		})
	}
}
