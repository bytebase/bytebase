package oracle

import (
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type generateMigrationCase struct {
	Description string `yaml:"description"`
	OldMetadata string `yaml:"oldMetadata"`
	NewMetadata string `yaml:"newMetadata"`
	Expected    string `yaml:"expected"`
}

// TestGenerateMigration pins the DDL that takes oldMetadata to newMetadata. The
// metadata is what Oracle itself reported for each case, captured once; the
// recorder that produced it is gone, so these inputs are frozen and only the
// expected DDL re-records.
func TestGenerateMigration(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/generate_migration.yaml"
	)

	var tests []generateMigrationCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			migration, err := schema.DiffMigration(storepb.Engine_ORACLE,
				parseMetadata(t, tc.OldMetadata), parseMetadata(t, tc.NewMetadata))
			require.NoError(t, err)

			if record {
				tests[i].Expected = migration
				return
			}
			require.Equal(t, tc.Expected, migration)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

func parseMetadata(t *testing.T, encoded string) *model.DatabaseMetadata {
	t.Helper()

	metadata := &metadatapb.DatabaseSchemaMetadata{}
	require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(encoded), metadata))
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_ORACLE, false)
}
