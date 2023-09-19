package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnowSqlExtractOrdinaryIdentifier(t *testing.T) {
	testCases := []struct {
		description string
		name        string
		want        string
	}{
		{
			description: "Should convert object name to uppercase if it is not quoted",
			name:        `table_name`,
			want:        `TABLE_NAME`,
		},
		{
			description: "Should **NOT** convert object name to uppercase if it is quoted",
			name:        `"table_name"`,
			want:        `table_name`,
		},
		{
			description: `Should convert '""' to '"' if it is quoted`,
			name:        `"table_name"""`,
			want:        `table_name"`,
		},
		{
			description: `Should be fine with unicode characters`,
			name:        `"😈😄"""`,
			want:        `😈😄"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractSnowSQLOrdinaryIdentifier(tc.name)
			require.Equal(t, tc.want, got, tc.description)
		})
	}
}

func TestExtractSnowflakeNormalizeResourceListFromSelectStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      []SchemaResource
	}{
		{
			statement: `SELECT * FROM T1;SELECT * FROM T2;`,
			want: []SchemaResource{
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
			want: []SchemaResource{
				{
					Database: "db",
					Schema:   "PUBLIC",
					Table:    "T1",
				},
			},
		},
		{
			statement: `SELECT * FROM t1;SELECT * FROM t2;`,
			want: []SchemaResource{
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
			want: []SchemaResource{
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
			want: []SchemaResource{
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
			want: []SchemaResource{
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
		res, err := extractSnowflakeNormalizeResourceListFromSelectStatement("db", "PUBLIC", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res, "for statement: %v", test.statement)
	}
}
