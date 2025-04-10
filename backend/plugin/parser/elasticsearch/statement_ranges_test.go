package elasticsearch

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementRanges(t *testing.T) {
	type testCase struct {
		Description string       `yaml:"description,omitempty"`
		Statement   string       `yaml:"statement,omitempty"`
		Result      []base.Range `yaml:"result,omitempty"`
	}

	var (
		filepath = "test-data/statement-ranges.yaml"
		record   = false
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())

	var testCases []testCase
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		got, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, tc.Statement)
		a.NoErrorf(err, "description: %s", tc.Description)
		if record {
			testCases[i].Result = got
		} else {
			a.Equalf(tc.Result, got, "description: %s", tc.Description)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
