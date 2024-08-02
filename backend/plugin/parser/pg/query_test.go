package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	type testData struct {
		sql      string
		valid    bool
		allQuery bool
	}
	tests := []testData{
		{
			sql:      `select* from t`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `explain select * from t;`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `explain    analyze select * from t`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `explain    analyze insert into t values (1)`,
			valid:    false,
			allQuery: false,
		},
		{
			sql:      `EXPLAIN ANALYZE WITH cte1 AS (DELETE FROM t RETURNING id) SELECT * FROM cte1`,
			valid:    false,
			allQuery: false,
		},
		{
			sql: `
				With t as (
					select * from t1
				), tx as (
					delete from t2
				)
				select * from t;
				`,
			valid:    false,
			allQuery: false,
		},
		{
			sql: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				update t set a = 1;
				`,
			valid:    false,
			allQuery: false,
		},
		{
			sql: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				insert into t values (1, 2, 3);
				`,
			valid:    false,
			allQuery: false,
		},
		{
			sql:      "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			valid:    true,
			allQuery: true,
		},
		{
			sql: `
				With t as (
					select * from t1 where a = 'insert'
				), tx as (
					select * from "delete"
				) /* UPDATE */
				select "update" from t;
				`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `create table t (a int);`,
			valid:    false,
			allQuery: false,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQuery(test.sql)
		require.NoError(t, err)
		require.Equal(t, test.valid, gotValid, test.sql)
		require.Equal(t, test.allQuery, gotAllQuery, test.sql)
	}
}

func TestExtractPostgresResourceList(t *testing.T) {
	tests := []struct {
		statement string
		want      []base.SchemaResource
	}{
		{
			statement: `SELECT * FROM t;SELECT * FROM t1;`,
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "t",
				},
				{
					Database: "db",
					Schema:   "public",
					Table:    "t1",
				},
			},
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1;",
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "schema1",
					Table:    "t1",
				},
				{
					Database: "db",
					Schema:   "schema2",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			want: []base.SchemaResource{
				{
					Database: "db",
					Schema:   "public",
					Table:    "t1",
				},
				{
					Database: "db",
					Schema:   "public",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := ExtractResourceList("db", "public", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res)
	}
}
