package doris

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRange(t *testing.T) {
	type testCase struct {
		Statement string       `yaml:"statement,omitempty"`
		Expected  []base.Range `yaml:"ranges,omitempty"`
	}

	const (
		record      = false
		testDataDir = "test-data/statement-ranges"
	)
	a := require.New(t)

	// Recursively find all YAML files in the testDataDir
	entries, err := os.ReadDir(testDataDir)
	a.NoError(err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filepath := path.Join(testDataDir, entry.Name())
		yamlFile, err := os.Open(filepath)
		a.NoError(err)
		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))
		for i, tc := range testCases {
			if tc.Statement == "" {
				continue
			}
			ranges, err := GetStatementRanges(context.TODO(), base.StatementRangeContext{}, tc.Statement)
			a.NoError(err)
			if record {
				testCases[i].Expected = ranges
			} else {
				a.Equal(tc.Expected, ranges, "statement: %s", tc.Statement)
			}
		}

		if record {
			yamlData, err := yaml.Marshal(testCases)
			a.NoError(err)
			err = os.WriteFile(filepath, yamlData, 0644)
			a.NoError(err)
		}
	}
}
