package model

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestBuildTablesMetadata(t *testing.T) {
	testCases := []struct {
		input       *storepb.TableMetadata
		wantNames   []string
		wantColumns []*storepb.ColumnMetadata
	}{
		// No partitions.
		{
			input: &storepb.TableMetadata{
				Name: "orders",
				Columns: []*storepb.ColumnMetadata{
					{
						Name: "id",
					},
				},
			},
			wantNames: []string{"orders"},
			wantColumns: []*storepb.ColumnMetadata{
				{
					Name: "id",
				},
			},
		},
		// Nested partitions.
		{
			input: &storepb.TableMetadata{
				Name: "orders",
				Columns: []*storepb.ColumnMetadata{
					{
						Name: "id",
					},
				},
				Partitions: []*storepb.TablePartitionMetadata{
					{
						Name: "orders_0_100",
						Subpartitions: []*storepb.TablePartitionMetadata{
							{
								Name: "orders_0_50",
							},
							{
								Name: "orders_50_100",
							},
						},
					},
					{
						Name: "orders_100_200",
						Subpartitions: []*storepb.TablePartitionMetadata{
							{
								Name: "orders_100_150",
							},
							{
								Name: "orders_150_200",
							},
						},
					},
				},
			},
			wantNames: []string{"orders", "orders_0_100", "orders_0_50", "orders_50_100", "orders_100_200", "orders_100_150", "orders_150_200"},
			wantColumns: []*storepb.ColumnMetadata{
				{
					Name: "id",
				},
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		tables, names := buildTablesMetadata(tc.input, nil /* tableCatalog */, true /* isDetailCaseSensitive */)

		// The length of the tables should be the same as the length of the names.
		a.Equal(len(tables), len(names))

		// The names should be the same as the expected names.
		a.Equal(sort.StringSlice(names), sort.StringSlice(tc.wantNames))

		// Each table should have the same columns as the input.
		for _, table := range tables {
			a.Equal(len(table.GetProto().GetColumns()), len(tc.wantColumns))
			for _, column := range tc.wantColumns {
				a.NotNil(table.GetColumn(column.Name))
			}
		}
	}
}

func TestSchemaMetadata_CreateTable(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")

	// Create a new table
	tableMeta, err := schemaMeta.CreateTable("products")

	require.Nil(t, err)
	require.NotNil(t, tableMeta)
	require.Equal(t, "products", tableMeta.GetProto().Name)

	// Verify table is now accessible via GetTable
	retrieved := schemaMeta.GetTable("products")
	require.NotNil(t, retrieved)
	require.Equal(t, "products", retrieved.GetProto().Name)
}

func TestSchemaMetadata_CreateTable_AlreadyExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")

	// Try to create table that already exists
	_, err := schemaMeta.CreateTable("users")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestSchemaMetadata_DropTable(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
					{Name: "products"},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")

	// Drop table
	err := schemaMeta.DropTable("users")

	require.Nil(t, err)

	// Verify table is gone
	retrieved := schemaMeta.GetTable("users")
	require.Nil(t, retrieved)

	// Verify other table still exists
	retrieved = schemaMeta.GetTable("products")
	require.NotNil(t, retrieved)
}

func TestSchemaMetadata_DropTable_NotExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{Name: "public"},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")

	// Try to drop non-existent table
	err := schemaMeta.DropTable("nonexistent")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestTableMetadata_CreateColumn(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")
	tableMeta := schemaMeta.GetTable("users")

	// Create a new column
	columnProto := &storepb.ColumnMetadata{
		Name:     "email",
		Type:     "varchar",
		Nullable: true,
	}
	err := tableMeta.CreateColumn(columnProto, nil /* columnCatalog */)

	require.Nil(t, err)

	// Verify column is now accessible
	retrieved := tableMeta.GetColumn("email")
	require.NotNil(t, retrieved)
	require.Equal(t, "email", retrieved.GetProto().Name)
	require.Equal(t, "varchar", retrieved.GetProto().Type)
}

func TestTableMetadata_CreateColumn_AlreadyExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")
	tableMeta := schemaMeta.GetTable("users")

	// Try to create column that already exists
	columnProto := &storepb.ColumnMetadata{
		Name: "id",
		Type: "bigint",
	}
	err := tableMeta.CreateColumn(columnProto, nil /* columnCatalog */)

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestTableMetadata_DropColumn(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "email", Type: "varchar"},
							{Name: "name", Type: "varchar"},
						},
					},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")
	tableMeta := schemaMeta.GetTable("users")

	// Drop column
	err := tableMeta.DropColumn("email")

	require.Nil(t, err)

	// Verify column is gone
	retrieved := tableMeta.GetColumn("email")
	require.Nil(t, retrieved)

	// Verify other columns still exist
	require.NotNil(t, tableMeta.GetColumn("id"))
	require.NotNil(t, tableMeta.GetColumn("name"))
}

func TestTableMetadata_DropColumn_NotExists(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{Name: "users"},
				},
			},
		},
	}

	schema := NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_POSTGRES, true)
	schemaMeta := schema.GetSchemaMetadata("public")
	tableMeta := schemaMeta.GetTable("users")

	// Try to drop non-existent column
	err := tableMeta.DropColumn("nonexistent")

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "does not exist")
}
