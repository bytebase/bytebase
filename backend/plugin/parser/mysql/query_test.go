package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestMySQLValidateForEditor(t *testing.T) {
	tests := []struct {
		statement string
		validate  bool
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			validate:  true,
		},
		{
			statement: "CREATE TABLE t1 (c1 INT);",
			validate:  false,
		},
		{
			statement: "UPDATE t1 SET c1 = 1;",
			validate:  false,
		},
		{
			statement: "EXPLAIN SELECT * FROM t1;",
			validate:  true,
		},
		{
			statement: "EXPLAIN FORMAT=JSON DELETE FROM t1;",
			validate:  false,
		},
	}

	for _, test := range tests {
		got := ValidateSQLForEditor(test.statement)
		require.Equal(t, test.validate, got)
	}
}

func TestExtractMySQLResourceList(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT * FROM db1.t1 JOIN db2.t2 ON t1.c1 = t2.c1;",
			expected: []base.SchemaResource{
				{
					Database: "db1",
					Table:    "t1",
				},
				{
					Database: "db2",
					Table:    "t2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		resources, err := ExtractResourceList("db", "", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}

func TestExtractMySQLChangedResources(t *testing.T) {
	tests := []struct {
		statement string
		expected  []base.SchemaResource
	}{
		{
			statement: "CREATE TABLE t1 (c1 INT);",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "DROP TABLE t1;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "ALTER TABLE t1 ADD COLUMN c1 INT;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
			},
		},
		{
			statement: "RENAME TABLE t1 TO t2;",
			expected: []base.SchemaResource{
				{
					Database: "db",
					Table:    "t1",
				},
				{
					Database: "db",
					Table:    "t2",
				},
			},
		},
	}

	for _, test := range tests {
		resources, err := ExtractChangedResources("db", "", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}

type testData struct {
	sql string
	ans bool
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
		ans := ValidateSQLForEditor(test.sql)
		require.Equal(t, test.ans, ans, test.sql)
	}
}
