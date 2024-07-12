package mysql

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
	Diff      string `yal:"diff"`
}

func runDifferTest(t *testing.T, file string, record bool) {
	var tests []DifferTestData
	filepath := filepath.Join("test-data/differ", file)
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		diff, err := SchemaDiff(base.DiffContext{IgnoreCaseSensitive: false, StrictMode: true}, test.OldSchema, test.NewSchema)
		require.NoErrorf(t, err, "Test Cases[%02d] Failed", i+1)
		if record {
			tests[i].Diff = diff
		} else {
			require.Equalf(t, test.Diff, diff, "Test Cases[%02d] Failed", i+1)
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

func TestSchemaDiffTable(t *testing.T) {
	testFile := "test_differ_table.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffColumn(t *testing.T) {
	testFile := "test_differ_column.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffIndex(t *testing.T) {
	testFile := "test_differ_index.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffView(t *testing.T) {
	testFile := "test_differ_view.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffFunction(t *testing.T) {
	testFile := "test_differ_function.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffProcedure(t *testing.T) {
	testFile := "test_differ_procedure.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffEvent(t *testing.T) {
	testFile := "test_differ_event.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffTrigger(t *testing.T) {
	testFile := "test_differ_trigger.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffConstraint(t *testing.T) {
	testFile := "test_differ_constraint.yaml"
	runDifferTest(t, testFile, false /* record */)
}

func TestSchemaDiffPartition(t *testing.T) {
	testFile := "test_differ_partition.yaml"
	runDifferTest(t, testFile, false /* record */)
}
