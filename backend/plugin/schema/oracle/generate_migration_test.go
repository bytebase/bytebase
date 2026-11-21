package oracle

import (
	"strings"
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
		Name: "APP",
		Tables: []*metadatapb.TableMetadata{{
			Name:    "T",
			Columns: []*metadatapb.ColumnMetadata{{Name: "ID", Type: "NUMBER", Nullable: false}},
		}},
	}}}
	newMeta := proto.CloneOf(oldMeta)
	newMeta.Schemas[0].Tables[0].Columns = append(newMeta.Schemas[0].Tables[0].Columns, &metadatapb.ColumnMetadata{
		Name:     "NOTE",
		Type:     "VARCHAR2(50 BYTE)",
		Nullable: true,
	})

	source := model.NewDatabaseMetadata(oldMeta, nil, nil, storepb.Engine_ORACLE, false)
	target := model.NewDatabaseMetadata(newMeta, nil, nil, storepb.Engine_ORACLE, false)

	empty, err := schema.DiffMigration(storepb.Engine_ORACLE, source, source)
	require.NoError(t, err)
	require.Empty(t, empty)

	// Bytebase appends a trailing newline to non-empty Oracle migration scripts.
	migration, err := schema.DiffMigration(storepb.Engine_ORACLE, source, target)
	require.NoError(t, err)
	require.Contains(t, migration, `ALTER TABLE "APP"."T" ADD "NOTE" VARCHAR2(50 BYTE);`)
	require.True(t, strings.HasSuffix(migration, "\n"), "non-empty Oracle migration must end with newline: %q", migration)

	fromNil, err := schema.DiffMigration(storepb.Engine_ORACLE, nil, source)
	require.NoError(t, err)
	require.Contains(t, fromNil, `CREATE TABLE "APP"."T"`)
	require.True(t, strings.HasSuffix(fromNil, "\n"))
}

// TestMetadataMigrationKeepsLegacyOmissions pins the migration path to the same
// compatibility strip as the definition path. An identity column or an invisible
// index in the target must not reach the catalog diff, and the input proto must
// not be mutated.
func TestMetadataMigrationKeepsLegacyOmissions(t *testing.T) {
	oldMetadata := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
		Name: "APP",
		Tables: []*metadatapb.TableMetadata{{
			Name:    "T",
			Columns: []*metadatapb.ColumnMetadata{{Name: "ID", Type: "NUMBER", Nullable: false}},
		}},
	}}}
	newMetadata := proto.CloneOf(oldMetadata)
	table := newMetadata.Schemas[0].Tables[0]
	table.Columns = append(table.Columns, &metadatapb.ColumnMetadata{
		Name:               "SEQ",
		Type:               "NUMBER",
		Nullable:           false,
		IsIdentity:         true,
		IdentityGeneration: metadatapb.ColumnMetadata_ALWAYS,
		IdentitySeed:       1,
		IdentityIncrement:  1,
	})
	table.Indexes = []*metadatapb.IndexMetadata{{
		Name:        "IDX_T_ID",
		Expressions: []string{"ID"},
		Visible:     false,
	}}

	source := model.NewDatabaseMetadata(oldMetadata, nil, nil, storepb.Engine_ORACLE, false)
	target := model.NewDatabaseMetadata(newMetadata, nil, nil, storepb.Engine_ORACLE, false)
	migration, err := schema.DiffMigration(storepb.Engine_ORACLE, source, target)
	require.NoError(t, err)
	require.NotEmpty(t, migration)
	require.NotContains(t, migration, "IDENTITY")
	require.NotContains(t, migration, "GENERATED")
	require.NotContains(t, migration, "INVISIBLE")
	require.True(t, newMetadata.Schemas[0].Tables[0].Columns[1].IsIdentity, "stripUnrenderedFacts must not mutate caller proto")
	require.False(t, newMetadata.Schemas[0].Tables[0].Indexes[0].Visible, "stripUnrenderedFacts must not mutate caller proto")
}
