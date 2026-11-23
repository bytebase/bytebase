package oceanbase

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func oceanBaseMetadata(proto *metadatapb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_OCEANBASE, true)
}

func oceanBaseSnapshot(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
	return oceanBaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: tables}},
	})
}

func TestDiffMetadataRegistryDispatch(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int", Nullable: false}
	extra := &metadatapb.ColumnMetadata{Name: "name", Type: "varchar(64)", Nullable: true, Default: "NULL"}

	migration, err := schema.DiffMigration(
		storepb.Engine_OCEANBASE,
		oceanBaseSnapshot(&metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{id}}),
		oceanBaseSnapshot(&metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{id, extra}}),
	)
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE `t` ADD COLUMN `name` varchar(64) DEFAULT NULL", migration)
}

// TestDiffMetadataRefusesEventWithoutDefinition pins the refusal seam: an event
// synced without its definition cannot be installed as defined, so any change to
// it is refused rather than guessed.
func TestDiffMetadataRefusesEventWithoutDefinition(t *testing.T) {
	withEvent := func(event *metadatapb.EventMetadata) *model.DatabaseMetadata {
		return oceanBaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    "db",
			Schemas: []*metadatapb.SchemaMetadata{{Name: "", Events: []*metadatapb.EventMetadata{event}}},
		})
	}
	withoutEvent := oceanBaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: ""}},
	})

	_, err := schema.DiffMigration(storepb.Engine_OCEANBASE, withEvent(&metadatapb.EventMetadata{Name: "e"}), withoutEvent)
	require.ErrorContains(t, err, "event:e")

	_, err = schema.DiffMigration(storepb.Engine_OCEANBASE, withoutEvent, withEvent(&metadatapb.EventMetadata{Name: "e"}))
	require.ErrorContains(t, err, "event:e")
}

// TestDiffMetadataRefusesCaseOnlyIdentities covers the catalog limit on the
// OceanBase fork: relation names that fold to one lower-cased key are refused.
func TestDiffMetadataRefusesCaseOnlyIdentities(t *testing.T) {
	table := func(name string) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{Name: name, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}}
	}
	_, err := schema.DiffMigration(storepb.Engine_OCEANBASE, oceanBaseSnapshot(table("Foo")), oceanBaseSnapshot(table("foo")))
	require.ErrorContains(t, err, "case-only rename")

	_, err = schema.DiffMigration(storepb.Engine_OCEANBASE, oceanBaseSnapshot(table("Foo"), table("foo")), oceanBaseSnapshot(table("Foo")))
	require.ErrorContains(t, err, "cannot distinguish")
}
