package mysql

import (
	"encoding/json"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

type getDatabaseMetadataCase struct {
	Description string `yaml:"description"`
	Schema      string `yaml:"schema"`
	Metadata    string `yaml:"metadata"`
}

// TestGetDatabaseMetadata pins what the parser extracts from each schema text.
// The goldens describe this package's own output, not MySQL's catalog: whether
// the parser agrees with a live server is engine conformance and belongs in omni.
func TestGetDatabaseMetadata(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_metadata.yaml"
	)

	var tests []getDatabaseMetadataCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			metadata, err := GetDatabaseMetadata(tc.Schema)
			require.NoError(t, err)

			encoded, err := json.MarshalIndent(metadata, "", "  ")
			require.NoError(t, err)
			result := string(encoded) + "\n"

			if record {
				tests[i].Metadata = result
				return
			}
			require.Equal(t, tc.Metadata, result)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// TestGetDatabaseMetadataParsesPrettyView proves GetDatabaseMetadata accepts the
// multi-line CREATE OR REPLACE VIEW the SDL dump pretty-prints.
func TestGetDatabaseMetadataParsesPrettyView(t *testing.T) {
	sdl, err := GetDatabaseDefinition(schema.GetDefinitionContext{SDLFormat: true}, &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{{
				Name:    "t1",
				Engine:  "InnoDB",
				Columns: []*metadatapb.ColumnMetadata{{Name: "a", Type: "int"}, {Name: "b", Type: "varchar(10)", Nullable: true, Default: "NULL"}},
				Indexes: []*metadatapb.IndexMetadata{{Name: "PRIMARY", Primary: true, Unique: true, Expressions: []string{"a"}, Visible: true}},
			}},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "v_pretty",
				Definition: "select `testdb`.`t1`.`a` AS `a`,`testdb`.`t1`.`b` AS `b` from `testdb`.`t1` where (`testdb`.`t1`.`a` > 0)",
			}},
		}},
	})
	require.NoError(t, err)
	require.Contains(t, sdl, "AS\nselect\n  `t1`.`a` AS `a`,", "fixture must actually be pretty-printed")

	t.Run("omni", func(t *testing.T) {
		meta, err := GetDatabaseMetadata(sdl)
		require.NoError(t, err)
		require.Len(t, meta.Schemas, 1)
		require.Len(t, meta.Schemas[0].Views, 1)
		view := meta.Schemas[0].Views[0]
		require.Equal(t, "v_pretty", view.Name)
		require.NotContains(t, view.Definition, "\n",
			"omni deparse must canonicalize the pretty body back to one line (whitespace washes out)")
	})
}
