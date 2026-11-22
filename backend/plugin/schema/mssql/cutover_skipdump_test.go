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

// TestMetadataMigrationSuppressesSkippedObjectChanges pins the legacy differ's
// rule for presence and content changes, not just a pure flag transition: a
// table skipped on either side takes no part in the migration.
func TestMetadataMigrationSuppressesSkippedObjectChanges(t *testing.T) {
	table := func(skip bool, columnType string) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{
			Name:     "t",
			SkipDump: skip,
			Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: columnType}},
		}
	}
	snapshot := func(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name:   "dbo",
			Tables: tables,
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}
	for _, tc := range []struct {
		name     string
		old, new *model.DatabaseMetadata
	}{
		{name: "dropped while skipped", old: snapshot(table(true, "int")), new: snapshot()},
		{name: "created while skipped", old: snapshot(), new: snapshot(table(true, "int"))},
		{name: "changed into skipped", old: snapshot(table(false, "int")), new: snapshot(table(true, "bigint"))},
		{name: "changed out of skipped", old: snapshot(table(true, "int")), new: snapshot(table(false, "bigint"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := schema.DiffMigration(storepb.Engine_MSSQL, tc.old, tc.new)
			require.NoError(t, err)
			require.Empty(t, sql)
		})
	}
}

// TestMetadataMigrationSuppressesSkippedTriggerChanges pins the same rule one
// level down: a trigger skipped on either side must not produce trigger DDL.
func TestMetadataMigrationSuppressesSkippedTriggerChanges(t *testing.T) {
	trigger := func(skip bool, body string) *metadatapb.TriggerMetadata {
		return &metadatapb.TriggerMetadata{Name: "trg", SkipDump: skip, Body: body}
	}
	snapshot := func(trg *metadatapb.TriggerMetadata) *model.DatabaseMetadata {
		table := &metadatapb.TableMetadata{
			Name:     "t",
			Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}},
			Triggers: []*metadatapb.TriggerMetadata{trg},
		}
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo", Tables: []*metadatapb.TableMetadata{table},
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}
	for _, tc := range []struct {
		name     string
		old, new *model.DatabaseMetadata
	}{
		{name: "flag flips on", old: snapshot(trigger(false, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1")), new: snapshot(trigger(true, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1"))},
		{name: "flag flips off", old: snapshot(trigger(true, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1")), new: snapshot(trigger(false, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1"))},
		{name: "body changes while skipped", old: snapshot(trigger(true, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 1")), new: snapshot(trigger(false, "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS SELECT 2"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := schema.DiffMigration(storepb.Engine_MSSQL, tc.old, tc.new)
			require.NoError(t, err)
			require.Empty(t, sql)
		})
	}
}

// TestMetadataMigrationMatchesSkippedNamesCaseInsensitively covers SQL Server's
// default case-insensitive identity: a skipped source "T" and a target "t" are
// the same object, so the skip must still suppress the change.
func TestMetadataMigrationMatchesSkippedNamesCaseInsensitively(t *testing.T) {
	snapshot := func(name string, skip bool, columnType string) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo",
			Tables: []*metadatapb.TableMetadata{{
				Name:     name,
				SkipDump: skip,
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: columnType}},
			}},
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}
	sql, err := schema.DiffMigration(storepb.Engine_MSSQL, snapshot("T", true, "int"), snapshot("t", false, "bigint"))
	require.NoError(t, err)
	require.Empty(t, sql)
}

// TestMetadataMigrationDropsSkippedObjectsWithTheirSchema covers a removed
// schema: its skipped table must still be dropped, or SQL Server rejects the
// schema drop because the physical table remains.
func TestMetadataMigrationDropsSkippedObjectsWithTheirSchema(t *testing.T) {
	old := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "app",
		Tables: []*metadatapb.TableMetadata{{
			Name:     "t",
			SkipDump: true,
			Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}},
		}},
	}}}, nil, nil, storepb.Engine_MSSQL, false)
	target := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{}, nil, nil, storepb.Engine_MSSQL, false)

	sql, err := schema.DiffMigration(storepb.Engine_MSSQL, old, target)
	require.NoError(t, err)
	require.Contains(t, sql, "DROP TABLE")
	require.True(t, old.GetProto().Schemas[0].Tables[0].SkipDump, "reconcileSkipDump must not mutate caller proto")
}

// TestMetadataMigrationMatchesSkippedNamesCaseSensitively covers case-sensitive
// databases: a skipped source "T" does not suppress a distinct target "t".
func TestMetadataMigrationMatchesSkippedNamesCaseSensitively(t *testing.T) {
	meta := func(name string, skip bool) *metadatapb.DatabaseSchemaMetadata {
		return &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo",
			Tables: []*metadatapb.TableMetadata{{
				Name:     name,
				SkipDump: skip,
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}},
			}},
		}}}
	}
	_, reconciled := reconcileSkipDump(meta("T", true), meta("t", false), true)
	require.Len(t, reconciled.Schemas[0].Tables, 2)
	require.Equal(t, "t", reconciled.Schemas[0].Tables[0].Name)
	require.Equal(t, "T", reconciled.Schemas[0].Tables[1].Name)
}
