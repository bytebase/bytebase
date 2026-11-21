package tsql

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestGenerateRestoreSQLSeam(t *testing.T) {
	dbMeta := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "name", Type: "nvarchar"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "pk_users", Primary: true, Expressions: []string{"id"}},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_MSSQL, false)

	script := "UPDATE dbo.other SET x = 1;\nUPDATE dbo.users SET name = 'bob' WHERE id = 1;"
	backups, err := base.TransformDMLToSelect(context.Background(), storepb.Engine_MSSQL, base.TransformContext{}, script, "app", "backup_db", "bb_")
	require.NoError(t, err)
	require.Len(t, backups, 2)
	usersBackup := backups[1]

	// Empty schema defaults to "dbo", primary key is plumbed, and inRange selects only the 2nd statement.
	item := &storepb.PriorBackupDetail_Item{
		SourceTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "instances/inst-1/databases/app",
			Schema:   "",
			Table:    "users",
		},
		TargetTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "instances/inst-1/databases/backup_db",
			Table:    usersBackup.TargetTableName,
		},
		StartPosition: usersBackup.StartPosition,
		EndPosition:   usersBackup.EndPosition,
	}

	rCtx := base.RestoreContext{
		InstanceID: "inst-1",
		GetDatabaseMetadataFunc: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			if db == "app" {
				return db, dbMeta, nil
			}
			return "", nil, nil
		},
	}

	got, err := base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, rCtx, script, item)
	require.NoError(t, err)
	require.Contains(t, got, "[app].[dbo].[users]")
	require.Contains(t, got, "[backup_db].[dbo].["+usersBackup.TargetTableName+"]")
	require.Contains(t, got, "[id]")

	// Also accepts projects/... resource name format.
	projectItem := &storepb.PriorBackupDetail_Item{
		SourceTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "projects/proj-1/instances/inst-1/databases/app",
			Schema:   "dbo",
			Table:    "users",
		},
		TargetTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "projects/proj-1/instances/inst-1/databases/backup_db",
			Table:    usersBackup.TargetTableName,
		},
		StartPosition: usersBackup.StartPosition,
		EndPosition:   usersBackup.EndPosition,
	}
	got, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, rCtx, script, projectItem)
	require.NoError(t, err)
	require.Contains(t, got, "[app].[dbo].[users]")

	// Guard rails on missing inputs / metadata.
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, rCtx, script, nil)
	require.ErrorContains(t, err, "backup item is nil")

	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, base.RestoreContext{}, script, item)
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is required")

	nilDBCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: func(_ context.Context, _, _ string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, nilDBCtx, script, item)
	require.ErrorContains(t, err, "database metadata is nil")

	errCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: func(_ context.Context, _, _ string) (string, *model.DatabaseMetadata, error) {
			return "", nil, errors.New("rpc failed")
		},
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, errCtx, script, item)
	require.ErrorContains(t, err, "rpc failed")

	missingSchemaItem := &storepb.PriorBackupDetail_Item{
		SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/inst-1/databases/app", Schema: "missing_schema", Table: "users"},
		TargetTable:   item.TargetTable,
		StartPosition: item.StartPosition,
		EndPosition:   item.EndPosition,
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, rCtx, script, missingSchemaItem)
	require.ErrorContains(t, err, "schema metadata not found")

	missingTableItem := &storepb.PriorBackupDetail_Item{
		SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/inst-1/databases/app", Schema: "dbo", Table: "missing_table"},
		TargetTable:   item.TargetTable,
		StartPosition: item.StartPosition,
		EndPosition:   item.EndPosition,
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_MSSQL, rCtx, script, missingTableItem)
	require.ErrorContains(t, err, "table metadata not found")
}
