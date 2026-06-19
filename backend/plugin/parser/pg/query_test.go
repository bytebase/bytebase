package pg

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			// Security regression: a MERGE smuggled into a data-modifying CTE must
			// be treated as a write (not read-only), so the editor routes it to Exec
			// (RETURNING rows discarded) and export rejects it. Before the fix,
			// hasDMLInTree missed MergeStmt, so this took the read path and returned
			// masked columns unmasked.
			sql:      `WITH x AS (MERGE INTO t a USING t b ON a.id = b.id WHEN MATCHED THEN UPDATE SET id = a.id RETURNING a.secret) SELECT secret FROM x`,
			valid:    false,
			allQuery: false,
		},
		{
			// Top-level MERGE ... RETURNING is already blocked by the default case;
			// pin it so the classifier sets stay aligned.
			sql:      `MERGE INTO t a USING t b ON a.id = b.id WHEN MATCHED THEN UPDATE SET id = a.id RETURNING a.secret`,
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
		// Multi-statement tests
		{
			sql:      `SELECT * FROM t1; SELECT * FROM t2;`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `SELECT * FROM t1; SELECT * FROM t2; SHOW search_path;`,
			valid:    true,
			allQuery: true,
		},
		{
			sql:      `SET work_mem = '128MB'; SELECT * FROM t1;`,
			valid:    true,
			allQuery: false, // SET doesn't return data
		},
		{
			sql:      `SELECT * FROM t1; INSERT INTO t2 VALUES (1);`,
			valid:    false, // INSERT not allowed
			allQuery: false,
		},
		{
			sql:      `SELECT * FROM t1; CREATE TABLE t2 (a int);`,
			valid:    false, // DDL not allowed
			allQuery: false,
		},
		{
			sql:      `EXPLAIN SELECT * FROM t1; SELECT * FROM t2;`,
			valid:    true,
			allQuery: true,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQueryANTLR(test.sql)
		require.NoError(t, err)
		require.Equal(t, test.valid, gotValid, test.sql)
		require.Equal(t, test.allQuery, gotAllQuery, test.sql)
	}
}
