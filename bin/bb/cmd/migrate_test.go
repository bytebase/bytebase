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
	//go:embed testdata/expected/migrate_test_TestMigrate_01
	_TestMigrate01 string
	//go:embed testdata/expected/migrate_test_TestMigrate_02
	_TestMigrate02 string
	//go:embed testdata/expected/migrate_test_TestMigrate_03
	_TestMigrate03 string
)

func TestMigrate(t *testing.T) {
	mysql, stop := mysql.SetupTestInstance(t, PortTestMigrate)
	defer stop()

	if err := mysql.Import("testdata/mysql_test_schema/1_todo.sql", os.Stdout, os.Stderr); err != nil {
		t.Errorf("error importing data: %v", err)
	}

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
			expected: _TestMigrate01,
		},
		{
			args: []string{
				"migrate",
				"--type", "mysql",
				"--username", "root",
				"--host", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"-c", `"
	CREATE TABLE bytebase_test_todo.book (
		id INTEGER PRIMARY KEY,
		name TEXT NULL
	);"`,
				"--database", "bytebase_test_todo",
			},
			expected: _TestMigrate02,
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
			expected: _TestMigrate03,
		},
	}
	tableTest(t, tt)
}

var (
	//go:embed testdata/expected/migrate_test_TestCreateDatabase_01
	_TestCreateDatabase01 string
	//go:embed testdata/expected/migrate_test_TestCreateDatabase_02
	_TestCreateDatabase02 string
)

func TestCreateDatabase(t *testing.T) {
	mysql, stop := mysql.SetupTestInstance(t, PortTestCreateDatabase)
	defer stop()

	tt := []testTable{
		{
			args: []string{
				"migrate",
				"--type", "mysql",
				"--username", "root",
				"--host", "localhost",
				"--port", fmt.Sprint(mysql.Port()),
				"-f", "testdata/mysql_test_schema/1_todo.sql",
			},
			expected: _TestCreateDatabase01,
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
			expected: _TestCreateDatabase02,
		},
	}
	tableTest(t, tt)
}
