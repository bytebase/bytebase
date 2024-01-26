package plsql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type conciseTestCase struct {
	Input string
	Want  string
}

func TestConciseSchema(t *testing.T) {
	tests := []conciseTestCase{}
	const (
		record = false
	)
	var (
		filepath = "test-data/test_concise_schema.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := GetConciseSchema(t.Input)
		a.NoError(err)
		if record {
			tests[i].Want = result
		} else {
			a.Equal(t.Want, result, t.Input)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
