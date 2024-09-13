package tidb

import (
	"context"
	"io"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/proto/generated-go/store"
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
		getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IgnoreCaseSensitive:     true,
		}, t.Input, "db", "backupDB", "_rollback")
		a.NoError(err)
		sort.Slice(result, func(i, j int) bool {
			if result[i].TargetTableName == result[j].TargetTableName {
				return result[i].Statement < result[j].Statement
			}
			return result[i].TargetTableName < result[j].TargetTableName
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

func buildFixedMockDatabaseMetadataGetterAndLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	schemaMetadata := []*store.SchemaMetadata{
		{
			Name: "",
			Tables: []*store.TableMetadata{
				{
					Name: "t_generated",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c_generated",
							Generation: &store.GenerationMetadata{
								Expression: "a + b",
							},
						},
					},
				},
				{
					Name: "t1",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "t2",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "test",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
				{
					Name: "test2",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c",
						},
					},
				},
			},
		},
	}

	return func(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
			return database, model.NewDatabaseMetadata(&store.DatabaseSchemaMetadata{
				Name:    database,
				Schemas: schemaMetadata,
			}), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db", "db1", "db2"}, nil
		}
}
