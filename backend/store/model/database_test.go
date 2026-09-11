package model

import (
	"sort"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestBuildTablesMetadata(t *testing.T) {
	testCases := []struct {
		input       *metadatapb.TableMetadata
		wantNames   []string
		wantColumns []*metadatapb.ColumnMetadata
	}{
		// No partitions.
		{
			input: &metadatapb.TableMetadata{
				Name: "orders",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name: "id",
					},
				},
			},
			wantNames: []string{"orders"},
			wantColumns: []*metadatapb.ColumnMetadata{
				{
					Name: "id",
				},
			},
		},
		// Nested partitions.
		{
			input: &metadatapb.TableMetadata{
				Name: "orders",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name: "id",
					},
				},
				Partitions: []*metadatapb.TablePartitionMetadata{
					{
						Name: "orders_0_100",
						Subpartitions: []*metadatapb.TablePartitionMetadata{
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
						Subpartitions: []*metadatapb.TablePartitionMetadata{
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
			wantColumns: []*metadatapb.ColumnMetadata{
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

func TestPartitionTable_CreateIndex(t *testing.T) {
	a := require.New(t)

	// Create a table with partitions
	tableProto := &metadatapb.TableMetadata{
		Name: "orders",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id"},
			{Name: "created_at"},
		},
		Partitions: []*metadatapb.TablePartitionMetadata{
			{Name: "orders_2024"},
			{Name: "orders_2025"},
		},
	}

	tables, names := buildTablesMetadata(tableProto, nil, true)
	a.Equal(3, len(tables))
	a.ElementsMatch([]string{"orders", "orders_2024", "orders_2025"}, names)

	// Verify that CreateIndex works on partition tables (was causing nil map panic)
	for i, table := range tables {
		index := &metadatapb.IndexMetadata{
			Name:        "idx_" + names[i] + "_created_at",
			Expressions: []string{"created_at"},
			Type:        "btree",
		}
		err := table.CreateIndex(index)
		a.NoError(err, "CreateIndex should not panic on table %s", names[i])
		a.NotNil(table.GetIndex(index.Name), "Index should be retrievable after creation")
	}
}

func TestSchemaMetadata_CreateTable(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
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
	columnProto := &metadatapb.ColumnMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
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
	columnProto := &metadatapb.ColumnMetadata{
		Name: "id",
		Type: "bigint",
	}
	err := tableMeta.CreateColumn(columnProto, nil /* columnCatalog */)

	require.NotNil(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestTableMetadata_DropColumn(t *testing.T) {
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "users",
						Columns: []*metadatapb.ColumnMetadata{
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
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
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

func TestListTableNames_ExcludesPartitions(t *testing.T) {
	a := require.New(t)

	// A range-partitioned table whose last partition is itself subpartitioned. Every
	// partition and subpartition is registered in internalTables under its own name but
	// carries the parent's proto, so listing them once yielded "sales" six times.
	sales := &metadatapb.TableMetadata{
		Name:    "sales",
		Columns: []*metadatapb.ColumnMetadata{{Name: "dt"}},
		Partitions: []*metadatapb.TablePartitionMetadata{
			{Name: "p0", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
			{Name: "p1", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
			{
				Name: "p2", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt",
				Subpartitions: []*metadatapb.TablePartitionMetadata{
					{Name: "p2s0", Type: metadatapb.TablePartitionMetadata_HASH, Expression: "dt"},
					{Name: "p2s1", Type: metadatapb.TablePartitionMetadata_HASH, Expression: "dt"},
				},
			},
		},
	}
	customers := &metadatapb.TableMetadata{
		Name:    "customers",
		Columns: []*metadatapb.ColumnMetadata{{Name: "id"}},
	}

	dbMetadata := NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{sales, customers}}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)
	schemaMetadata := dbMetadata.GetSchemaMetadata("")

	a.Equal([]string{"customers", "sales"}, schemaMetadata.ListTableNames())

	// Partitions stay resolvable by name; only the listing changed.
	for _, partitionName := range []string{"p0", "p1", "p2", "p2s0", "p2s1"} {
		a.NotNil(schemaMetadata.GetTable(partitionName), "partition %s should still resolve", partitionName)
	}
	a.NotNil(schemaMetadata.GetTable("sales"))
}

// Partition names live in a namespace separate from table names, so a partition may carry
// its own table's name. Both land in internalTables under the same key and the partition
// wrapper wins, so the root table has to be listed from the schema's own table list rather
// than inferred from that map.
func TestListTableNames_PartitionSharingTableName(t *testing.T) {
	a := require.New(t)

	dbMetadata := NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{
				Name:    "sales",
				Columns: []*metadatapb.ColumnMetadata{{Name: "dt"}},
				Partitions: []*metadatapb.TablePartitionMetadata{
					{Name: "sales", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
					{Name: "p1", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
				},
			},
		}}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)

	a.Equal([]string{"sales"}, dbMetadata.GetSchemaMetadata("").ListTableNames())
}

// A partition of one table may share the name of a different root table. The root listing
// and the root lookup must agree: resolving that name has to return the root table's own
// proto, not the partition owner's, or the differ generates one table using another's
// columns and indexes.
func TestGetTable_PartitionAliasDoesNotShadowRootTable(t *testing.T) {
	a := require.New(t)

	dbMetadata := NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{
				Name:    "archive",
				Columns: []*metadatapb.ColumnMetadata{{Name: "archive_col"}},
			},
			{
				Name:    "events",
				Columns: []*metadatapb.ColumnMetadata{{Name: "events_col"}},
				Partitions: []*metadatapb.TablePartitionMetadata{
					{Name: "archive", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
					{Name: "recent", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
				},
			},
		}}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)
	schemaMetadata := dbMetadata.GetSchemaMetadata("")

	a.Equal([]string{"archive", "events"}, schemaMetadata.ListTableNames())

	// Every listed name must resolve to that same table.
	for _, name := range schemaMetadata.ListTableNames() {
		a.Equal(name, schemaMetadata.GetTable(name).GetProto().GetName(),
			"GetTable(%q) resolved to the wrong table", name)
	}
	a.NotNil(schemaMetadata.GetTable("archive").GetColumn("archive_col"))

	// A partition name that is not also a root table still resolves to its owner.
	a.Equal("events", schemaMetadata.GetTable("recent").GetProto().GetName())
}

// A partition name does not reserve a table name, and dropping a partition by name is not
// dropping a table. Table creation and drops therefore have to consult real tables only.
func TestTableMutationsIgnorePartitionAliases(t *testing.T) {
	a := require.New(t)

	dbMetadata := NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{
			{
				Name:    "events",
				Columns: []*metadatapb.ColumnMetadata{{Name: "events_col"}},
				Partitions: []*metadatapb.TablePartitionMetadata{
					{Name: "archive", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"},
				},
			},
		}}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)
	schemaMetadata := dbMetadata.GetSchemaMetadata("")

	a.ErrorContains(schemaMetadata.DropTable("archive"), "does not exist")

	created, err := schemaMetadata.CreateTable("archive")
	a.NoError(err)
	a.Equal("archive", created.GetProto().GetName())
	a.Equal([]string{"archive", "events"}, schemaMetadata.ListTableNames())
	a.Equal("archive", schemaMetadata.GetTable("archive").GetProto().GetName())
}
