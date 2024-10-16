package pg

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type filterTestCase struct {
	Input  string
	Output string
}

func TestFilter(t *testing.T) {
	const (
		record       = false
		testDataPath = "test-data/filter.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(testDataPath)
	a.NoError(err)

	var testCases []filterTestCase
	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yamlFile.Close())
	a.NoError(yaml.Unmarshal(byteValue, &testCases))

	for i, tc := range testCases {
		result, err := FilterBackupSchema(tc.Input)
		a.NoError(err)
		if record {
			testCases[i].Output = result
		} else {
			a.Equal(tc.Output, result, "Input: %s", tc.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(testCases)
		a.NoError(err)
		err = os.WriteFile(testDataPath, byteValue, 0644)
		a.NoError(err)
	}
}
