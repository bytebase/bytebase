package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type testCase struct {
	old  string
	new  string
	want string
}

func testDiffWithoutDisableForeignKeyCheck(t *testing.T, testCases []testCase) {
	a := require.New(t)
	for _, test := range testCases {
		out, err := SchemaDiff(base.DiffContext{
			IgnoreCaseSensitive: true,
			StrictMode:          true,
		}, test.old, test.new)
		a.NoError(err)
		a.Equalf(test.want, out, "old: %s\nnew: %s\n", test.old, test.new)
	}
}
