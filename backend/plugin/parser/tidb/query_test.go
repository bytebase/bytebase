package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		gotAllQuery bool
		err         bool
	}{
		{
			statement:   "SELECT 1 UNION ALL SELECT 2;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "SHOW CREATE TABLE bytebase;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "DESC bytebase;",
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
			statement:   "EXPLAIN FORMAT=JSON DELETE FROM t1;",
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
			statement:   `explain    analyze select * from t`,
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `explain    analyze update t set a = 5`,
			valid:       false,
			gotAllQuery: true,
		},
		{
			statement: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				update t set a = 1;
				`,
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement: `
				With t as (
					select * from t1
				), tx as (
					select * from t1
				)
				insert into t values (1, 2, 3);
				`,
			valid:       false,
			gotAllQuery: false,
			err:         true,
		},
		{
			statement:   "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement: `
				With t as (
					select * from t1 where a = 'insert'
				), tx as (` +
				"   select * from `delete`" +
				`) /* UPDATE */` +
				"select `update` from t;",
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

// EXPLAIN ANALYZE executes what it wraps, unlike a plain EXPLAIN, so it is
// allowed over a read and refused over everything else. Before this it was
// refused outright, which cost the legitimate case: profiling a SELECT.
func TestValidateQueryExplainAnalyze(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		wantValid bool
	}{
		{"a plain EXPLAIN plans without running", "EXPLAIN SELECT count(*) FROM t WHERE a = 1", true},
		{"EXPLAIN ANALYZE over a read", "EXPLAIN ANALYZE SELECT count(*) FROM t WHERE a = 1", true},
		{"EXPLAIN ANALYZE over a set operation", "EXPLAIN ANALYZE SELECT a FROM t UNION SELECT a FROM s", true},
		{"EXPLAIN ANALYZE really deletes", "EXPLAIN ANALYZE DELETE FROM t WHERE a = 1", false},
		{"EXPLAIN ANALYZE really inserts", "EXPLAIN ANALYZE INSERT INTO t SELECT a FROM s", false},
		{"EXPLAIN ANALYZE really updates", "EXPLAIN ANALYZE UPDATE t SET a = 1", false},
		{"a plain EXPLAIN of a DML still only plans", "EXPLAIN DELETE FROM t WHERE a = 1", true},
		{"EXPLAIN ANALYZE over a SELECT that writes a file", "EXPLAIN ANALYZE SELECT a FROM t INTO OUTFILE '/tmp/x'", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, _, err := validateQuery(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, gotValid, tc.statement)
		})
	}
}
