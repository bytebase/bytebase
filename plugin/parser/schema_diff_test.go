package parser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/parser"
	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name       string
		engineType parser.EngineType
		oldSchema  string
		newSchema  string
		want       string
		errPart    string
	}{
		{
			name:       "diffCreateTableInPostgres",
			engineType: parser.Postgres,
			oldSchema:  `CREATE TABLE projects ();`,
			newSchema: `CREATE TABLE users (
	id serial PRIMARY KEY,
	username TEXT NOT NULL
);
CREATE TABLE projects ();
CREATE TABLE repositories (
	id serial PRIMARY KEY
);`,
			want: `CREATE TABLE users (
	id serial PRIMARY KEY,
	username TEXT NOT NULL
);
CREATE TABLE repositories (
	id serial PRIMARY KEY
);
`,
			errPart: "",
		},
	}

	for _, test := range tests {
		oldSchemaNodes, err := parser.Parse(test.engineType, parser.ParseContext{}, test.oldSchema)
		// This is an unrelated error and should always be nil.
		require.NoError(t, err)
		newSchemaNodes, err := parser.Parse(test.engineType, parser.ParseContext{}, test.newSchema)
		require.NoError(t, err)

		diff, err := parser.SchemaDiff(oldSchemaNodes, newSchemaNodes)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart, test.name)
		}
		require.Equal(t, test.want, diff)
	}
}
