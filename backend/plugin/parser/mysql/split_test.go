package mysql

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestMySQLSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}

func TestMySQLSplitSQLLargeInsertScriptScalesLinearly(t *testing.T) {
	const rowCount = 2000
	padding := strings.Repeat("x", 1024)
	var builder strings.Builder
	for i := 0; i < rowCount; i++ {
		fmt.Fprintf(&builder, "INSERT INTO perf_omni_mysql (id, payload) VALUES (%d, '%s');\n", i, padding)
	}

	started := time.Now()
	statements, err := SplitSQL(builder.String())
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.Len(t, base.FilterEmptyStatements(statements), rowCount)
	require.Less(t, elapsed, time.Second)
}

func TestSplitMySQLStatements(t *testing.T) {
	tests := []struct {
		statement string
		expected  []string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []string{
				"SELECT * FROM t1 WHERE c1 = 1;",
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END; SELECT * FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
			  SELECT name INTO name FROM users WHERE id = id;
			END;`,
				" SELECT * FROM t2;",
			},
		},
		{
			statement: `CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END; SELECT REPEAT('123', a) FROM t2;`,
			expected: []string{
				`CREATE PROCEDURE my_procedure (IN id INT, OUT name VARCHAR(255))
			BEGIN
				SELECT IF(id = 1, 'one', 'other') INTO name FROM users;
			END;`,
				" SELECT REPEAT('123', a) FROM t2;",
			},
		},
	}

	for _, test := range tests {
		list, err := SplitSQL(test.statement)
		require.NoError(t, err)
		require.Equal(t, len(test.expected), len(list))
		for i, statement := range list {
			require.Equal(t, test.expected[i], statement.Text)
		}
	}
}
