package mysql

import (
	"fmt"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"os"

	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

type generateMigrationCase struct {
	Description string `yaml:"description"`
	OldSchema   string `yaml:"oldSchema"`
	NewSchema   string `yaml:"newSchema"`
	Expected    string `yaml:"expected"`
}

// TestGenerateMigration pins the DDL that takes oldSchema to newSchema. Cases come
// in both directions for each schema pair, so an object appears once as a create
// and once as a drop.
func TestGenerateMigration(t *testing.T) {
	const (
		record   = false
		filepath = "testdata/generate_migration.yaml"
	)

	var tests []generateMigrationCase
	content, err := os.ReadFile(filepath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(content, &tests))

	for i, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MYSQL,
				parseSchema(t, tc.OldSchema), parseSchema(t, tc.NewSchema))
			require.NoError(t, err)

			migration, err := schema.GenerateMigration(storepb.Engine_MYSQL, diff)
			require.NoError(t, err)

			if record {
				tests[i].Expected = migration
				return
			}
			require.Equal(t, tc.Expected, migration)
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

// parseSchema reads a schema text into the model the differ takes. Empty text is
// an empty database rather than a parse, which is how create-from-nothing and
// drop-everything cases are written.
func parseSchema(t *testing.T, text string) *model.DatabaseMetadata {
	t.Helper()

	metadata := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{}}}
	if text != "" {
		parsed, err := GetDatabaseMetadata(text)
		require.NoError(t, err)
		metadata = parsed
	}
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, false)
}

func partitionedTable(partitionCount int) *metadatapb.TableMetadata {
	table := &metadatapb.TableMetadata{
		Name:      "ads_member_asset_by_account_channel_d",
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_general_ci",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "dt", Type: "date", Nullable: false},
			{Name: "member_id", Type: "bigint", Nullable: false},
			{Name: "account_type", Type: "varchar(50)", Nullable: false},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"dt", "member_id", "account_type"}, Primary: true, Unique: true, Visible: true, Type: "BTREE"},
			{Name: "sindex", Expressions: []string{"dt", "member_id", "account_type"}, Visible: true, Type: "BTREE"},
		},
	}
	for i := range partitionCount {
		table.Partitions = append(table.Partitions, &metadatapb.TablePartitionMetadata{
			Name:       fmt.Sprintf("p%d", i),
			Type:       metadatapb.TablePartitionMetadata_RANGE,
			Expression: "to_days(`dt`)",
			Value:      fmt.Sprintf("%d", 739000+i),
		})
	}
	return table
}
func databaseOf(tables ...*metadatapb.TableMetadata) *model.DatabaseMetadata {
	return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: tables}},
	}, nil, &storepb.DatabaseConfig{}, storepb.Engine_MYSQL, true)
}

func TestDiffMigrationEventWithoutDefinition(t *testing.T) {
	oceanBaseDatabaseOf := func(events ...*metadatapb.EventMetadata) *model.DatabaseMetadata {
		return model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
			Name:    "db",
			Schemas: []*metadatapb.SchemaMetadata{{Name: "", Events: events}},
		}, nil, &storepb.DatabaseConfig{}, storepb.Engine_OCEANBASE, true)
	}
	withEvent := oceanBaseDatabaseOf(&metadatapb.EventMetadata{Name: "e"})

	migration, err := schema.DiffMigration(storepb.Engine_OCEANBASE, oceanBaseDatabaseOf(), withEvent)
	require.NoError(t, err)
	require.Empty(t, migration)

	migration, err = schema.DiffMigration(storepb.Engine_OCEANBASE, withEvent, oceanBaseDatabaseOf())
	require.NoError(t, err)
	require.Equal(t, "DROP EVENT IF EXISTS `e`;\n\n", migration)
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
			bare := &metadatapb.TableMetadata{
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
		a.Contains(migration, "PARTITION `"+partition.Name+"`", "partition %s must be created", partition.Name)
	}
}

// MySQL rejects a partitioned table whose partitions declare a different storage engine
// than the table itself: "ERROR 1497 (HY000): The mix of handlers in the partitions is
// not allowed in this version of MySQL". Partitioning a non-InnoDB table is still legal
// on MySQL 5.7, so the partition clause has to carry the table's own engine.
func TestDiffMigrationPartitionEngineMatchesTable(t *testing.T) {
	for _, engine := range []string{"InnoDB", "MyISAM", "ARCHIVE"} {
		t.Run(engine, func(t *testing.T) {
			a := require.New(t)
			source := partitionedTable(2)
			source.Engine = engine

			migration, err := schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(), databaseOf(source))
			a.NoError(err)
			a.Equal(len(source.Partitions), strings.Count(migration, "PARTITION `p"),
				"every partition must be emitted:\n%s", migration)
			a.Equal(len(source.Partitions)+1, strings.Count(migration, "ENGINE="+engine),
				"the table and every partition must declare the same engine:\n%s", migration)
		})
	}
}

