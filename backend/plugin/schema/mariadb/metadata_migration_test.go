package mariadb

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func mariadbMetadata(proto *metadatapb.DatabaseSchemaMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(proto, nil, nil, storepb.Engine_MARIADB, true)
}

func mariadbTable(columns ...*metadatapb.ColumnMetadata) *metadatapb.DatabaseSchemaMetadata {
	return &metadatapb.DatabaseSchemaMetadata{
		Name: "d",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: columns}},
		}},
	}
}

// TestDiffMetadataKeepsSystemVersioning pins the reason the metadata diff runs on
// MariaDB's own catalog: omni's MySQL loader installs this table with plain NOT NULL
// columns and no versioning at all, so a MySQL diff would rewrite it away.
func TestDiffMetadataKeepsSystemVersioning(t *testing.T) {
	implicit := &metadatapb.TableMetadata{
		Name:          "history",
		Engine:        "InnoDB",
		CreateOptions: "WITH SYSTEM VERSIONING",
		Columns:       []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true, Visible: true, Type: "BTREE"},
		},
	}
	explicit := &metadatapb.TableMetadata{
		Name:          "explicit_period",
		Engine:        "InnoDB",
		CreateOptions: "WITH SYSTEM VERSIONING",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "int", Nullable: true},
			{Name: "valid_from", Type: "timestamp(6)", Generation: &metadatapb.GenerationMetadata{Type: metadatapb.GenerationMetadata_TYPE_STORED, Expression: "ROW START"}},
			{Name: "valid_to", Type: "timestamp(6)", Generation: &metadatapb.GenerationMetadata{Type: metadatapb.GenerationMetadata_TYPE_STORED, Expression: "ROW END"}},
		},
	}
	snapshot := func(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
		return mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    "db",
			Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: tables}},
		})
	}

	migration, err := schema.DiffMigration(storepb.Engine_MARIADB, snapshot(), snapshot(implicit, explicit))
	require.NoError(t, err)
	require.Equal(t, "SET @saved_explicit_defaults_for_timestamp = @@explicit_defaults_for_timestamp, explicit_defaults_for_timestamp = 1;\n"+
		"CREATE TABLE `explicit_period` (\n"+
		"  `id` int(11) DEFAULT NULL,\n"+
		"  `valid_from` timestamp(6) GENERATED ALWAYS AS ROW START,\n"+
		"  `valid_to` timestamp(6) GENERATED ALWAYS AS ROW END,\n"+
		"  PERIOD FOR SYSTEM_TIME (`valid_from`, `valid_to`)\n"+
		") ENGINE=InnoDB WITH SYSTEM VERSIONING;\n"+
		"CREATE TABLE `history` (\n"+
		"  `id` int(11) NOT NULL,\n"+
		"  PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB WITH SYSTEM VERSIONING;\n"+
		"SET explicit_defaults_for_timestamp = @saved_explicit_defaults_for_timestamp", migration)

	migration, err = schema.DiffMigration(storepb.Engine_MARIADB, snapshot(implicit, explicit), snapshot(implicit, explicit))
	require.NoError(t, err)
	require.Empty(t, migration)
}

// TestDiffMetadataColumnAttributes re-verifies per engine what the old generator
// hard-coded for both engines. MariaDB has no SRID column attribute, so an SRID-only
// change is no change, and INVISIBLE is plain MariaDB syntax rather than MySQL's
// version-gated comment.
func TestDiffMetadataColumnAttributes(t *testing.T) {
	srid := func(v uint32) *uint32 { return &v }
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int", Nullable: false}

	diff := func(oldCol, newCol *metadatapb.ColumnMetadata) string {
		t.Helper()
		sql, err := schema.DiffMigration(storepb.Engine_MARIADB,
			mariadbMetadata(mariadbTable(id, oldCol)), mariadbMetadata(mariadbTable(id, newCol)))
		require.NoError(t, err)
		return sql
	}

	require.Empty(t, diff(
		&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false},
		&metadatapb.ColumnMetadata{Name: "pt", Type: "point", Nullable: false, Srid: srid(4326)},
	))
	require.Equal(t,
		"ALTER TABLE `t` MODIFY COLUMN `c` int(11) DEFAULT NULL INVISIBLE",
		diff(&metadatapb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL"},
			&metadatapb.ColumnMetadata{Name: "c", Type: "int", Nullable: true, Default: "NULL", IsInvisible: true}))
}

