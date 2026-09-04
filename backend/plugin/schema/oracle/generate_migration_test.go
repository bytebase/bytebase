package oracle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// TestGenerateMigration runs the rollback generator over metadata Oracle itself
// reported for each fixture: diff the post-migration schema back to the initial
// one and compare the DDL against the recorded golden. The metadata comes from
// testdata/generate_migration/*.json, captured by TestRecordGenerateMigrationFixtures.
//
// Set record to regenerate the .sql goldens; it needs no Oracle. Refreshing the
// .json metadata does, and is the recorder's job.
func TestGenerateMigration(t *testing.T) {
	t.Parallel()

	const record = false

	for _, tc := range generateMigrationCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			schemaA, schemaB := loadGenerateMigrationFixture(t, tc.name)

			dbSchemaA := model.NewDatabaseMetadata(schemaA, nil, nil, storepb.Engine_ORACLE, false)
			dbSchemaB := model.NewDatabaseMetadata(schemaB, nil, nil, storepb.Engine_ORACLE, false)

			// Diff B back to A: the migration under test is the rollback.
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_ORACLE, dbSchemaB, dbSchemaA)
			require.NoError(t, err, "failed to diff %s", tc.description)

			rollbackDDL, err := schema.GenerateMigration(storepb.Engine_ORACLE, diff)
			require.NoError(t, err, "failed to generate migration for %s", tc.description)

			goldenPath := generateMigrationGoldenPath(tc.name)
			if record {
				require.NoError(t, os.WriteFile(goldenPath, []byte(rollbackDDL), 0o644))
				return
			}

			want, err := os.ReadFile(goldenPath)
			require.NoErrorf(t, err, "missing golden %s; set record to regenerate", goldenPath)
			require.Equal(t, string(want), rollbackDDL, "generated migration changed for %s", tc.description)
		})
	}
}
