package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/parser"
	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

func TestComputeDiff(t *testing.T) {
	oldSchema, err := parser.Parse(parser.Postgres, parser.ParseContext{}, `
CREATE TABLE projects ();
`)
	require.NoError(t, err)
	newSchema, err := parser.Parse(parser.Postgres, parser.ParseContext{}, `
CREATE TABLE users (
	id serial PRIMARY KEY,
	username TEXT NOT NULL
);
CREATE TABLE projects ();
CREATE TABLE repositories (
	id serial PRIMARY KEY
);
`)
	require.NoError(t, err)

	got, err := parser.SchemaDiff(oldSchema, newSchema)
	require.NoError(t, err)

	// The DDLs for the diff should be in the exact same order as the DDLs in the new schema.
	want := `CREATE TABLE users (
	id serial PRIMARY KEY,
	username TEXT NOT NULL
);
CREATE TABLE repositories (
	id serial PRIMARY KEY
);
`
	assert.Equal(t, want, got)
}
