//go:build mysql
// +build mysql

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
	//go:embed testdata/expected/dump_test_TestDump
	_TestDump string
)

func TestDump(t *testing.T) {
	mysql, stop := mysql.SetupTestInstance(t, PortTestDump)
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
			expected: _TestDump,
		},
	}
	tableTest(t, tt)
}
