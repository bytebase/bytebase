package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		gotAllQuery bool
		err         bool
	}{
		{
			statement:   "SHOW TABLES;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "DESC TABLE bytebase;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "CREATE TABLE t1 (c1 INT);",
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   "UPDATE t1 SET c1 = 1;",
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   "EXPLAIN SELECT * FROM t1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `select* from t`,
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `explain select * from t;`,
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `create table t (a int);`,
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   `SET max_execution_time = 1000; select * from t`,
			valid:       true,
			gotAllQuery: false,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.valid, gotValid, test.statement)
			require.Equal(t, test.gotAllQuery, gotAllQuery, test.statement)
		}
	}
}

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
