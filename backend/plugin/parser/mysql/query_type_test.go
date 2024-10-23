package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQueryType(t *testing.T) {
	type testCase struct {
		Query    string
		Expected base.QueryType
	}

	tests := []testCase{}
	const (
		record = false
	)
	var (
		filepath = "test-data/test_query_type.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := GetQueryType(t.Query)
		a.NoError(err)

		if record {
			tests[i].Expected = result
		} else {
			a.Equal(t.Expected, result, t.Query)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
