package cmd

import (
	"fmt"
	"os"
	"testing"

	// Embeded expected output.
	_ "embed"

	"github.com/bytebase/bytebase/resources/mysql"
)

var (
	//go:embed testdata/expected/test_dump_todo
	expectedTodo string
)

func TestDump(t *testing.T) {
	basedir, datadir := t.TempDir(), t.TempDir()
	t.Log("Installing MySQL...")
	mysql, err := mysql.Install(basedir, datadir, "root")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Starting MySQL...")
	if err := mysql.Start(13306, os.Stdout, os.Stderr, 60); err != nil {
		t.Fatal(err)
	}
	defer func() {
		t.Log("Stopping MySQL...")
		if err := mysql.Stop(os.Stdout, os.Stderr); err != nil {
			t.Fatal(err)
		}
	}()

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
