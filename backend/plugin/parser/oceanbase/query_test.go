package oceanbase

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
			// OceanBase has no EXPLAIN ANALYZE; the parser refuses it.
			statement: `explain    analyze select * from t`,
			err:       true,
		},
		{
			statement: `explain    analyze update t set a = 5`,
			err:       true,
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
		{
			statement: "SELECT * FORM invalid_table;",
			err:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.statement, func(t *testing.T) {
			gotValid, gotAllQuery, err := validateQuery(test.statement)
			if test.err {
				require.Error(t, err)
				require.ErrorAs(t, err, new(*base.SyntaxError))
			} else {
				require.NoError(t, err)
				require.Equal(t, test.valid, gotValid, test.statement)
				require.Equal(t, test.gotAllQuery, gotAllQuery, test.statement)
			}
		})
	}
}

// OceanBase has no EXPLAIN ANALYZE (its grammar answers 1064), so only a plain
// EXPLAIN reaches validation here; it plans without running whatever it wraps.
func TestValidateQueryExplainAnalyze(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		wantValid bool
	}{
		{"a plain EXPLAIN plans without running", "EXPLAIN SELECT count(*) FROM t WHERE a = 1", true},
		{"a plain EXPLAIN of a DML still only plans", "EXPLAIN DELETE FROM t WHERE a = 1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, _, err := validateQuery(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, gotValid, tc.statement)
		})
	}
}

// An export under EXPLAIN is still an export request: the editor must not label
// as a read what it refuses when asked directly. The reads beneath keep the
// gate from being satisfied by refusing every EXPLAIN.
func TestValidateQueryExportGate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		wantValid bool
	}{
		{"a bare export", "SELECT a FROM t INTO OUTFILE '/tmp/x'", false},
		{"a bare dump", "SELECT a FROM t INTO DUMPFILE '/tmp/x'", false},
		{"an export on the second arm", "SELECT 1 UNION SELECT a FROM t INTO OUTFILE '/tmp/x'", false},
		{"an export in a braced arm", "(SELECT a FROM t INTO OUTFILE '/tmp/x') UNION (SELECT 1)", false},
		{"an export inside a subquery", "SELECT * FROM (SELECT a FROM t INTO OUTFILE '/tmp/x') y", false},
		{"an export under a plain EXPLAIN", "EXPLAIN SELECT a FROM t INTO OUTFILE '/tmp/x'", false},
		{"a dump under a plain EXPLAIN", "EXPLAIN SELECT a FROM t INTO DUMPFILE '/tmp/x'", false},
		{"an export on an arm under a plain EXPLAIN", "EXPLAIN SELECT 1 UNION SELECT a FROM t INTO OUTFILE '/tmp/x'", false},

		{"a set operation that exports nothing", "SELECT 1 UNION SELECT 2", true},
		{"a variable assignment writes no file", "SELECT a FROM t INTO @v", true},
		{"a variable assignment under a plain EXPLAIN only plans", "EXPLAIN SELECT a FROM t INTO @v", true},
		{"the clause named in a string literal", "SELECT 'INTO OUTFILE' FROM t", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, _, err := validateQuery(tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, gotValid, tc.statement)
		})
	}
}
