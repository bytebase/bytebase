package mssql

import (
	"encoding/json"
	"io"
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
	Input  string
	Result string
}

func TestGetDatabaseMetadata(t *testing.T) {
	tests := []getDatabaseMetadataCase{}
	const (
		record = false
	)
	var (
		filepath = "test-data/test_get_database_metadata.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		meta, err := GetDatabaseMetadata(t.Input)
		a.NoError(err)

		jsonBytes, err := json.MarshalIndent(meta, "", "  ")
		a.NoError(err)
		result := string(jsonBytes)

		if record {
			tests[i].Result = result
		} else {
			a.Equal(t.Result, result, t.Input)
		}
	}
	if record {
		yamltest.Record(t, filepath, tests)
	}
}
