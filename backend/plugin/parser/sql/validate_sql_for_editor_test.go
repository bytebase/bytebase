package parser_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

type testData struct {
	sql string
	ans bool
}

func TestValidateSQLForPG(t *testing.T) {
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
		ans := parser.ValidateSQLForEditor(parser.Postgres, test.sql)
		require.Equal(t, test.ans, ans, test.sql)
	}
}

func TestValidateSQLForMySQL(t *testing.T) {
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
				), tx as (` +
				"   select * from `delete`" +
				`) /* UPDATE */` +
				"select `update` from t;",
			ans: true,
		},
		{
			sql: `create table t (a int);`,
			ans: false,
		},
	}

	for _, test := range tests {
		ans := parser.ValidateSQLForEditor(parser.MySQL, test.sql)
		require.Equal(t, test.ans, ans, test.sql)

		ans = parser.ValidateSQLForEditor(parser.TiDB, test.sql)
		require.Equal(t, test.ans, ans, test.sql)

		ans = parser.ValidateSQLForEditor(parser.MariaDB, test.sql)
		require.Equal(t, test.ans, ans, test.sql)
	}
}

func TestValidateSQLForStandard(t *testing.T) {
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
		ans := parser.ValidateSQLForEditor(parser.Standard, test.sql)
		require.Equal(t, test.ans, ans, test.sql)

		ans = parser.ValidateSQLForEditor(parser.Oracle, test.sql)
		require.Equal(t, test.ans, ans, test.sql)

		ans = parser.ValidateSQLForEditor(parser.MSSQL, test.sql)
		require.Equal(t, test.ans, ans, test.sql)
	}
}
