package standard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLForStandard(t *testing.T) {
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
		ans, err := ValidateSQLForEditor(test.sql)
		require.NoError(t, err)
		require.Equal(t, test.ans, ans, test.sql)
	}
}
