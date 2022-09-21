package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name      string
		oldSchema string
		newSchema string
		want      string
		errPart   string
	}{
		{
			name:      "diffCreateTableInPostgres",
			oldSchema: `CREATE TABLE projects ();`,
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

	pgDiffer := &SchemaDiffer{}
	for _, test := range tests {
		diff, err := pgDiffer.SchemaDiff(test.oldSchema, test.newSchema)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart, test.name)
		}
		require.Equal(t, test.want, diff)
	}
}
