package pg

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	type testData struct {
		sql string
		ans bool
	}
	tests := []testData{
		{
			sql: `select* from t`,
			ans: true,
		},
		{
			sql: `explain select * from t;`,
			ans: true,
		},
		{
			sql: `explain    analyze select * from t`,
			ans: false,
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
			ans: false,
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
			ans: false,
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
			ans: false,
		},
		{
			sql: "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			ans: true,
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
			ans: true,
		},
		{
			sql: `create table t (a int);`,
			ans: false,
		},
	}

	for _, test := range tests {
		ans, err := validateQuery(test.sql)
		require.NoError(t, err)
		require.Equal(t, test.ans, ans, test.sql)
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
