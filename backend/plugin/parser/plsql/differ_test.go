// Package plsql provides the plsql parser plugin.
package plsql

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type DifferTestData struct {
	OldSchema string `yaml:"oldSchema"`
	NewSchema string `yaml:"newSchema"`
	Diff      string `yaml:"diff"`
}

func runDifferTest(t *testing.T, file string, record bool, strict bool) {
	var tests []DifferTestData
	filepath := filepath.Join("test-data", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		diff, err := SchemaDiff(base.DiffContext{
			IgnoreCaseSensitive: false,
			StrictMode:          strict,
		}, test.OldSchema, test.NewSchema)
		require.NoError(t, err)
		if record {
			tests[i].Diff = diff
		} else {
			require.Equal(t, test.Diff, diff, test.OldSchema)
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err = yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

func TestPLSQLDiffer(t *testing.T) {
	testFileList := []string{
		"test_differ_data.yaml",
	}
	for _, file := range testFileList {
		runDifferTest(t, file, false /* record */, true /* strict */)
	}
}

func TestPlSQLDifferNonStrict(t *testing.T) {
	testFileList := []string{
		"test_differ_non_strict.yaml",
	}
	for _, file := range testFileList {
		runDifferTest(t, file, false /* record */, false /* strict */)
	}
}
