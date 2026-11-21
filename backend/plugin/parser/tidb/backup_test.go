package tidb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func buildFixedMockDatabaseMetadataGetterAndLister() (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	schemaMetadata := []*metadatapb.SchemaMetadata{{
		Name: "",
		Tables: []*metadatapb.TableMetadata{{
			Name:    "test",
			Columns: []*metadatapb.ColumnMetadata{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			Indexes: []*metadatapb.IndexMetadata{
				{Name: "PRIMARY", Primary: true, Expressions: []string{"c"}},
				{Name: "uk_a", Unique: true, Expressions: []string{"a"}},
			},
		}},
	}}
	return func(_ context.Context, _ string, database string) (string, *model.DatabaseMetadata, error) {
			return database, model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
				Name:    database,
				Schemas: schemaMetadata,
			}, nil, nil, store.Engine_TIDB, false /* isObjectCaseSensitive */), nil
		}, func(_ context.Context, _ string) ([]string, error) {
			return []string{"db"}, nil
		}
}

func TestTransformDMLToSelectSeam(t *testing.T) {
	getter, lister := buildFixedMockDatabaseMetadataGetterAndLister()
	stmt := "SELECT 1;\nUPDATE test SET b = 2 WHERE c = 1;"

	result, err := base.TransformDMLToSelect(context.Background(), store.Engine_TIDB, base.TransformContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		IsCaseSensitive:         false,
	}, stmt, "db", "backupDB", "_rollback")
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "test", result[0].SourceTableName)
	require.Equal(t, "_rollback_1_test", result[0].TargetTableName)
	require.Equal(t, &store.Position{Line: 1, Column: 10}, result[0].StartPosition)
	require.Equal(t, &store.Position{Line: 2, Column: 35}, result[0].EndPosition)
	require.Contains(t, result[0].Statement, "CREATE TABLE `backupDB`.`_rollback_1_test` LIKE `db`.`test`;")

	_, err = TransformDMLToSelect(context.Background(), base.TransformContext{}, "DELETE FROM test WHERE c = 1;", "db", "backupDB", "_rollback")
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is not set")
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
		}, nil, nil, store.Engine_TIDB, false), nil
	}
	lister := func(context.Context, string) ([]string, error) { return []string{"db", "app"}, nil }
	transform := func(statement string, referencing base.ListDatabaseNamesFunc) error {
		_, err := base.TransformDMLToSelect(context.Background(), store.Engine_TIDB, base.TransformContext{
			GetDatabaseMetadataFunc:      getter,
			ListDatabaseNamesFunc:        lister,
			ListReferencingDatabasesFunc: referencing,
		}, statement, "db", "backupDB", "_rollback")
		return err
	}
	const cascade = "foreign key fk_remote of app.child is ON DELETE CASCADE"

	require.ErrorContains(t, transform("DELETE FROM parent WHERE id = 1", nil), cascade)
	require.ErrorContains(t, transform("DELETE FROM parent WHERE id = 1", func(context.Context, string) ([]string, error) {
		return []string{"app"}, nil
	}), cascade)
	calls := 0
	require.NoError(t, transform("DELETE FROM parent WHERE id = 1; DELETE FROM other WHERE id = 1", func(context.Context, string) ([]string, error) {
		calls++
		return nil, nil
	}))
	require.Equal(t, 1, calls)
}
