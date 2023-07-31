package pg

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

type DifferTestData struct {
	OldSchema string `yaml:"oldSchema"`
	NewSchema string `yaml:"newSchema"`
	Diff      string `yaml:"diff"`
}

func runDifferTest(t *testing.T, file string, record bool) {
	pgDiffer := &SchemaDiffer{}

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
		diff, err := pgDiffer.SchemaDiff(test.OldSchema, test.NewSchema, false /* ignoreCaseSensitive */)
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

func TestComputeDiff(t *testing.T) {
	testFileList := []string{
		"test_differ_data.yaml",
		// Schema
		"test_differ_schema.yaml",
		// Constraint
		"test_differ_constraint.yaml",
		// Merge
		"test_differ_merge.yaml",
		// Sequence
		"test_differ_sequence.yaml",
	}
	for _, test := range testFileList {
		runDifferTest(t, test, false /* record */)
	}
}
