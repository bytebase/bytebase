package mariadb

import (
	"context"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
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
		slices.SortFunc(result, func(x, y base.BackupStatement) int {
			if x.TargetTableName == y.TargetTableName {
				if x.Statement < y.Statement {
					return -1
				} else if x.Statement > y.Statement {
					return 1
				}
				return 0
			}
			if x.TargetTableName < y.TargetTableName {
				return -1
			} else if x.TargetTableName > y.TargetTableName {
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

func TestMariaDBTransformDMLToSelectRegistration(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()

	result, err := base.TransformDMLToSelect(context.Background(), store.Engine_MARIADB, base.TransformContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE b1 = 1;", "db", "backupDB", "_rollback")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "test", result[0].SourceTableName)
	require.Equal(t, "_rollback_test_db", result[0].TargetTableName)
	require.Contains(t, result[0].Statement, "CREATE TABLE `backupDB`.`_rollback_test_db` LIKE `db`.`test`;")
}

func buildFixedMockDatabaseMetadataGetterAndLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	schemaMetadata := []*metadatapb.SchemaMetadata{
		{
			Name: "",
			Tables: []*metadatapb.TableMetadata{
				{
					Name: "t_generated",
					Columns: []*metadatapb.ColumnMetadata{
						{
							Name: "a",
						},
						{
							Name: "b",
						},
						{
							Name: "c_generated",
							Generation: &metadatapb.GenerationMetadata{
								Expression: "a + b",
							},
						},
					},
					Indexes: []*metadatapb.IndexMetadata{
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
					},
				},
				{
					Name: "t1",
					Columns: []*metadatapb.ColumnMetadata{
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
					Columns: []*metadatapb.ColumnMetadata{
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
					Columns: []*metadatapb.ColumnMetadata{
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
					Indexes: []*metadatapb.IndexMetadata{
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
					Columns: []*metadatapb.ColumnMetadata{
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
					Columns: []*metadatapb.ColumnMetadata{
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
			return database, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				Name:    database,
				Schemas: schemaMetadata,
			}, nil, nil, store.Engine_MARIADB, true /* isObjectCaseSensitive */), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db", "db1", "db2"}, nil
		}
}

// TestTransformDMLToSelectStripsReturning covers a clause MySQL has no grammar
// for, so the transform this was forked from never had to drop it. RETURNING is
// the last clause of a single-table MariaDB DELETE and means nothing in a
// SELECT: copying through to the statement end emitted
// `SELECT `test`.* FROM test WHERE b1 = 1 RETURNING b1`, which the server
// rejects when the backup runs — after the change had already passed review.
func TestTransformDMLToSelectStripsReturning(t *testing.T) {
	for _, statement := range []string{
		"DELETE FROM test WHERE b1 = 1 RETURNING b1;",
		"DELETE FROM test WHERE b1 = 1 RETURNING b1, c1;",
		"DELETE FROM test WHERE b1 = 1 RETURNING *;",
		"DELETE FROM test WHERE b1 = 1 ORDER BY b1 LIMIT 1 RETURNING b1;",
		// The keyword is found by token, not by text: a comment or a quoted
		// string can carry the word and must not be mistaken for the clause.
		"DELETE FROM test WHERE b1 = 1 RETURNING /* RETURNING */ b1;",
		"DELETE FROM test WHERE b1 = 1 RETURNING\n-- RETURNING\n b1;",
		"DELETE FROM test WHERE b1 = 1 RETURNING # RETURNING\n b1;",
	} {
		getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
		result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
			IsCaseSensitive:         false,
		}, statement, "db", "backupDB", "_rollback")
		require.NoError(t, err, statement)
		require.NotEmpty(t, result, statement)
		for _, bs := range result {
			require.NotContains(t, strings.ToUpper(bs.Statement), "RETURNING", statement)
			require.Contains(t, bs.Statement, "WHERE b1 = 1", statement)
		}
	}
}

// TestTransformDMLToSelectKeepsTheRestOfTheSuffix guards the cut from taking the
// clauses before RETURNING with it.
func TestTransformDMLToSelectKeepsTheRestOfTheSuffix(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE b1 = 1 ORDER BY b1 LIMIT 1 RETURNING b1;", "db", "backupDB", "_rollback")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Contains(t, result[0].Statement, "ORDER BY b1")
	require.Contains(t, result[0].Statement, "LIMIT 1")
}

// TestTransformDMLToSelectKeepsAQuotedReturning holds the other half of locating
// the clause by token: the word inside a string literal belongs to the WHERE and
// must stay, or the backup would copy the wrong rows.
func TestTransformDMLToSelectKeepsAQuotedReturning(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	result, err := TransformDMLToSelect(context.Background(), base.TransformContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, "DELETE FROM test WHERE v1 = 'RETURNING' RETURNING b1;", "db", "backupDB", "_rollback")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Contains(t, result[0].Statement, "WHERE v1 = 'RETURNING'")
	require.NotContains(t, strings.ToUpper(result[0].Statement), "RETURNING B1")
}
