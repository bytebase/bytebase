package cassandra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestParseCassandraSQL(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantStatCount int
		wantErr       bool
	}{
		{
			name:          "Single SELECT statement",
			statement:     "SELECT * FROM users;",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Multiple statements",
			statement:     "SELECT * FROM users; SELECT * FROM orders;",
			wantStatCount: 2,
			wantErr:       false,
		},
		{
			name:          "Statement without semicolon",
			statement:     "SELECT * FROM users",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Multiple statements with comments",
			statement:     "-- First query\nSELECT * FROM users;\n-- Second query\nSELECT * FROM orders;",
			wantStatCount: 2,
			wantErr:       false,
		},
		{
			name:          "Empty statement",
			statement:     "",
			wantStatCount: 0,
			wantErr:       false,
		},
		{
			name:          "Only semicolon",
			statement:     ";",
			wantStatCount: 1, // Semicolon creates one statement (not filtered by ParseCassandraSQL)
			wantErr:       false,
		},
		{
			name:          "CREATE TABLE statement",
			statement:     "CREATE TABLE users (id UUID PRIMARY KEY, name TEXT);",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Multiple DDL statements",
			statement:     "CREATE KEYSPACE test WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}; CREATE TABLE test.users (id UUID PRIMARY KEY);",
			wantStatCount: 2,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseCassandraSQL(tt.statement)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatCount, len(results))

			// Verify each result has the required fields
			for i, result := range results {
				assert.NotNil(t, result.Tree, "Result %d should have a Tree", i)
				assert.NotNil(t, result.Tokens, "Result %d should have Tokens", i)
			}
		})
	}
}

func TestParseCassandraSQLBaseLine(t *testing.T) {
	statement := `SELECT * FROM users;
	SELECT * FROM orders;
	SELECT * FROM products;`
	results, err := ParseCassandraSQL(statement)
	require.NoError(t, err)
	require.Equal(t, 3, len(results))

	// BaseLine follows the pattern from SplitSQL based on token positions
	// First statement: line 0
	assert.Equal(t, 0, base.GetLineOffset(results[0].StartPosition))
	// Second statement: includes newline+tab prefix, but first token is on line 2 (BaseLine = line - 1 = 1-1 = 0)
	// Actually the newline is part of the previous statement's text, so first real token is on line 2
	assert.Equal(t, 0, base.GetLineOffset(results[1].StartPosition)) // First token after semicolon is still on line 1 (0-indexed)
	// Third statement
	assert.Equal(t, 1, base.GetLineOffset(results[2].StartPosition))
}

func TestParseCassandraSQLErrors(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "Invalid CQL syntax",
			statement: "SELCT * FORM users;",
		},
		{
			name:      "Unclosed string",
			statement: "SELECT * FROM users WHERE name = 'test;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCassandraSQL(tt.statement)
			require.Error(t, err, "Expected error for invalid CQL")
		})
	}
}
