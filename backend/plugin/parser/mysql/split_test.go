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

func TestMySQLSplitSQLDelimiterDirectives(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND//\nDELIMITER ;\nCALL p();"

	list, err := SplitSQL(statement)
	require.NoError(t, err)

	list = base.FilterEmptyStatements(list)
	require.Len(t, list, 2)
	require.Equal(t, "CREATE PROCEDURE p()\nBEGIN\n  SELECT delimiter FROM t;\nEND", list[0].Text)
	require.Equal(t, "CALL p();", list[1].Text)
	require.Equal(t, statement[list[0].Range.Start:list[0].Range.End], list[0].Text)
	require.Equal(t, statement[list[1].Range.Start:list[1].Range.End], list[1].Text)
}

func TestMySQLSplitSQLDelimiterLabelIsNotDirective(t *testing.T) {
	statement := "DELIMITER: LOOP\n  SELECT 1;\n  LEAVE DELIMITER;\nEND LOOP DELIMITER;"

	list, err := SplitSQL(statement)
	require.NoError(t, err)

	list = base.FilterEmptyStatements(list)
	require.Len(t, list, 1)
	require.Equal(t, statement, list[0].Text)
}

func TestMySQLSplitSQLDelimiterIdentifierIsNotDirective(t *testing.T) {
	statement := "CREATE TABLE t (\n  delimiter INT\n);\nSELECT delimiter FROM t;"

	list, err := SplitSQL(statement)
	require.NoError(t, err)

	list = base.FilterEmptyStatements(list)
	require.Len(t, list, 2)
	require.Equal(t, "CREATE TABLE t (\n  delimiter INT\n);", list[0].Text)
	require.Equal(t, "\nSELECT delimiter FROM t;", list[1].Text)
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
