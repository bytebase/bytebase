package doris

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		description string
	}{
		{
			statement:   "SELECT * FROM users",
			valid:       true,
			description: "Basic SELECT query",
		},
		{
			statement:   "SELECT * FROM users WHERE id = 1",
			valid:       true,
			description: "SELECT with WHERE clause",
		},
		{
			statement:   "SHOW DATA",
			valid:       true,
			description: "SHOW DATA statement",
		},
		{
			statement:   "SHOW DATA FROM db1",
			valid:       true,
			description: "SHOW DATA with FROM clause",
		},
		{
			statement:   "SHOW DATABASES",
			valid:       true,
			description: "SHOW DATABASES statement",
		},
		{
			statement:   "SHOW TABLES",
			valid:       true,
			description: "SHOW TABLES statement",
		},
		{
			statement:   "SHOW TABLETS FROM table1",
			valid:       true,
			description: "SHOW TABLETS statement",
		},
		{
			statement:   "SHOW VARIABLES",
			valid:       true,
			description: "SHOW VARIABLES statement",
		},
		{
			statement:   "SHOW CREATE TABLE users",
			valid:       true,
			description: "SHOW CREATE TABLE statement",
		},
		{
			statement:   "SHOW CREATE DATABASE db1",
			valid:       true,
			description: "SHOW CREATE DATABASE statement",
		},
		{
			statement:   "INSERT INTO users (id, name) VALUES (1, 'test')",
			valid:       false,
			description: "INSERT statement should be invalid",
		},
		{
			statement:   "UPDATE users SET name = 'test' WHERE id = 1",
			valid:       false,
			description: "UPDATE statement should be invalid",
		},
		{
			statement:   "DELETE FROM users WHERE id = 1",
			valid:       false,
			description: "DELETE statement should be invalid",
		},
		{
			statement:   "CREATE TABLE test (id INT)",
			valid:       false,
			description: "CREATE TABLE should be invalid",
		},
		{
			statement:   "DROP TABLE users",
			valid:       false,
			description: "DROP TABLE should be invalid",
		},
		{
			statement:   "EXPLAIN INSERT INTO users (id, name) VALUES (1, 'test')",
			valid:       true,
			description: "EXPLAIN INSERT should be valid (read-only)",
		},
		{
			statement:   "EXPLAIN UPDATE users SET name = 'test' WHERE id = 1",
			valid:       true,
			description: "EXPLAIN UPDATE should be valid (read-only)",
		},
		{
			statement:   "EXPLAIN DELETE FROM users WHERE id = 1",
			valid:       true,
			description: "EXPLAIN DELETE should be valid (read-only)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			a := require.New(t)
			valid, _, err := validateQuery(tc.statement)
			a.NoError(err)
			a.Equal(tc.valid, valid, "statement: %s", tc.statement)
		})
	}
}
