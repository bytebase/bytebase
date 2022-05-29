package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/bytebase/bytebase/common/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	PortTestDump = iota + 13306
	PortTestMigrate
	PortTestCreateDatabase
)

func execute(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return buf.String(), err
}

type testTable struct {
	args        []string
	expectedErr error
	expected    string
}

func tableTest(t *testing.T, tables []testTable) {
	log.Init()
	log.SetLevel(zap.DebugLevel)
	t.Helper()
	for _, tc := range tables {
		actual, err := execute(t, NewRootCmd(), tc.args...)

		if err != tc.expectedErr {
			t.Errorf(`"> bb %v"
got unexpected error:
%s
`, strings.Join(tc.args, " "), cmp.Diff(err, tc.expectedErr))
		}

		if actual != tc.expected {
			t.Errorf(`"> bb %v"
got unexpected output:
%s
`, strings.Join(tc.args, " "), cmp.Diff(actual, tc.expected))

			f, err := os.CreateTemp("", "*")
			if err != nil {
				t.Log(err)
			}
			defer f.Close()
			if _, err := io.WriteString(f, actual); err != nil {
				t.Log(err)
			}
			t.Logf("Actual output written to %s", f.Name())
		}
	}
}
