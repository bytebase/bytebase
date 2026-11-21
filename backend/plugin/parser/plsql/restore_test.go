package plsql

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
		Name: "HR",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "EMP",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "ID", Type: "NUMBER"},
							{Name: "NAME", Type: "VARCHAR2"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "PK_EMP", Primary: true, Expressions: []string{"ID"}},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_ORACLE, false)

	script := "UPDATE other_tbl SET x = 1;\nUPDATE emp SET name = 'bob' WHERE id = 1;"
	item := &storepb.PriorBackupDetail_Item{
		SourceTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "instances/inst-1/databases/HR",
			Table:    "EMP",
		},
		TargetTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "instances/inst-1/databases/BACKUP",
			Table:    "BB_EMP",
		},
		StartPosition: &storepb.Position{Line: 1, Column: 1},
		EndPosition:   &storepb.Position{Line: 2, Column: 42},
	}

	rCtx := base.RestoreContext{
		InstanceID: "inst-1",
		GetDatabaseMetadataFunc: func(_ context.Context, _, db string) (string, *model.DatabaseMetadata, error) {
			if db == "HR" {
				return db, dbMeta, nil
			}
			return "", nil, nil
		},
	}

	// Filters to statements referencing EMP and plumbs primary key metadata.
	got, err := base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, rCtx, script, item)
	require.NoError(t, err)
	require.Contains(t, got, `"HR"."EMP"`)
	require.Contains(t, got, `"BACKUP"."BB_EMP"`)
	require.Contains(t, got, `"ID"`)

	// Also accepts projects/... resource name format.
	projectItem := &storepb.PriorBackupDetail_Item{
		SourceTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "projects/proj-1/instances/inst-1/databases/HR",
			Table:    "EMP",
		},
		TargetTable: &storepb.PriorBackupDetail_Item_Table{
			Database: "projects/proj-1/instances/inst-1/databases/BACKUP",
			Table:    "BB_EMP",
		},
		StartPosition: item.StartPosition,
		EndPosition:   item.EndPosition,
	}
	got, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, rCtx, script, projectItem)
	require.NoError(t, err)
	require.Contains(t, got, `"HR"."EMP"`)

	// Guard rails on missing inputs / metadata.
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, rCtx, script, nil)
	require.ErrorContains(t, err, "backup item is nil")

	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, base.RestoreContext{}, script, item)
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is required")

	unsyncedCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: func(_ context.Context, _, _ string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, unsyncedCtx, script, item)
	require.ErrorContains(t, err, "database not synced")

	errCtx := base.RestoreContext{
		GetDatabaseMetadataFunc: func(_ context.Context, _, _ string) (string, *model.DatabaseMetadata, error) {
			return "", nil, errors.New("rpc failed")
		},
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, errCtx, script, item)
	require.ErrorContains(t, err, "rpc failed")

	missingTableItem := &storepb.PriorBackupDetail_Item{
		SourceTable:   &storepb.PriorBackupDetail_Item_Table{Database: "instances/inst-1/databases/HR", Table: "NO_SUCH_TABLE"},
		TargetTable:   item.TargetTable,
		StartPosition: &storepb.Position{Line: 1, Column: 1},
		EndPosition:   &storepb.Position{Line: 1, Column: 30},
	}
	_, err = base.GenerateRestoreSQL(context.Background(), storepb.Engine_ORACLE, rCtx, "UPDATE no_such_table SET x = 1;", missingTableItem)
	require.ErrorContains(t, err, "no table metadata")
}
