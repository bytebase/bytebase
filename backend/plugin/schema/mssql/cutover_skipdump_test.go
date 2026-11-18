package mssql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestMetadataMigrationIgnoresSkipDumpTransitions pins the legacy differ's
// behavior: SkipDump controls a dump, not whether the object exists, so a
// change in either direction produces no migration.
func TestMetadataMigrationIgnoresSkipDumpTransitions(t *testing.T) {
	metadata := func(skip bool) *metadatapb.DatabaseSchemaMetadata {
		return &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:     "dbo",
			SkipDump: skip,
			Tables: []*metadatapb.TableMetadata{{
				Name:     "t",
				SkipDump: skip,
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v",
				SkipDump:   skip,
				Definition: "CREATE VIEW [dbo].[v] AS SELECT 1",
			}},
		}}}
	}
	wrap := func(meta *metadatapb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(meta, nil, nil, storepb.Engine_MSSQL, false)
	}

	for _, tc := range []struct {
		name     string
		old, new bool
	}{
		{name: "dump to skipped", old: false, new: true},
		{name: "skipped to dump", old: true, new: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := schema.DiffMigration(storepb.Engine_MSSQL, wrap(metadata(tc.old)), wrap(metadata(tc.new)))
			require.NoError(t, err)
			require.Empty(t, sql)
		})
	}
}
