package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		statement string
		validate  bool
		err       bool
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
		{
			statement: `select* from t`,
			validate:  true,
		},
		{
			statement: `explain select * from t;`,
			validate:  true,
		},
		{
			statement: `explain    analyze select * from t`,
			validate:  true,
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
			validate: false,
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
			validate: false,
			err:      true,
		},
		{
			statement: "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			validate:  true,
		},
		{
			statement: `
				With t as (
					select * from t1 where a = 'insert'
				), tx as (` +
				"   select * from `delete`" +
				`) /* UPDATE */` +
				"select `update` from t;",
			validate: true,
		},
		{
			statement: `create table t (a int);`,
			validate:  false,
		},
	}

	for _, test := range tests {
		got, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.validate, got)
		}
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
		resources, err := extractChangedResources("db", "", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
