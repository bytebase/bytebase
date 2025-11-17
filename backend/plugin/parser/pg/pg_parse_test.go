package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePostgreSQL(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantStatCount int
		wantErr       bool
	}{
		{
			name:          "Single SELECT statement",
			statement:     "SELECT * FROM users",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "SELECT with WHERE clause",
			statement:     "SELECT id, name FROM users WHERE age > 18",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Multiple statements with semicolon",
			statement:     "SELECT * FROM t1; SELECT * FROM t2;",
			wantStatCount: 2,
			wantErr:       false,
		},
		{
			name:          "INSERT statement",
			statement:     "INSERT INTO users (id, name) VALUES (1, 'John')",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "UPDATE statement",
			statement:     "UPDATE users SET name = 'Jane' WHERE id = 1",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "DELETE statement",
			statement:     "DELETE FROM users WHERE id = 1",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "CREATE TABLE statement",
			statement:     "CREATE TABLE users (id INT PRIMARY KEY, name TEXT)",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Empty string",
			statement:     "",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Only whitespace",
			statement:     "   \n  \t  ",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Only semicolon",
			statement:     ";",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Multiple semicolons",
			statement:     ";;",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Statement with trailing semicolon",
			statement:     "SELECT * FROM users;",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Statement without trailing semicolon",
			statement:     "SELECT * FROM users",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Comment only",
			statement:     "-- This is a comment",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Statement with comment",
			statement:     "SELECT * FROM users -- Get all users",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Block comment",
			statement:     "/* This is a block comment */",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Statement with block comment",
			statement:     "SELECT /* comment */ * FROM users",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:      "Invalid SQL syntax",
			statement: "SELCT * FRM users",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParsePostgreSQL(tt.statement)
			if tt.wantErr {
				require.Error(t, err, "Expected error for invalid PostgreSQL SQL")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatCount, len(results))

			// Verify each result has the required fields
			for i, result := range results {
				assert.NotNil(t, result.Tree, "Result %d should have a Tree", i)
				assert.NotNil(t, result.Tokens, "Result %d should have Tokens", i)
				assert.GreaterOrEqual(t, result.BaseLine, 0, "Result %d should have non-negative BaseLine", i)
			}
		})
	}
}

func TestParsePostgreSQLBaseLine(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		wantLines []int
	}{
		{
			name:      "Single statement",
			statement: "SELECT * FROM users",
			wantLines: []int{0},
		},
		{
			name:      "Two statements on same line",
			statement: "SELECT * FROM t1; SELECT * FROM t2;",
			wantLines: []int{0, 0},
		},
		{
			name: "Two statements on different lines",
			statement: `SELECT * FROM t1;
SELECT * FROM t2;`,
			wantLines: []int{0, 1},
		},
		{
			name: "Three statements on different lines",
			statement: `SELECT * FROM t1;
SELECT * FROM t2;
SELECT * FROM t3;`,
			wantLines: []int{0, 1, 2},
		},
		{
			name: "Statements with empty lines",
			statement: `SELECT * FROM t1;

SELECT * FROM t2;`,
			wantLines: []int{0, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParsePostgreSQL(tt.statement)
			require.NoError(t, err)
			require.Equal(t, len(tt.wantLines), len(results))

			for i, result := range results {
				assert.Equal(t, tt.wantLines[i], result.BaseLine, "Statement %d BaseLine mismatch", i)
			}
		})
	}
}
