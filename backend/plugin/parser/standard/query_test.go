package standard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLForStandard(t *testing.T) {
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
		gotValid, gotAllQuery, err := ValidateSQLForEditor(test.sql)
		require.NoError(t, err)
		require.Equal(t, test.valid, gotValid, test.sql)
		require.Equal(t, test.allQuery, gotAllQuery, test.sql)
	}
}
