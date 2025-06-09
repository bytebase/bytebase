package pg

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type rollbackCase struct {
	Input  string
	Result []base.BackupStatement
}

func TestBackup(t *testing.T) {
	tests := []rollbackCase{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_backup.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: fixedMockDatabaseMetadataGetter,
			DatabaseName:            "",
		}, t.Input, "", "backupSchema", "rollback")
		a.NoError(err)
		slices.SortFunc(result, func(i, j base.BackupStatement) int {
			if i.TargetTableName == j.TargetTableName {
				if i.Statement < j.Statement {
					return -1
				}
				if i.Statement > j.Statement {
					return 1
				}
				return 0
			}
			if i.TargetTableName < j.TargetTableName {
				return -1
			}
			if i.TargetTableName > j.TargetTableName {
				return 1
			}
			return 0
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
