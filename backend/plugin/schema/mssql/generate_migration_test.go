package mssql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestDiffMigrationSeam(t *testing.T) {
	oldMeta := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "dbo",
		Tables: []*metadatapb.TableMetadata{{
			Name:    "t",
			Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
		}},
	}}}
	newMeta := proto.CloneOf(oldMeta)
	newMeta.Schemas[0].Tables[0].Columns = append(newMeta.Schemas[0].Tables[0].Columns, &metadatapb.ColumnMetadata{
		Name:     "note",
		Type:     "nvarchar(50)",
		Nullable: true,
	})

	source := model.NewDatabaseMetadata(oldMeta, nil, nil, storepb.Engine_MSSQL, false)
	target := model.NewDatabaseMetadata(newMeta, nil, nil, storepb.Engine_MSSQL, false)
	blank := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, nil, nil, storepb.Engine_MSSQL, false)

	empty, err := schema.DiffMigration(storepb.Engine_MSSQL, source, source)
	require.NoError(t, err)
	require.Empty(t, empty)

	migration, err := schema.DiffMigration(storepb.Engine_MSSQL, source, target)
	require.NoError(t, err)
	require.Contains(t, migration, "ALTER TABLE [dbo].[t] ADD [note] nvarchar(50) NULL;")

	fromEmpty, err := schema.DiffMigration(storepb.Engine_MSSQL, blank, source)
	require.NoError(t, err)
	require.Contains(t, fromEmpty, "CREATE TABLE [dbo].[t]")

	nilOld, nilNew := reconcileSkipDump(nil, nil, false)
	require.Empty(t, nilOld.GetSchemas())
	require.Empty(t, nilNew.GetSchemas())
}
