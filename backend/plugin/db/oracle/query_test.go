package oracle

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetStatementWithResultLimit(t *testing.T) {
	runLimitTest(t, "test_limit.yaml", false /* record */)
}

type limitTestData struct {
	Stmt  string `yaml:"stmt"`
	Limit int    `yaml:"limit"`
	Want  string `yaml:"want"`
}

func runLimitTest(t *testing.T, file string, record bool) {
	var testCases []limitTestData
	filepath := filepath.Join("test-data", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &testCases)
	require.NoError(t, err)

	for i, tc := range testCases {
		want := addLimitFor12cAndLater(tc.Stmt, tc.Limit)
		if record {
			testCases[i].Want = want
		} else {
			require.Equal(t, tc.Want, want, tc.Stmt)
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