// TestDiffMetadataAcrossDatabaseNames proves the synthetic database the diff loads
// both snapshots under leaves no trace: sync qualifies a view body with its own
// database, so the two sides are equal, and a new view loses the qualifier.
func TestDiffMetadataAcrossDatabaseNames(t *testing.T) {
	id := &metadatapb.ColumnMetadata{Name: "id", Type: "int"}
	view := func(db string) *metadatapb.ViewMetadata {
		return &metadatapb.ViewMetadata{Name: "v", Definition: "select `" + db + "`.`t`.`id` AS `id` from `" + db + "`.`t`"}
	}
	snapshot := func(db string, views ...*metadatapb.ViewMetadata) *model.DatabaseMetadata {
		return mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: db,
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{Name: "t", Columns: []*metadatapb.ColumnMetadata{id}}},
				Views:  views,
			}},
		})
	}

	sql, err := schema.DiffMigration(storepb.Engine_MARIADB, snapshot("prod", view("prod")), snapshot("staging", view("staging")))
	require.NoError(t, err)
	require.Empty(t, sql)

	sql, err = schema.DiffMigration(storepb.Engine_MARIADB, snapshot("prod"), snapshot("staging", view("staging")))
	require.NoError(t, err)
	require.NotContains(t, sql, "staging")
	require.Contains(t, sql, "VIEW `v` AS select `t`.`id` AS `id` from `t`")
}

// TestDiffMetadataRefusesAChangedUnloadableObject pins the refusal rule: an object
// the catalog cannot install as defined is skipped while it is the same on both
// sides, and named when it differs. A routine parsed from a schema text carries its
// body alone, which is not the SHOW CREATE text the loader installs.
func TestDiffMetadataRefusesAChangedUnloadableObject(t *testing.T) {
	withRoutine := func(body string) *model.DatabaseMetadata {
		return mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "d",
			Schemas: []*metadatapb.SchemaMetadata{{
				Procedures: []*metadatapb.ProcedureMetadata{{Name: "p", Definition: body}},
			}},
		})
	}

	_, err := schema.DiffMigration(storepb.Engine_MARIADB, withRoutine("BEGIN\n  SELECT 1;\nEND"), withRoutine("BEGIN\n  SELECT 2;\nEND"))
	require.ErrorContains(t, err, "procedure:p")

	// Unchanged, so the stand-in both loads agree on is skipped rather than refused.
	sql, err := schema.DiffMigration(storepb.Engine_MARIADB, withRoutine("BEGIN\n  SELECT 1;\nEND"), withRoutine("BEGIN\n  SELECT 1;\nEND"))
	require.NoError(t, err)
	require.Empty(t, sql)
}

// TestDiffMetadataKeepsExplicitTableCharset keeps clearImplicitTableCharsets
// narrow: only an option the snapshot leaves unset is dropped, so an explicit
// charset and collation still reach the migration.
func TestDiffMetadataKeepsExplicitTableCharset(t *testing.T) {
	source := mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "d", Schemas: []*metadatapb.SchemaMetadata{{}}})
	target := mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "d",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:      "t",
				Engine:    "InnoDB",
				Charset:   "latin1",
				Collation: "latin1_swedish_ci",
				Columns:   []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
			}},
		}},
	})
	sql, err := schema.DiffMigration(storepb.Engine_MARIADB, source, target)
	require.NoError(t, err)
	require.Contains(t, sql, "DEFAULT CHARSET=latin1 COLLATE=latin1_swedish_ci")
}

// TestDiffMetadataRefusesCaseOnlyIdentities covers the one snapshot shape
// omni's MariaDB catalog cannot represent: names that differ only by case.
func TestDiffMetadataRefusesCaseOnlyIdentities(t *testing.T) {
	table := func(name string) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{Name: name, Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int"}}}
	}
	snapshot := func(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
		return mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "d", Schemas: []*metadatapb.SchemaMetadata{{Tables: tables}}})
	}

	_, err := schema.DiffMigration(storepb.Engine_MARIADB, snapshot(table("Foo")), snapshot(table("foo")))
	require.ErrorContains(t, err, "case-only rename")

	_, err = schema.DiffMigration(storepb.Engine_MARIADB, snapshot(table("Foo"), table("foo")), snapshot(table("Foo")))
	require.ErrorContains(t, err, "cannot distinguish")
}
