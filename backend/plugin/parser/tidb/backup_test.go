package tidb

import (
	"context"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
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
			IsCaseSensitive:         false,
		}, t.Input, "db", "backupDB", "_rollback")
		a.NoError(err)
		slices.SortFunc(result, func(a, b base.BackupStatement) int {
			if a.TargetTableName == b.TargetTableName {
				if a.Statement < b.Statement {
					return -1
				}
				if a.Statement > b.Statement {
					return 1
				}
				return 0
			}
			if a.TargetTableName < b.TargetTableName {
				return -1
			}
			if a.TargetTableName > b.TargetTableName {
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
		yamltest.Record(t, filepath, tests)
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
					Indexes: []*store.IndexMetadata{
						{
							Name:    "PRIMARY",
							Primary: true,
							Unique:  true,
							Expressions: []string{
								"b",
							},
						},
						{
							Name:   "uk_a",
							Unique: true,
							Expressions: []string{
								"a",
							},
						},
						// Unique key on a generated column (c_generated = a + b).
						// Used by TestGenerateRestoreSQLGeneratedColumnUKSkipped to
						// pin that hasDisjointUniqueKey skips UKs whose
						// expressions reference generated columns. Pre-fix this
						// UK would false-positive as disjoint via naive string
						// comparison; post-fix it's correctly skipped.
						{
							Name:   "uk_c_generated",
							Unique: true,
							Expressions: []string{
								"c_generated",
							},
						},
						// Unique key with empty Expressions — represents the
						// TiDB-metadata shape for some expression/functional
						// index parts that don't populate key.Column (per
						// backend/plugin/schema/tidb/get_database_metadata.go).
						// Used by TestGenerateRestoreSQLEmptyExpressionsUKSkipped
						// to pin that hasDisjointUniqueKey skips empty-
						// Expressions UKs. Pre-fix: disjoint([]) returns
						// vacuously true, false-positive disjoint. Post-fix:
						// empty-Expressions UKs are skipped explicitly.
						{
							Name:        "uk_empty_expressions",
							Unique:      true,
							Expressions: nil,
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
					Indexes: []*store.IndexMetadata{
						{
							Name:    "PRIMARY",
							Primary: true,
							Expressions: []string{
								"c",
							},
						},
						{
							Name:   "PRIMARY",
							Unique: true,
							Expressions: []string{
								"a",
							},
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
				{
					Name: "t3",
					Columns: []*store.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "d",
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
			}, nil, nil, store.Engine_TIDB, false /* isObjectCaseSensitive */), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db", "db1", "db2"}, nil
		}
}
