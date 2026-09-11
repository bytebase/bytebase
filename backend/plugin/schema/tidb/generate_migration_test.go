package tidb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type generateMigrationCase struct {
	Description string `yaml:"description"`
	OldSchema   string `yaml:"oldSchema"`
	NewSchema   string `yaml:"newSchema"`
	Expected    string `yaml:"expected"`
}

// TestGenerateMigration pins the DDL that takes oldSchema to newSchema. Cases come
// in both directions for each schema pair, so an object appears once as a create
// and once as a drop.
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
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_TIDB,
				parseSchema(t, tc.OldSchema), parseSchema(t, tc.NewSchema))
			require.NoError(t, err)

			migration, err := generateMigration(diff)
			require.NoError(t, err)

			if !knownInvalidOutput[tc.Description] {
				requireParses(t, migration)
			}

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

// parseSchema reads a schema text into the model the differ takes. Empty text is
// an empty database rather than a parse, which is how create-from-nothing and
// drop-everything cases are written.
func parseSchema(t *testing.T, text string) *model.DatabaseMetadata {
	t.Helper()

	metadata := &storepb.DatabaseSchemaMetadata{Schemas: []*storepb.SchemaMetadata{{}}}
	if text != "" {
		parsed, err := GetDatabaseMetadata(text)
		require.NoError(t, err)
		metadata = parsed
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_TIDB, false)
}

// knownInvalidOutput names the cases whose generated migration is not valid
// TiDB today: the writer puts comment text into a single-quoted literal without
// escaping, so an apostrophe ends the string early. The golden records that
// output as it is; fixing the writer removes the entry.
var knownInvalidOutput = map[string]bool{
	"Reverse of comments with special characters - removing all special character comments": true,
}

// requireParses proves the generated migration is valid TiDB, which a golden
// alone does not.
func requireParses(t *testing.T, migration string) {
	t.Helper()

	if strings.TrimSpace(migration) == "" {
		return
	}
	p := parser.New()
	mode, err := mysql.GetSQLMode(mysql.DefaultSQLMode)
	require.NoError(t, err)
	p.SetSQLMode(mode)

	_, _, err = p.Parse(migration, "", "")
	require.NoError(t, err, "generated migration should parse")
}
