package mysql

import (
	"os"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

type getDatabaseDefinitionCase struct {
	Description string `yaml:"description"`
	Metadata    string `yaml:"metadata"`
	Expected    string `yaml:"expected"`
}

// TestGetDatabaseDefinition pins the DDL generated for each metadata input,
// and re-parses that DDL to prove it is valid MySQL syntax.
//
// The input is metadata rather than a schema text on purpose: deriving it by
// parsing would cap this test's reach at whatever GetDatabaseMetadata happens to
// extract, so any generator branch the parser cannot feed would go untested.
// unbalancedDelimiterBlocks names the cases whose definition ends a routine with
// ";;" without a DELIMITER directive having made ";;" the terminator. writeEvent,
// writeTrigger and writeProcedure all emit the opening "DELIMITER ;;"; only
// writeFunction leaves it out, so a client restoring such a dump ends the
// function at the first semicolon in its body. The golden records that output;
// giving writeFunction the opener empties this map.
var unbalancedDelimiterBlocks = map[string]bool{
	"Stored procedures and functions":                                               true,
	"Check constraints with string literals and function with multiline parameters": true,
}

// requireDelimiterDiscipline checks what a mysql client relies on to restore a
// dump: a line ending in ";;" terminates a statement only while a DELIMITER
// directive has ";;" active.
func requireDelimiterDiscipline(t *testing.T, definition string) {
	t.Helper()

	active := ";"
	for i, line := range strings.Split(definition, "\n") {
		trimmed := strings.TrimSpace(line)
		if directive, ok := strings.CutPrefix(trimmed, "DELIMITER "); ok {
			active = strings.TrimSpace(directive)
			continue
		}
		if strings.HasSuffix(trimmed, ";;") && active != ";;" {
			t.Fatalf("line %d ends a statement with %q while %q is the active delimiter, so a client cuts it short: %s",
				i+1, ";;", active, trimmed)
		}
	}
}

func TestGetDatabaseDefinition(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/get_database_definition.yaml"
	)

	var tests []getDatabaseDefinitionCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			var metadata metadatapb.DatabaseSchemaMetadata
			require.NoError(t, common.ProtojsonUnmarshaler.Unmarshal([]byte(tc.Metadata), &metadata))

			definition, err := GetDatabaseDefinition(schema.GetDefinitionContext{PrintHeader: true}, &metadata)
			require.NoError(t, err)
			require.NotEmpty(t, definition)

			// Routines are emitted inside DELIMITER blocks, which is mysql-client
			// script syntax the schema parser does not accept, so those
			// definitions are checked for delimiter discipline instead.
			if strings.Contains(definition, "DELIMITER") {
				if !unbalancedDelimiterBlocks[tc.Description] {
					requireDelimiterDiscipline(t, definition)
				}
			} else {
				_, err = GetDatabaseMetadata(definition)
				require.NoError(t, err, "generated definition should parse")
			}

			if record {
				tests[i].Expected = definition
				return
			}
			require.Equal(t, tc.Expected, definition)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}
