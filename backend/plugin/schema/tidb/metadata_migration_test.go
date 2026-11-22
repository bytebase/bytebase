package tidb

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func tidbMetadata(proto *metadatapb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_TIDB, true)
}

func tidbTable(columns ...*metadatapb.ColumnMetadata) *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Name: "d",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: columns}},
		}},
	}
}

// TestDiffMetadataRefusesAChangedUnloadableObject pins the refusal rule: an object
// the catalog could not install as defined is skipped while it is the same on both
// sides, and named when it differs, rather than diffed as its stand-in.
func TestDiffMetadataRefusesAChangedUnloadableObject(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	unparsable := &metadatapb.ColumnMetadata{Name: "id", Type: "int", Default: "((("}

	_, err := schema.DiffMigration(storepb.Engine_TIDB, tidbMetadata(tidbTable(id)), tidbMetadata(tidbTable(unparsable)))
	require.ErrorContains(t, err, "table:t")

	sql, err := schema.DiffMigration(storepb.Engine_TIDB, tidbMetadata(tidbTable(unparsable)), tidbMetadata(tidbTable(unparsable)))
	require.NoError(t, err)
	require.Empty(t, sql)
}

// TestDiffMetadataAcrossDatabaseNames proves the synthetic database both snapshots
// load under leaves no trace: sync qualifies a view body with its own database name,
// so the two sides are equal, and a new view loses the qualifier.
func TestDiffMetadataAcrossDatabaseNames(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	view := func(db string) *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{Name: "v", Definition: "select `" + db + "`.`t`.`id` AS `id` from `" + db + "`.`t`"}
	}
	snapshot := func(db string, views ...*metadatapb.ViewMetadata) *model.DatabaseMetadata {
		return tidbMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: db,
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: []*metadatapb.ColumnMetadata{id}}},
				Views:  views,
			}},
		})
	}

	sql, err := schema.DiffMigration(storepb.Engine_TIDB, snapshot("prod", view("prod")), snapshot("staging", view("staging")))
	require.NoError(t, err)
	require.Empty(t, sql)

	sql, err = schema.DiffMigration(storepb.Engine_TIDB, snapshot("prod"), snapshot("staging", view("staging")))
	require.NoError(t, err)
	require.NotContains(t, sql, "staging")
	require.Contains(t, sql, "VIEW `v` (`id`) AS SELECT `t`.`id` AS `id` FROM `t`")
}

// TestDiffMetadataTiDBColumnAttributes pins the two TiDB-only column attributes on a
// MODIFY COLUMN. TiDB reports AUTO_RANDOM with its shard count and a primary key with
// the clustered-index comment; a change to either has to reach the migration.
func TestDiffMetadataTiDBColumnAttributes(t *testing.T) {
	base := &metadatapb.ColumnMetadata{Name: "id", Type: "bigint", Nullable: false}
	autoRandom := &metadatapb.ColumnMetadata{Name: "id", Type: "bigint", Nullable: false, Default: "AUTO_RANDOM(5)"}

	sql, err := schema.DiffMigration(storepb.Engine_TIDB,
		tidbMetadata(tidbTable(base)), tidbMetadata(tidbTable(autoRandom)))
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE `t` MODIFY COLUMN `id` bigint NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */", sql)

	sql, err = schema.DiffMigration(storepb.Engine_TIDB,
		tidbMetadata(tidbTable(autoRandom)), tidbMetadata(tidbTable(base)))
	require.NoError(t, err)
	require.Equal(t, "ALTER TABLE `t` MODIFY COLUMN `id` bigint NOT NULL", sql)
}

// TestDiffMetadataRefusesCaseOnlyIdentities covers the catalog's lower-cased
// relation keys on an object-case-sensitive TiDB instance.
func TestDiffMetadataRefusesCaseOnlyIdentities(t *testing.T) {
	table := func(name string) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{Name: name, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}}
	}
	snapshot := func(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
		return tidbMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "d", Schemas: []*metadatapb.SchemaMetadata{{Tables: tables}}})
	}
	_, err := schema.DiffMigration(storepb.Engine_TIDB, snapshot(table("Foo")), snapshot(table("foo")))
	require.ErrorContains(t, err, "case-only rename")

	_, err = schema.DiffMigration(storepb.Engine_TIDB, snapshot(table("Foo"), table("foo")), snapshot(table("Foo")))
	require.ErrorContains(t, err, "cannot distinguish")
}
