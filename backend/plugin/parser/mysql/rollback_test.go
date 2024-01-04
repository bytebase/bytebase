package mysql

import (
	"io"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type rollbackCase struct {
	Input  string
	Result []base.RollbackStatement
}

func TestRollback(t *testing.T) {
	tests := []rollbackCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_rollback.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := TransformDMLToSelect(t.Input, "db", "backupDB", "_rollback")
		a.NoError(err)
		sort.Slice(result, func(i, j int) bool {
			if result[i].TableName == result[j].TableName {
				return result[i].Statement < result[j].Statement
			}
			return result[i].TableName < result[j].TableName
		})

		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Input)
		}
	}
	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}
