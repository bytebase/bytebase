package oracle

import (
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// cutoverParityCase holds a fixture's two expected migration texts: the
// catalog's, and the same script with its trailing blank line.
type cutoverParityCase struct {
	Description     string `yaml:"description"`
	LegacyExpected  string `yaml:"legacyExpected"`
	CatalogExpected string `yaml:"catalogExpected"`
}

// TestCutoverParity pins the fixture cases where the catalog migration ends
// after the last statement while the second expected text keeps one trailing
// blank line. Every other case in generate_migration.yaml is byte-identical and
// is covered by TestGenerateMigration.
func TestCutoverParity(t *testing.T) {
	content, err := os.ReadFile("testdata/cutover_parity.yaml")
	require.NoError(t, err)
	var cases []cutoverParityCase
	require.NoError(t, yaml.Unmarshal(content, &cases))

	fixtureContent, err := os.ReadFile("testdata/generate_migration.yaml")
	require.NoError(t, err)
	var fixture []generateMigrationCase
	require.NoError(t, yaml.Unmarshal(fixtureContent, &fixture))
	byDescription := make(map[string]generateMigrationCase, len(fixture))
	for _, tc := range fixture {
		byDescription[tc.Description] = tc
	}

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			source, ok := byDescription[tc.Description]
			require.True(t, ok, "parity case is missing from generate_migration.yaml")
			migration, err := schema.DiffMigration(storepb.Engine_ORACLE,
				parseMetadata(t, source.OldMetadata), parseMetadata(t, source.NewMetadata))
			require.NoError(t, err)
			require.Equal(t, tc.CatalogExpected, migration)
			// The fixture's second text keeps the trailing blank line; the
			// catalog's does not.
			require.Equal(t, tc.LegacyExpected, migration+"\n")
		})
	}
}

// TestMetadataMigrationKeepsLegacyOmissions pins the migration path to the same
// compatibility strip as the definition path. An identity column or an invisible
// index in the target must not reach the catalog diff.
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
}

// TestMaterializedViewDefinitionDropsIndexVisibility pins the per-object path to
// the same compatibility strip as the database definition: an index's INVISIBLE
// state is deferred.
func TestMaterializedViewDefinitionDropsIndexVisibility(t *testing.T) {
	definition, err := GetMaterializedViewDefinition("", &metadatapb.MaterializedViewMetadata{
		Name:       "MV",
		Definition: "SELECT 1 FROM DUAL",
		Indexes:    []*metadatapb.IndexMetadata{{Name: "IDX_MV", Expressions: []string{"1"}, Visible: false}},
	})
	require.NoError(t, err)
	require.NotContains(t, definition, "INVISIBLE")
}
