package cassandra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCassandraStatements(t *testing.T) {
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
			name:          "CREATE TABLE statement",
			statement:     "CREATE TABLE users (id int PRIMARY KEY, name text);",
			wantStatCount: 1,
			wantErr:       false,
		},
		{
			name:          "Multiple DDL statements",
			statement:     "CREATE KEYSPACE test WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': '1'}; CREATE TABLE test.users (id int PRIMARY KEY);",
			wantStatCount: 2,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := parseCassandraStatements(tt.statement)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatCount, len(results))

			for i, result := range results {
				assert.NotNil(t, result.AST, "Result %d should have an AST", i)
				assert.NotEmpty(t, result.Text, "Result %d should have Text", i)
			}
		})
	}
}

func TestParseCassandraStatementsPositions(t *testing.T) {
	statement := "SELECT * FROM users;\nSELECT * FROM orders;\nSELECT * FROM products;"
	results, err := parseCassandraStatements(statement)
	require.NoError(t, err)
	require.Equal(t, 3, len(results))

	assert.Equal(t, int32(1), results[0].Start.Line)
	assert.Equal(t, int32(2), results[1].Start.Line)
	assert.Equal(t, int32(3), results[2].Start.Line)
}

func TestParseCassandraStatementsErrors(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{
			name:      "Invalid CQL syntax",
			statement: "SELCT * FORM users;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCassandraStatements(tt.statement)
			require.Error(t, err, "Expected error for invalid CQL")
		})
	}
}
