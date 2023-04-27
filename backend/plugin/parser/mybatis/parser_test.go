// Package mybatis defines the sql extractor for mybatis mapper xml.
package mybatis

import (
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestData is the test data for mybatis parser. It contains the xml and the expected sql.
// And the sql is expected to be restored from the xml.
type TestData struct {
	XML string `yaml:"xml"`
	SQL string `yaml:"sql"`
}

// runTest is a helper function to run the test.
// It will parse the xml given by `filepath` and compare the restored sql with `sql`.
func runTest(t *testing.T, filepath string, record bool) {
	var testCases []TestData
	yamlFile, err := os.Open(filepath)
	assert.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	assert.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &testCases)
	assert.NoError(t, err)

	for i, testCase := range testCases {
		node, err := Parse(testCase.XML)
		assert.NoError(t, err)
		assert.NotNil(t, node)

		var stringsBuilder strings.Builder
		err = node.RestoreSQL(&stringsBuilder)
		assert.NoError(t, err)
		if record {
			testCases[i].SQL = stringsBuilder.String()
		} else {
			assert.Equal(t, testCase.SQL, stringsBuilder.String())
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err = yaml.Marshal(testCases)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

func TestParser(t *testing.T) {
	testFileList := []string{
		"test-data/test_simple_mapper.yaml",
	}
	for _, filepath := range testFileList {
		runTest(t, filepath, true)
	}
}
