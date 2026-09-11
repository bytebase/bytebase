package mssql

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetDatabaseMetadataSequenceUsesExplicitSchema(t *testing.T) {
	metadata, err := GetDatabaseMetadata(`
CREATE SCHEMA audit;
CREATE SEQUENCE audit.EventSeq AS BIGINT;
`)
	require.NoError(t, err)

	require.Equal(t, []*storepb.SchemaMetadata{
		{
			Name:   "audit",
			Tables: []*storepb.TableMetadata{},
			Sequences: []*storepb.SequenceMetadata{
				{
					Name:     "EventSeq",
					DataType: "BIGINT",
				},
			},
		},
	}, metadata.Schemas)
}

func TestGetDatabaseMetadataColumnForeignKeyUsesTableSchemaFallback(t *testing.T) {
	metadata, err := GetDatabaseMetadata(`
CREATE SCHEMA sales;
CREATE TABLE sales.customers (
    id INT PRIMARY KEY
);
CREATE TABLE sales.orders (
    id INT PRIMARY KEY,
    customer_id INT REFERENCES customers(id)
);
`)
	require.NoError(t, err)

	var orders *storepb.TableMetadata
	for _, table := range metadata.Schemas[0].Tables {
		if table.Name == "orders" {
			orders = table
			break
		}
	}
	require.NotNil(t, orders)
	require.Len(t, orders.ForeignKeys, 1)
	require.Equal(t, "sales", orders.ForeignKeys[0].ReferencedSchema)
	require.Equal(t, "customers", orders.ForeignKeys[0].ReferencedTable)
	require.Equal(t, []string{"customer_id"}, orders.ForeignKeys[0].Columns)
	require.Equal(t, []string{"id"}, orders.ForeignKeys[0].ReferencedColumns)
}

type getDatabaseMetadataCase struct {
	Description string `yaml:"description"`
	Schema      string `yaml:"schema"`
	Metadata    string `yaml:"metadata"`
}

// TestGetDatabaseMetadata pins what the parser extracts from each schema text.
// The goldens describe this package's own output, not SQL Server's catalog:
// whether the parser agrees with a live server is engine conformance and belongs
// in omni.
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
