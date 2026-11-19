package mariadb

import (
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	mysqlschema "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
)

// cutoverParityCase records one migration whose text the cutover moved. The
// legacy text was captured from the generator this change deleted —
// schema/mysql's Bytebase differ, which served MariaDB and OceanBase alike — so
// it is the differential even though only the catalog path is left to run. A
// case with Err set is one the catalog refuses instead of rendering.
type cutoverParityCase struct {
	Description     string `yaml:"description"`
	Difference      string `yaml:"difference"`
	OldSchema       string `yaml:"oldSchema"`
	NewSchema       string `yaml:"newSchema"`
	LegacyExpected  string `yaml:"legacyExpected"`
	CatalogExpected string `yaml:"catalogExpected"`
	Err             string `yaml:"err"`
}

// TestCutoverParity names every kind of difference the cutover introduced. All 36
// cases of the deleted generator corpus moved — the catalog separates statements
// with a newline instead of a blank line, so no case could stay byte-identical —
// each case falls into one of the groups recorded here, and the group's text is
// asserted against the catalog.
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
			migration, err := schema.DiffMigration(storepb.Engine_MARIADB,
				parityMetadata(t, tc.OldSchema), parityMetadata(t, tc.NewSchema))
			if record {
				tests[i].CatalogExpected, tests[i].Err = migration, ""
				if err != nil {
					tests[i].Err, tests[i].CatalogExpected = err.Error(), ""
				}
				return
			}
			if tc.Err != "" {
				require.ErrorContains(t, err, tc.Err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.CatalogExpected, migration)
			require.NotEqual(t, tc.LegacyExpected, tc.CatalogExpected,
				"the case matches what the deleted generator produced; move it back to a behavior test")
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// parityMetadata reads a corpus schema text into the model the diff takes. It goes
// through the MySQL extractor because that is what a MariaDB or OceanBase schema
// text is parsed with — the engines have no extractor of their own — and it is the
// input the deleted generator was fed too.
func parityMetadata(t *testing.T, text string) *model.DatabaseMetadata {
	t.Helper()

	metadata := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{}}}
	if text != "" {
		parsed, err := mysqlschema.GetDatabaseMetadata(text)
		require.NoError(t, err)
		metadata = parsed
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MARIADB, false)
}
