package pg

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDeparseData is the test data struct.
type TestDeparseData struct {
	Stmt string `yaml:"stmt"`
	Want string `yaml:"want"`
}

func runDeparseTest(t *testing.T, file string, record bool) {
	var tests []TestDeparseData
	filepath := filepath.Join("test-data", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		nodeList, err := Parse(ParseContext{}, test.Stmt)
		require.NoError(t, err)
		require.Len(t, nodeList, 1)
		res, err := Deparse(DeparseContext{}, nodeList[0])
		require.NoError(t, err)
		if record {
			tests[i].Want = res
		} else {
			require.Equal(t, test.Want, res, test.Stmt)
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

func TestDeparse(t *testing.T) {
	testFileList := []string{
		// Table
		"test_create_table_data.yaml",
		"test_alter_table_data.yaml",
		"test_drop_table_data.yaml",
		// Schema
		"test_create_schema_data.yaml",
		"test_drop_schema_data.yaml",
		"test_rename_schema_data.yaml",
		// Index
		"test_create_index_data.yaml",
		"test_drop_index_data.yaml",
		// Sequence
		"test_create_sequence_data.yaml",
		"test_alter_sequence_data.yaml",
		"test_drop_sequence_data.yaml",
		// Extension
		"test_extension_data.yaml",
		// Function
		"test_function_data.yaml",
		// Trigger
		"test_trigger_data.yaml",
		// Type
		"test_type_data.yaml",
		// Comment
		"test_comment_data.yaml",
	}
	for _, test := range testFileList {
		runDeparseTest(t, test, false /* record */)
	}
}
