package mssql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

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
