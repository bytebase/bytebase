package taskcheck

import (
	"testing"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

func TestPostgresExtractResourcesFromCommentStatement(t *testing.T) {
	tests := []struct {
		statement string
		resources []parser.SchemaResource
	}{
		{
			statement: `COMMENT ON COLUMN public.user.id IS 'The unique ID of the user.';`,
			resources: []parser.SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "user",
				},
			},
		},
		{
			statement: `COMMENT ON CONSTRAINT c1 ON "user" IS 'The unique ID of the user.';`,
			resources: []parser.SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "user",
				},
			},
		},
		{
			statement: `COMMENT ON TABLE c1 IS 'The unique ID of the user.';`,
			resources: []parser.SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "c1",
				},
			},
		},
	}

	a := require.New(t)
	for _, test := range tests {
		res, err := postgresExtractResourcesFromCommentStatement("db", "public", test.statement)
		a.NoError(err)
		a.Equal(test.resources, res)
	}
}
