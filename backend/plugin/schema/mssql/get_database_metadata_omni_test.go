package mssql

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetDatabaseMetadataOmniParityWithANTLR(t *testing.T) {
	tests := []getDatabaseMetadataCase{}
	yamlFile, err := os.Open("test-data/test_get_database_metadata.yaml")
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(byteValue, &tests))

	for _, tc := range tests {
		antlrMeta, err := getDatabaseMetadataANTLR(tc.Input)
		require.NoError(t, err, tc.Input)
		omniMeta, err := getDatabaseMetadataOmni(tc.Input)
		require.NoError(t, err, tc.Input)

		antlrJSON, err := json.MarshalIndent(antlrMeta, "", "  ")
		require.NoError(t, err)
		omniJSON, err := json.MarshalIndent(omniMeta, "", "  ")
		require.NoError(t, err)
		require.Equal(t, string(antlrJSON), string(omniJSON), tc.Input)
	}
}

func TestGetDatabaseMetadataOmniSequenceUsesExplicitSchema(t *testing.T) {
	metadata, err := getDatabaseMetadataOmni(`
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

func TestGetDatabaseMetadataOmniColumnForeignKeyUsesTableSchemaFallback(t *testing.T) {
	metadata, err := getDatabaseMetadataOmni(`
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
