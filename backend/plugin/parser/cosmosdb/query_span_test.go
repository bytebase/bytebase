package cosmosdb

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQuerySpan(t *testing.T) {
	type testCase struct {
		Description string              `yaml:"description,omitempty"`
		Statement   string              `yaml:"statement,omitempty"`
		QuerySpan   *base.YamlQuerySpan `yaml:"querySpan,omitempty"`
	}

	var (
		record        = true
		testDataPaths = []string{
			"test-data/query-span/standard.yaml",
		}
	)

	a := require.New(t)
	for _, testDataPath := range testDataPaths {
		testDataPath := testDataPath

		yamlFile, err := os.Open(testDataPath)
		a.NoError(err)

		var testCases []testCase
		byteValue, err := io.ReadAll(yamlFile)
		a.NoError(err)
		a.NoError(yamlFile.Close())
		a.NoError(yaml.Unmarshal(byteValue, &testCases))

		for i, tc := range testCases {
			result, err := GetQuerySpan(context.TODO(), base.GetQuerySpanContext{}, base.Statement{Text: tc.Statement}, "", "", false)
			a.NoErrorf(err, "statement: %s", tc.Statement)
			resultYaml := result.ToYaml()
			if record {
				testCases[i].QuerySpan = resultYaml
			} else {
				a.Equalf(tc.QuerySpan, resultYaml, "statement: %s", tc.Statement)
			}
		}

		if record {
			byteValue, err := yaml.Marshal(testCases)
			a.NoError(err)
			err = os.WriteFile(testDataPath, byteValue, 0644)
			a.NoError(err)
		}
	}
}
