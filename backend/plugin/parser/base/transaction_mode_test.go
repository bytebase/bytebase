package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanDirectives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no directives",
			input:    "ALTER TABLE users ADD COLUMN age INT;",
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "txn-mode directive only",
			input: `-- txn-mode = on
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "txn-isolation directive only",
			input: `-- txn-isolation = REPEATABLE READ
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "ghost directive only",
			input: `-- gh-ost = {"max-lag-millis":"1500"}
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "all directives",
			input: `-- txn-mode = on
-- txn-isolation = REPEATABLE READ
-- gh-ost = {"cut-over":"default"}
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "directives with other comments",
			input: `-- txn-mode = on
-- This is a regular comment
-- gh-ost = {}
ALTER TABLE users ADD COLUMN age INT;`,
			expected: `-- This is a regular comment
ALTER TABLE users ADD COLUMN age INT;`,
		},
		{
			name: "case insensitive",
			input: `-- TXN-MODE = ON
-- TXN-ISOLATION = repeatable read
-- GH-OST = {"key":"value"}
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name: "directives with extra spaces",
			input: `--   txn-mode   =   on
--  gh-ost  =  {"key":"value"}
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name: "multiline statement preserved",
			input: `-- gh-ost = {}
CREATE TABLE users (
    id INT PRIMARY KEY,
    name VARCHAR(100)
);`,
			expected: `CREATE TABLE users (
    id INT PRIMARY KEY,
    name VARCHAR(100)
);`,
		},
		{
			name: "ghost directive with comment",
			input: `-- gh-ost = {} /*gh-ost with default config*/
ALTER TABLE users ADD COLUMN age INT;`,
			expected: "ALTER TABLE users ADD COLUMN age INT;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanDirectives(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
