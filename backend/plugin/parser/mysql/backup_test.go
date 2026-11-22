package mysql

import (
	"context"
	"io"
	"os"
	"slices"
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
	// Error is the refusal, when prior backup cannot back the input up.
	Error string `yaml:",omitempty"`
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
		errText := ""
		if err != nil {
			errText = err.Error()
		}
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
			tests[i].Error = errText
		} else {
			a.Equal(t.Error, errText, t.Input)
			if errText == "" {
				a.Equal(t.Result, result, t.Input)
			}
		}
	}
	if record {
		yamltest.Record(t, filepath, tests)
	}
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
					// Prior backup needs a row key to back up a table a join reads.
					Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}}},
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
					// Prior backup needs a row key to back up a table a join reads.
					Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}}},
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
					// Prior backup needs a row key to back up a table a join reads.
					Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}}},
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
					// Prior backup needs a row key to back up a table a join reads.
					Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}}},
				},
			},
		},
	}
	return func(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
			return database, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				Name:    database,
				Schemas: schemaMetadata,
			}, nil, nil, store.Engine_MYSQL, true /* isObjectCaseSensitive */), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db", "db1", "db2"}, nil
		}
}

func TestBackupReadsReferencingDatabases(t *testing.T) {
	keyed := func(name string, foreignKeys ...*metadatapb.ForeignKeyMetadata) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{
			Name:        name,
			Columns:     []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "parent_id"}},
			Indexes:     []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"id"}}},
			ForeignKeys: foreignKeys,
		}
	}
	tables := map[string][]*metadatapb.TableMetadata{
		"db":  {keyed("parent"), keyed("other")},
		"app": {keyed("child", &metadatapb.ForeignKeyMetadata{Name: "fk_remote", Columns: []string{"parent_id"}, ReferencedSchema: "db", ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "CASCADE"})},
	}
	var getter base.GetDatabaseMetadataFunc = func(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
		return database, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    database,
			Schemas: []*metadatapb.SchemaMetadata{{Tables: tables[database]}},
		}, nil, nil, store.Engine_MYSQL, false), nil
	}
	lister := func(context.Context, string) ([]string, error) { return []string{"db", "app"}, nil }
	transform := func(statement string, referencing base.ListDatabaseNamesFunc) error {
		_, err := base.TransformDMLToSelect(context.Background(), store.Engine_MYSQL, base.TransformContext{
			GetDatabaseMetadataFunc:      getter,
			ListDatabaseNamesFunc:        lister,
			ListReferencingDatabasesFunc: referencing,
		}, statement, "db", "backupDB", "_rollback")
		return err
	}
	const cascade = "foreign key fk_remote of app.child is ON DELETE CASCADE"

	// Without the hook, every database is read.
	require.ErrorContains(t, transform("DELETE FROM parent WHERE id = 1", nil), cascade)
	require.ErrorContains(t, transform("DELETE FROM parent WHERE id = 1", func(context.Context, string) ([]string, error) {
		return []string{"app"}, nil
	}), cascade)
	// The hook narrows the databases read, and is called once per backup.
	calls := 0
	require.NoError(t, transform("DELETE FROM parent WHERE id = 1; DELETE FROM other WHERE id = 1", func(context.Context, string) ([]string, error) {
		calls++
		return nil, nil
	}))
	require.Equal(t, 1, calls)
}
