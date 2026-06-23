package cassandra

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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

func TestSyntaxErrorIsSyntaxError(t *testing.T) {
	_, err := parseCassandraStatements("SELCT * FORM users")
	require.Error(t, err)
	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr), "parse error should be *base.SyntaxError, got %T", err)
	require.NotNil(t, syntaxErr.Position)
}

func TestSyntaxErrorColumnOffset(t *testing.T) {
	input := "SELECT * FROM t;\n  SELCT * FROM users"
	_, err := parseCassandraStatements(input)
	require.Error(t, err)
	var syntaxErr *base.SyntaxError
	require.True(t, errors.As(err, &syntaxErr))

	// The splitter trims "SELCT * FROM users" starting at line 2, col 3.
	// The parse error on "SELCT" is at byte 0 of the trimmed text → local (line=1, col=1).
	// After adjustment: line = 1 + 2 - 1 = 2, col = 1 + 3 - 1 = 3.
	require.Equal(t, int32(2), syntaxErr.Position.Line)
	require.Equal(t, int32(3), syntaxErr.Position.Column)
}

func TestParseCassandraOmniAST(t *testing.T) {
	results, err := parseCassandraStatements("SELECT id, name FROM users")
	require.NoError(t, err)
	require.Len(t, results, 1)

	node, ok := GetOmniNode(results[0].AST)
	require.True(t, ok, "AST should be an OmniAST")
	require.NotNil(t, node)
}

func TestParseCassandraComprehensiveDDL(t *testing.T) {
	ddlStatements := []string{
		"CREATE KEYSPACE ks WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}",
		"ALTER KEYSPACE ks WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 3}",
		"DROP KEYSPACE IF EXISTS ks",
		"CREATE TABLE ks.t (id int PRIMARY KEY, name text, data blob)",
		"ALTER TABLE ks.t ADD email text",
		"DROP TABLE IF EXISTS ks.t",
		"CREATE INDEX idx ON ks.t (name)",
		"DROP INDEX IF EXISTS ks.idx",
		"CREATE TYPE ks.addr (street text, city text)",
		"ALTER TYPE ks.addr ADD zip text",
		"DROP TYPE IF EXISTS ks.addr",
		"CREATE MATERIALIZED VIEW ks.mv AS SELECT id, name FROM ks.t WHERE id IS NOT NULL AND name IS NOT NULL PRIMARY KEY (name, id)",
		"ALTER MATERIALIZED VIEW ks.mv WITH compression = {'sstable_compression': 'LZ4Compressor'}",
		"DROP MATERIALIZED VIEW IF EXISTS ks.mv",
		"CREATE FUNCTION ks.double(val int) RETURNS NULL ON NULL INPUT RETURNS int LANGUAGE java AS 'return val * 2;'",
		"DROP FUNCTION IF EXISTS ks.double",
		"CREATE TRIGGER tr ON ks.t USING 'org.example.Trigger'",
		"DROP TRIGGER IF EXISTS tr ON ks.t",
		"TRUNCATE ks.t",
	}
	for _, stmt := range ddlStatements {
		name := stmt
		if len(name) > 40 {
			name = name[:40]
		}
		t.Run(name, func(t *testing.T) {
			results, err := parseCassandraStatements(stmt)
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.NotNil(t, results[0].AST)
		})
	}
}
