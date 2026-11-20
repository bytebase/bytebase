package mssql

import (
	"encoding/json"
	"os"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
)

func TestGetDatabaseMetadataSequenceUsesExplicitSchema(t *testing.T) {
	metadata, err := GetDatabaseMetadata(`
CREATE SCHEMA audit;
CREATE SEQUENCE audit.EventSeq AS BIGINT;
`)
	require.NoError(t, err)

	require.Equal(t, []*metadatapb.SchemaMetadata{
		{
			Name:   "audit",
			Tables: []*metadatapb.TableMetadata{},
			Sequences: []*metadatapb.SequenceMetadata{
				{
					Name:     "EventSeq",
					DataType: "BIGINT",
				},
			},
		},
	}, metadata.Schemas)
}

func TestGetDatabaseMetadataColumnForeignKeyUsesDefaultSchemaFallback(t *testing.T) {
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

	var orders *metadatapb.TableMetadata
	for _, table := range metadata.Schemas[0].Tables {
		if table.Name == "orders" {
			orders = table
			break
		}
	}
	require.NotNil(t, orders)
	require.Len(t, orders.ForeignKeys, 1)
	require.Equal(t, "dbo", orders.ForeignKeys[0].ReferencedSchema)
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

func TestGetDatabaseMetadataCaseSensitiveSchemasIndexedViewAndDefaults(t *testing.T) {
	metadata, err := GetDatabaseMetadata(`
CREATE TABLE [s].[t] ([a] INT NOT NULL DEFAULT 0);
CREATE TABLE [S].[t] ([b] INT NOT NULL);
GO
CREATE VIEW [dbo].[v] WITH SCHEMABINDING AS SELECT 1 AS [id];
GO
CREATE UNIQUE CLUSTERED INDEX [ix_v] ON [dbo].[v] ([id]);
GO
CREATE TABLE [dbo].[a_b] ([c] INT NOT NULL DEFAULT 0);
CREATE TABLE [dbo].[a] ([b_c] INT NOT NULL CONSTRAINT [DF_custom] DEFAULT 1);
`)
	require.NoError(t, err)

	var schemaNames []string
	var sLower, sUpper, dbo *metadatapb.SchemaMetadata
	for _, s := range metadata.Schemas {
		schemaNames = append(schemaNames, s.Name)
		switch s.Name {
		case "s":
			sLower = s
		case "S":
			sUpper = s
		case "dbo":
			dbo = s
		default:
			t.Fatalf("unexpected schema %q", s.Name)
		}
	}
	require.Equal(t, []string{"S", "dbo", "s"}, schemaNames)
	require.Len(t, sLower.Tables, 1)
	require.Equal(t, "a", sLower.Tables[0].Columns[0].Name)
	require.Empty(t, sLower.Tables[0].Columns[0].DefaultConstraintName)
	require.Len(t, sUpper.Tables, 1)
	require.Equal(t, "b", sUpper.Tables[0].Columns[0].Name)

	require.Len(t, dbo.Views, 1)
	require.Len(t, dbo.Tables, 2)
	require.Equal(t, "a_b", dbo.Tables[0].Name)
	require.Empty(t, dbo.Tables[0].Columns[0].DefaultConstraintName)
	require.Equal(t, "a", dbo.Tables[1].Name)
	require.Equal(t, "DF_custom", dbo.Tables[1].Columns[0].DefaultConstraintName)
}
