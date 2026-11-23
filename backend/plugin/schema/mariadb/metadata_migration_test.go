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

// TestDiffMetadataRefusesAChangedUnloadableObject pins the refusal rule: an object
// the catalog cannot install as defined is skipped while it is the same on both
// sides, and named when it differs.
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

	sql, err := schema.DiffMigration(storepb.Engine_MARIADB, withRoutine("BEGIN\n  SELECT 1;\nEND"), withRoutine("BEGIN\n  SELECT 1;\nEND"))
	require.NoError(t, err)
	require.Empty(t, sql)
}

// TestDiffMetadataTableCharsetPolicy pins clearImplicitTableCharsets: an unset
// table charset/collation is omitted so older MariaDB servers apply their own
// defaults, while an explicit charset and collation still reach the migration.
func TestDiffMetadataTableCharsetPolicy(t *testing.T) {
	source := mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{Name: "d", Schemas: []*metadatapb.SchemaMetadata{{}}})

	t.Run("explicit charset and collation preserved", func(t *testing.T) {
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
	})

	t.Run("implicit charset and collation omitted", func(t *testing.T) {
		target := mariadbMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name: "d",
			Schemas: []*metadatapb.SchemaMetadata{{
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t",
					Engine:  "InnoDB",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
				}},
			}},
		})
		sql, err := schema.DiffMigration(storepb.Engine_MARIADB, source, target)
		require.NoError(t, err)
		require.Contains(t, sql, "CREATE TABLE `t`")
		require.NotContains(t, sql, "CHARSET=")
		require.NotContains(t, sql, "COLLATE=")
	})
}

// TestDiffMetadataRefusesCaseOnlyIdentities covers the snapshot shape omni's
// MariaDB catalog cannot represent: names that differ only by case.
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
