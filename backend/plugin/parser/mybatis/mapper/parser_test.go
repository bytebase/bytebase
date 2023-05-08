// Package mapper defines the sql extractor for mybatis mapper xml.
package mapper

import (
	"io"
	"os"
	"strings"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &testCases)
	require.NoError(t, err)

	for i, testCase := range testCases {
		parser := NewParser(testCase.XML)
		node, err := parser.Parse()
		require.NoError(t, err)
		require.NotNil(t, node)

		var stringsBuilder strings.Builder
		err = node.RestoreSQL(parser.GetRestoreContext(), &stringsBuilder)
		require.NoError(t, err)
		require.NoError(t, err)
		if record {
			testCases[i].SQL = stringsBuilder.String()
		} else {
			require.Equal(t, testCase.SQL, stringsBuilder.String())
		}
		// The result should be parsed correctly by MySQL parser.
		_, err = pg_query.Parse(testCases[i].SQL)
		require.NoError(t, err, "failed to parse restored sql: %s", testCases[i].SQL)
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
		"test-data/test_dynamic_sql_mapper.yaml",
	}
	for _, filepath := range testFileList {
		runTest(t, filepath, false)
	}
}
