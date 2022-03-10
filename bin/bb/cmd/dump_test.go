package cmd

import (
	"fmt"
	"os"
	"testing"

	// Embedded expected output.
	_ "embed"

	"github.com/bytebase/bytebase/resources/mysql"
)

var (
	//go:embed testdata/expected/test_dump_todo
	expectedTodo string
)

func TestDump(t *testing.T) {
	mysql, stop := mysql.SetupTestInstance(t, 13306)
	defer stop()

	t.Log("Importing MySQL data...")
	if err := mysql.Import("testdata/mysql_test_schema", os.Stdout, os.Stderr); err != nil {
		t.Fatal(err)
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
			expected: expectedTodo,
		},
	}
	tableTest(t, tt)
}
