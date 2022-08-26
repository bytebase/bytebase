package schemadiff_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/schemadiff"
	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
	// Register PostgreSQL differ engine.
	_ "github.com/bytebase/bytebase/plugin/schemadiff/engine/pg"
)

// todo: add more test cases
func TestCompute(t *testing.T) {
	oldSchema, err := parser.Parse(parser.Postgres, parser.Context{}, `
CREATE TABLE users (
	id serial PRIMARY KEY
);
CREATE TABLE repositories (
	id serial PRIMARY KEY,
	name VARCHAR(255) NOT NULL
);
`)
	require.NoError(t, err)
	newSchema, err := parser.Parse(parser.Postgres, parser.Context{}, `
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

	diff, err := schemadiff.Compute(parser.Postgres, oldSchema, newSchema)
	require.NoError(t, err)
	got := diff.String()

	want := `ALTER TABLE "users" ADD COLUMN "username" text;
CREATE TABLE projects ();
ALTER TABLE "repositories" DROP COLUMN "name";
`
	assert.Equal(t, want, got)
}
