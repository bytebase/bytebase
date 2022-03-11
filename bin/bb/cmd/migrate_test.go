package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/bytebase/bytebase/resources/mysql"

	// embed expected output
	_ "embed"
)

var (
	expectedMigrate string
	//go:embed testdata/expected/test_migrate_after
	expectedDumpAfterMigrate string
)

func TestMigrate(t *testing.T) {
	mysql, stop := mysql.SetupTestInstance(t, 23306)
	defer stop()

	mysql.Import("testdata/mysql_test_schema/1_todo.sql", os.Stdout, os.Stderr)
	// mysql.Import("testdata/mysql_test_schema/2_blog.sql", os.Stdout, os.Stderr)

	tt := []testTable{
		{
			args: []string{
				"dump",
				"--type", "mysql",
				"--username", "root",
				"--hostname", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"--database", "bytebase_test_todo",
			},
			expected: expectedTodo,
		},
		{
			args: []string{
				"migrate",
				"--type", "mysql",
				"--username", "root",
				"--hostname", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"--sql", "testdata/mysql_test_schema/2_blog.sql",
				"--database", "bytebase_test_blog",
				"--create-database",
			},
			expected: expectedMigrate,
		},
		{
			args: []string{
				"dump",
				"--type", "mysql",
				"--username", "root",
				"--hostname", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"--database", "bytebase_test_todo",
			},
			expected: expectedTodo,
		},
		{
			args: []string{
				"dump",
				"--type", "mysql",
				"--username", "root",
				"--hostname", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"--database", "bytebase_test_blog",
			},
			expected: expectedDumpAfterMigrate,
		},
	}
	tableTest(t, tt)
}
