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

// TestDiffMetadataEvent pins the event migration both ways. OceanBase rejects SHOW
// CREATE EVENT, so the sync builds the definition from information_schema, and the
// diff renders it as the sync recorded it.
func TestDiffMetadataEvent(t *testing.T) {
	event := &metadatapb.EventMetadata{
		Name:       "e",
		Definition: "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' ON COMPLETION PRESERVE DISABLE DO INSERT INTO `t` VALUES (1)",
	}
	snapshot := func(events ...*metadatapb.EventMetadata) *model.DatabaseMetadata {
		return oceanBaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "db",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}}},
				Events: events,
			}},
		})
	}

	migration, err := schema.DiffMigration(storepb.Engine_OCEANBASE, snapshot(), snapshot(event))
	require.NoError(t, err)
	require.Contains(t, migration, "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' ON COMPLETION PRESERVE DISABLE COMMENT '' DO INSERT INTO `t` VALUES (1)")

	migration, err = schema.DiffMigration(storepb.Engine_OCEANBASE, snapshot(event), snapshot())
	require.NoError(t, err)
	require.Equal(t, "DROP EVENT IF EXISTS `e`", migration)
}

// TestDiffMetadataRefusesEventWithoutDefinition pins a behavior change of the
// cutover. OceanBase's sync used to record an event's name alone, and the deleted
// generator still dropped such an event by name. The catalog cannot install an event
// without its definition, so a change to one is refused rather than guessed: the
// database has to be re-synced, after which the event carries its definition.
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

// TestDiffMetadataColumnAttributes re-verifies on OceanBase's own catalog what the
// old generator hard-coded for both engines: the SRID and INVISIBLE attributes keep
// MySQL's version-gated spelling.
func TestDiffMetadataColumnAttributes(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int", Nullable: false}

	diff := func(oldCol, newCol *metadatapb.ColumnMetadata) string {
		t.Helper()
		sql, err := schema.DiffMigration(storepb.Engine_OCEANBASE,
			oceanBaseSnapshot(&metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{id, oldCol}}),
			oceanBaseSnapshot(&metadatapb.TableMetadata{Name: "t", Columns: []*metadatapb.ColumnMetadata{id, newCol}}))
		require.NoError(t, err)
		return sql
	}

	require.Equal(t,
		"ALTER TABLE `t` MODIFY COLUMN `pt` point NOT NULL /*!80003 SRID 4326 */",
		diff(&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false},
			&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)}))
	require.Equal(t,
		"ALTER TABLE `t` MODIFY COLUMN `pt` point NOT NULL /*!80003 SRID 0 */",
		diff(&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false},
			&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(0)}))
	require.Empty(t, diff(
		&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
		&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)}))
	require.Equal(t,
		"ALTER TABLE `t` MODIFY COLUMN `c` int DEFAULT NULL /*!80023 INVISIBLE */",
		diff(&metadatapb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL"},
			&metadatapb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL", IsInvisible: true}))
}

// TestDiffMetadataRefusesCaseOnlyIdentities covers the same catalog limit on the
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
