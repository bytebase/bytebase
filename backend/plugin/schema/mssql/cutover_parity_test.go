package mssql

import (
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// cutoverParityCase records one migration where omni's catalog deliberately
// differs from the Bytebase generator this package deleted. The legacy text was
// captured before the deletion, so it is the differential even though only the
// catalog path is left to run.
type cutoverParityCase struct {
	Description     string `yaml:"description"`
	Difference      string `yaml:"difference"`
	LegacyExpected  string `yaml:"legacyExpected"`
	CatalogExpected string `yaml:"catalogExpected"`
}

// TestCutoverParity names every migration whose text moved with the cutover.
// Each case is asserted against the catalog output, and the recorded legacy text
// must still differ: if a future catalog update makes a case match, this test
// fails so the case moves back into TestGenerateMigration.
func TestCutoverParity(t *testing.T) {
	content, err := os.ReadFile("testdata/cutover_parity.yaml")
	require.NoError(t, err)
	var cases []cutoverParityCase
	require.NoError(t, yaml.Unmarshal(content, &cases))
	require.NotEmpty(t, cases)

	fixtureContent, err := os.ReadFile("testdata/generate_migration.yaml")
	require.NoError(t, err)
	var fixture []generateMigrationCase
	require.NoError(t, yaml.Unmarshal(fixtureContent, &fixture))
	byDescription := make(map[string]int, len(fixture))
	for i, tc := range fixture {
		byDescription[tc.Description] = i
	}

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			i, ok := byDescription[tc.Description]
			require.True(t, ok, "parity case is missing from generate_migration.yaml")
			source := fixture[i]
			oldDB, newDB := parseMigrationSides(t, i, source.Description, source.OldSchema, source.NewSchema)
			migration, err := schema.DiffMigration(storepb.Engine_MSSQL, oldDB, newDB)
			require.NoError(t, err)
			require.Equal(t, tc.CatalogExpected, migration)
			require.NotEqual(t, tc.LegacyExpected, migration, tc.Difference)
		})
	}
}

// TestChangesBeyondTheOldGenerator names the migration capabilities the catalog
// adds outside the recorded corpus. The old generator rendered CREATE SEQUENCE
// only, without its options, and ignored a sequence alter or drop; it had no
// trigger or synonym branch at all. A synced snapshot can carry those objects
// (the driver sync reports them), so a change now emits what the snapshot says
// instead of silently doing nothing. These are fixes, so they are asserted here
// rather than hidden behind the definition strip.
func TestChangesBeyondTheOldGenerator(t *testing.T) {
	sequence := func(increment string) *model.DatabaseMetadata {
		metadata := &metadatapb.DatabaseSchemaMetadata{}
		if increment != "" {
			metadata.Schemas = []*metadatapb.SchemaMetadata{{Name: "dbo", Sequences: []*metadatapb.SequenceMetadata{{
				Name: "seq", DataType: "bigint", Start: "1", Increment: increment,
			}}}}
		}
		return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MSSQL, false)
	}

	triggerBody := "CREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS BEGIN SELECT 1 END"
	withTrigger := func() *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo",
			Tables: []*metadatapb.TableMetadata{{
				Name:     "t",
				Columns:  []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
				Indexes:  []*metadatapb.IndexMetadata{{Name: "PK_t", Type: "CLUSTERED", Primary: true, Unique: true, IsConstraint: true, Expressions: []string{"id"}}},
				Triggers: []*metadatapb.TriggerMetadata{{Name: "trg", Body: triggerBody}},
			}},
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}
	withoutTrigger := func() *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo",
			Tables: []*metadatapb.TableMetadata{{
				Name:    "t",
				Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
				Indexes: []*metadatapb.IndexMetadata{{Name: "PK_t", Type: "CLUSTERED", Primary: true, Unique: true, IsConstraint: true, Expressions: []string{"id"}}},
			}},
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}

	synonym := func(synonyms []*metadatapb.SynonymMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{
			Name: "dbo", Synonyms: synonyms,
		}}}, nil, nil, storepb.Engine_MSSQL, false)
	}

	tests := []struct {
		name string
		old  *model.DatabaseMetadata
		new  *model.DatabaseMetadata
		want string
	}{
		{name: "create sequence keeps the options", old: sequence(""), new: sequence("1"), want: "CREATE SEQUENCE [dbo].[seq] AS bigint START WITH 1 INCREMENT BY 1;\nGO\n"},
		{name: "alter sequence", old: sequence("1"), new: sequence("5"), want: "ALTER SEQUENCE [dbo].[seq] INCREMENT BY 5 NO CYCLE;\nGO\n"},
		{name: "drop sequence", old: sequence("1"), new: sequence(""), want: "DROP SEQUENCE [dbo].[seq];\nGO\n"},
		{name: "create trigger", old: withoutTrigger(), new: withTrigger(), want: "\nGO\nCREATE TRIGGER [dbo].[trg] ON [dbo].[t] AFTER INSERT AS BEGIN SELECT 1 END\nGO\n"},
		{name: "drop trigger", old: withTrigger(), new: withoutTrigger(), want: "DROP TRIGGER [dbo].[trg];\nGO\n\n"},
		{name: "create synonym", old: synonym(nil), new: synonym([]*metadatapb.SynonymMetadata{{Name: "s1", TargetSchema: "dbo", TargetName: "t"}}), want: "CREATE SYNONYM [dbo].[s1] FOR [dbo].[t];\nGO\n"},
		{name: "drop synonym", old: synonym([]*metadatapb.SynonymMetadata{{Name: "s1", TargetSchema: "dbo", TargetName: "t"}}), new: synonym(nil), want: "DROP SYNONYM [dbo].[s1];\nGO\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := schema.DiffMigration(storepb.Engine_MSSQL, tc.old, tc.new)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
