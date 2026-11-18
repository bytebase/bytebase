package tidb

import (
	"os"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// cutoverParityCase records one migration whose text the cutover moved. The legacy
// text was captured from the generator this change deleted, so it is the
// differential even though only the catalog path is left to run.
type cutoverParityCase struct {
	Description     string `yaml:"description"`
	Difference      string `yaml:"difference"`
	OldSchema       string `yaml:"oldSchema"`
	NewSchema       string `yaml:"newSchema"`
	LegacyExpected  string `yaml:"legacyExpected"`
	CatalogExpected string `yaml:"catalogExpected"`
}

// TestCutoverParity names every kind of difference the cutover introduced. All 29
// cases of the deleted generator corpus moved — the catalog separates statements with
// a newline where the generator wrote a blank line — and each falls into one of the
// groups recorded here. The catalog output is asserted, and the recorded legacy text
// must still differ: if a future catalog update makes a case match, this test fails so
// the group can be retired.
func TestCutoverParity(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/cutover_parity.yaml"
	)

	var tests []cutoverParityCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			migration, err := schema.DiffMigration(storepb.Engine_TIDB,
				parityMetadata(t, tc.OldSchema), parityMetadata(t, tc.NewSchema))
			require.NoError(t, err)

			if record {
				tests[i].CatalogExpected = migration
				return
			}
			require.Equal(t, tc.CatalogExpected, migration)
			requireValidTiDB(t, migration)
			require.NotEqual(t, tc.LegacyExpected, tc.CatalogExpected,
				"the case matches what the deleted generator produced; retire the group")
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// requireValidTiDB proves the migration parses, which a golden alone does not. The
// deleted corpus asserted the same for every case but the one it recorded as invalid
// output: the catalog escapes a comment's apostrophe, so that exception is gone.
func requireValidTiDB(t *testing.T, migration string) {
	t.Helper()

	if strings.TrimSpace(migration) == "" {
		return
	}
	_, err := tidbparser.ParseTiDB(migration)
	require.NoError(t, err, "generated migration should parse")
}

// parityMetadata reads a corpus schema text into the model the diff takes.
func parityMetadata(t *testing.T, text string) *model.DatabaseMetadata {
	t.Helper()

	metadata := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{}}}
	if text != "" {
		parsed, err := GetDatabaseMetadata(text)
		require.NoError(t, err)
		metadata = parsed
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_TIDB, false)
}
