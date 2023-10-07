package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExtractSnowflakeNormalizeResourceListFromSelectStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      []base.SchemaResource
	}{
		{
			statement: `SELECT * FROM T1;SELECT * FROM T2;`,
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T1",
				},
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T2",
				},
			},
		},
		{
			statement: `SELECT * FROM t1;SELECT * FROM T1;`,
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T1",
				},
			},
		},
		{
			statement: `SELECT * FROM t1;SELECT * FROM t2;`,
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T1",
				},
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT * FROM SCHEMA_1.T1 JOIN SCHEMA_2.T2 ON T1.C1 = T2.C2;",
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "SCHEMA_1",
					Table:    "T1",
				},
				{
					Database: "db",
					Schema:   "SCHEMA_2",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT * FROM DB_1.SCHEMA_1.T1 JOIN DB_2.SCHEMA_2.T2 ON T1.C1 = T2.C2;",
			want: []base.SchemaResource{
				{
					Database: "DB_1",
					Schema:   "SCHEMA_1",
					Table:    "T1",
				},
				{
					Database: "DB_2",
					Schema:   "SCHEMA_2",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT A > (SELECT MAX(A) FROM T1) FROM T2;",
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T1",
				},
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T2",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := ExtractResourceList("db", "PUBLIC", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res, "for statement: %v", test.statement)
	}
}
