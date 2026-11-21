package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestBackupSeam(t *testing.T) {
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: "public",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{{
				Name:    "t",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "a"}},
			}},
		}},
	}
	tCtx := base.TransformContext{
		InstanceID:              "i1",
		DatabaseName:            "db",
		GetDatabaseMetadataFunc: buildSeamMetadataGetter(meta),
	}

	backups, err := base.TransformDMLToSelect(context.Background(), storepb.Engine_POSTGRES, tCtx, "UPDATE t SET a = 1 WHERE id = 1;", "db", "bbarchive", "rollback")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	require.Equal(t, "public", backups[0].SourceSchema)
	require.Equal(t, "t", backups[0].SourceTableName)
	require.Equal(t, "rollback_t_public", backups[0].TargetTableName)
	// Start column is 0-based to match restore convention.
	require.Equal(t, &storepb.Position{Line: 1, Column: 0}, backups[0].StartPosition)
	require.Equal(t, &storepb.Position{Line: 1, Column: 32}, backups[0].EndPosition)

	_, err = TransformDMLToSelect(context.Background(), base.TransformContext{DatabaseName: "db"}, "UPDATE t SET a = 1;", "db", "bbarchive", "rollback")
	require.ErrorContains(t, err, "GetDatabaseMetadataFunc is not set")

	_, err = TransformDMLToSelect(context.Background(), base.TransformContext{
		DatabaseName: "unsynced",
		GetDatabaseMetadataFunc: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "", nil, nil
		},
	}, "UPDATE t SET a = 1;", "unsynced", "bbarchive", "rollback")
	require.ErrorContains(t, err, `database metadata for database "unsynced" not found (database not synced)`)
}
