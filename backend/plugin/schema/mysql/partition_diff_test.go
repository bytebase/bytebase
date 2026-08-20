package mysql

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// partitionedTable is the shape reported in BYT-10057: a daily BI table with a composite
// primary key, a secondary index over the same ordered columns, and one range partition
// per day. Each partition used to add another copy of the table to the diff, so the
// generated script repeated the whole DDL block once per partition and died on the second
// copy with "Duplicate key name" (1061) or "Multiple primary key defined" (1068).
func partitionedTable(partitionCount int) *storepb.TableMetadata {
	table := &storepb.TableMetadata{
		Name:      "ads_member_asset_by_account_channel_d",
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_general_ci",
		Columns: []*storepb.ColumnMetadata{
			{Name: "dt", Type: "date", Nullable: false},
			{Name: "member_id", Type: "bigint", Nullable: false},
			{Name: "account_type", Type: "varchar(50)", Nullable: false},
		},
		Indexes: []*storepb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"dt", "member_id", "account_type"}, Primary: true, Unique: true, Visible: true, Type: "BTREE"},
			{Name: "sindex", Expressions: []string{"dt", "member_id", "account_type"}, Visible: true, Type: "BTREE"},
		},
	}
	for i := range partitionCount {
		table.Partitions = append(table.Partitions, &storepb.TablePartitionMetadata{
			Name:       fmt.Sprintf("p%d", i),
			Type:       storepb.TablePartitionMetadata_RANGE,
			Expression: "to_days(`dt`)",
			Value:      fmt.Sprintf("%d", 739000+i),
		})
	}
	return table
}

func databaseOf(tables ...*storepb.TableMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*storepb.SchemaMetadata{{Name: "", Tables: tables}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)
}

func TestDiffMigrationPartitionedTableEmittedOnce(t *testing.T) {
	for _, partitionCount := range []int{0, 1, 3, 180} {
		t.Run(fmt.Sprintf("partitions=%d", partitionCount), func(t *testing.T) {
			a := require.New(t)
			source := partitionedTable(partitionCount)

			// Target does not have the table yet: the 1061 shape.
			migration, err := schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(), databaseOf(source))
			a.NoError(err)
			a.Equal(1, strings.Count(migration, "CREATE TABLE"), "CREATE TABLE must be emitted once:\n%s", migration)
			a.Equal(1, strings.Count(migration, "CREATE INDEX `sindex`"), "sindex must be emitted once:\n%s", migration)

			// Target has the table but neither the primary key nor the index: the 1068 shape.
			bare := &storepb.TableMetadata{
				Name:      source.Name,
				Engine:    source.Engine,
				Charset:   source.Charset,
				Collation: source.Collation,
				Columns:   source.Columns,
			}
			migration, err = schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(bare), databaseOf(source))
			a.NoError(err)
			a.Equal(1, strings.Count(migration, "ADD PRIMARY KEY"), "primary key must be added once:\n%s", migration)
			a.Equal(1, strings.Count(migration, "CREATE INDEX `sindex`"), "sindex must be emitted once:\n%s", migration)

			// Diffing the table against itself stays convergent.
			migration, err = schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(source), databaseOf(partitionedTable(partitionCount)))
			a.NoError(err)
			a.Empty(migration)
		})
	}
}

func TestDiffMigrationPreservesPartitioning(t *testing.T) {
	a := require.New(t)
	source := partitionedTable(3)

	migration, err := schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(), databaseOf(source))
	a.NoError(err)
	a.Contains(migration, "PARTITION BY RANGE (to_days(`dt`))", "created table must keep its partitioning:\n%s", migration)
	for _, partition := range source.Partitions {
		a.Contains(migration, "PARTITION "+partition.Name, "partition %s must be created", partition.Name)
	}
}
