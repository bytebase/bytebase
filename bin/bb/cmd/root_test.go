package cmd

import (
	"bytes"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	logger = zap.NewNop()
}

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
	t.Helper()
	for _, tc := range tables {
		actual, err := execute(t, rootCmd, tc.args...)

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
		}
	}
}
