package cockroachdb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSplitSQLStatement(t *testing.T) {
	type testCase struct {
		Statement string   `yaml:"statement"`
		Want      []string `yaml:"want"`
	}

	const record = false

	a := require.New(t)

	testFilepath := "testdata/split_sql_statement.yaml"
	content, err := os.ReadFile(testFilepath)
	a.NoError(err)

	var testCases []testCase
	err = yaml.Unmarshal(content, &testCases)
	a.NoError(err)

	for idx, tc := range testCases {
		got, err := SplitSQLStatement(tc.Statement)
		a.NoErrorf(err, "test case %d: %s", idx, tc.Statement)
		if record {
			testCases[idx].Want = got
		} else {
			a.Equalf(tc.Want, got, "test case %d: %s", idx, tc.Statement)
		}
	}

	if record {
		content, err = yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testFilepath, content, 0644)
		a.NoError(err)
	}
}