// Partition names are identifiers, so a reserved word is legal when quoted at creation and
// comes back from information_schema unquoted. MySQL's own SHOW CREATE TABLE quotes them,
// and the regenerated DDL has to as well or it is a syntax error.
func TestDiffMigrationQuotesPartitionNames(t *testing.T) {
	a := require.New(t)
	source := partitionedTable(0)
	source.Partitions = []*metadatapb.TablePartitionMetadata{
		{Name: "select", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "to_days(`dt`)", Value: "739000"},
		{Name: "order", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "to_days(`dt`)", Value: "MAXVALUE"},
	}

	migration, err := schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(), databaseOf(source))
	a.NoError(err)
	a.Contains(migration, "PARTITION `select`", "reserved-word partition name must be quoted:\n%s", migration)
	a.Contains(migration, "PARTITION `order`", "reserved-word partition name must be quoted:\n%s", migration)
}

// tableAction returns the action the diff recorded for one table, or "" if absent.
func tableAction(diff *schema.MetadataDiff, name string) schema.MetadataDiffAction {
	for _, tableDiff := range diff.TableChanges {
		if tableDiff.TableName == name {
			return tableDiff.Action
		}
	}
	return ""
}

// The differ decides whether a table exists in the other snapshot by name. A partition of
// some other table may carry that name, and resolving it as if it were the table makes the
// differ miss a drop and mistake a create for a modification of the partition's owner.
func TestDiffMigrationPartitionAliasDoesNotMaskTableExistence(t *testing.T) {
	archive := &metadatapb.TableMetadata{
		Name: "archive", Engine: "InnoDB",
		Columns: []*metadatapb.ColumnMetadata{{Name: "archive_col", Type: "int"}},
	}
	events := func(partitions ...*metadatapb.TablePartitionMetadata) *metadatapb.TableMetadata {
		return &metadatapb.TableMetadata{
			Name: "events", Engine: "InnoDB",
			Columns:    []*metadatapb.ColumnMetadata{{Name: "events_col", Type: "int"}},
			Partitions: partitions,
		}
	}
	alias := &metadatapb.TablePartitionMetadata{Name: "archive", Type: metadatapb.TablePartitionMetadata_RANGE, Expression: "dt"}

	t.Run("drop", func(t *testing.T) {
		a := require.New(t)
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MYSQL,
			databaseOf(archive, events()), databaseOf(events(alias)))
		a.NoError(err)
		a.Equal(schema.MetadataDiffActionDrop, tableAction(diff, "archive"))
	})

	t.Run("create", func(t *testing.T) {
		a := require.New(t)
		diff, err := schema.GetDatabaseSchemaDiff(storepb.Engine_MYSQL,
			databaseOf(events(alias)), databaseOf(archive, events()))
		a.NoError(err)
		a.Equal(schema.MetadataDiffActionCreate, tableAction(diff, "archive"))
	})
}

// A MariaDB table that loses WITH SYSTEM VERSIONING silently stops keeping history, and
// explicit ROW START/ROW END columns are rejected (4125) without both that option and a
// PERIOD FOR SYSTEM_TIME element.
func TestDiffMigrationKeepsSystemVersioning(t *testing.T) {
	a := require.New(t)
	implicit := &metadatapb.TableMetadata{
		Name:          "history",
		Engine:        "InnoDB",
		CreateOptions: "WITH SYSTEM VERSIONING",
		Columns:       []*metadatapb.ColumnMetadata{{Name: "id", Type: "int", Nullable: false}},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true, Visible: true, Type: "BTREE"},
		},
	}
	explicit := &metadatapb.TableMetadata{
		Name:          "explicit_period",
		Engine:        "InnoDB",
		CreateOptions: "WITH SYSTEM VERSIONING",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "int", Nullable: true},
			{Name: "valid_from", Type: "timestamp(6)", Generation: &metadatapb.GenerationMetadata{Type: metadatapb.GenerationMetadata_TYPE_STORED, Expression: "ROW START"}},
			{Name: "valid_to", Type: "timestamp(6)", Generation: &metadatapb.GenerationMetadata{Type: metadatapb.GenerationMetadata_TYPE_STORED, Expression: "ROW END"}},
		},
	}

	migration, err := schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(), databaseOf(implicit, explicit))
	a.NoError(err)
	a.Equal("CREATE TABLE IF NOT EXISTS `explicit_period` (\n"+
		"  `id` int,\n"+
		"  `valid_from` timestamp(6) GENERATED ALWAYS AS ROW START NOT NULL,\n"+
		"  `valid_to` timestamp(6) GENERATED ALWAYS AS ROW END NOT NULL,\n"+
		"  PERIOD FOR SYSTEM_TIME (`valid_from`, `valid_to`)\n"+
		") ENGINE=InnoDB WITH SYSTEM VERSIONING;\n"+
		"\n"+
		"CREATE TABLE IF NOT EXISTS `history` (\n"+
		"  `id` int NOT NULL,\n"+
		"  PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB WITH SYSTEM VERSIONING;\n"+
		"\n", migration)

	migration, err = schema.DiffMigration(storepb.Engine_MYSQL, databaseOf(implicit, explicit), databaseOf(implicit, explicit))
	a.NoError(err)
	a.Empty(migration)
}
