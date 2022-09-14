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
		engineType parser.EngineType
		oldSchema  string
		newSchema  string
		want       string
		err        error
	}{
		{
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
			err: nil,
		},
		{
			engineType: parser.Postgres,
			oldSchema:  `CREATE TABLE projects ();`,
			newSchema: `CREATE TABLE projects (
	id serial PRIMARY KEY
);`,
			// FIXME(@joe): this is an unwanted result.
			want: ``,
			err:  nil,
		},
	}

	for _, test := range tests {
		oldSchemaNodes, err := parser.Parse(test.engineType, parser.ParseContext{}, test.oldSchema)
		require.NoError(t, err)
		newSchemaNodes, err := parser.Parse(test.engineType, parser.ParseContext{}, test.newSchema)
		require.NoError(t, err)

		diff, err := parser.SchemaDiff(oldSchemaNodes, newSchemaNodes)
		if err != nil {
			if test.err != nil {
				require.Equal(t, test.err.Error(), err.Error())
			} else {
				t.Error(err)
			}
		} else {
			require.Equal(t, test.want, diff)
		}
	}
}
