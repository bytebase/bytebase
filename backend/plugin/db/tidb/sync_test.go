package tidb

import (
	"context"
	"fmt"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestSyncDBSchemaReadsForeignKeyReferencedDatabase keeps the expectation the
// old sync test held on getForeignKeyList, read through SyncDBSchema now that the
// sync queries are omni's: a foreign key keeps its referenced database only when
// it points outside the synced one.
func TestSyncDBSchemaReadsForeignKeyReferencedDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	other := database + "_other"
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
CREATE DATABASE %[2]s;
CREATE TABLE %[2]s.parent (id INT PRIMARY KEY);
CREATE TABLE %[1]s.parent (id INT PRIMARY KEY);
CREATE TABLE %[1]s.child (
	id INT PRIMARY KEY,
	local_id INT,
	qualified_id INT,
	remote_id INT,
	CONSTRAINT fk_local FOREIGN KEY (local_id) REFERENCES %[1]s.parent (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
	CONSTRAINT fk_qualified FOREIGN KEY (qualified_id) REFERENCES %[1]s.parent (id) ON DELETE SET NULL ON UPDATE RESTRICT,
	CONSTRAINT fk_remote FOREIGN KEY (remote_id) REFERENCES %[2]s.parent (id) ON DELETE CASCADE ON UPDATE RESTRICT
);
`, database, other))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := container.GetDB().ExecContext(context.Background(), fmt.Sprintf("DROP TABLE %s.child; DROP DATABASE %s", database, other))
		require.NoError(t, err)
	})

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	var child *metadatapb.TableMetadata
	for _, table := range metadata.GetSchemas()[0].GetTables() {
		if table.GetName() == "child" {
			child = table
		}
	}
	require.NotNil(t, child)
	require.Empty(t, cmp.Diff([]*metadatapb.ForeignKeyMetadata{
		{Name: "fk_local", Columns: []string{"local_id"}, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "RESTRICT", OnUpdate: "RESTRICT", MatchType: "NONE"},
		{Name: "fk_qualified", Columns: []string{"qualified_id"}, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "SET NULL", OnUpdate: "RESTRICT", MatchType: "NONE"},
		{Name: "fk_remote", Columns: []string{"remote_id"}, ReferencedSchema: other, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "CASCADE", OnUpdate: "RESTRICT", MatchType: "NONE"},
	}, child.GetForeignKeys(), protocmp.Transform()))
}

// TestSyncInstanceReportsTheTiDBVersion keeps the verbatim VERSION() text and
// the instance-level facts the product stores: lower_case_table_names and the
// databases without their schemas.
func TestSyncInstanceReportsTheTiDBVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	instance, err := driver.SyncInstance(ctx)
	require.NoError(t, err)
	require.Contains(t, instance.Version, "TiDB")
	require.NotNil(t, instance.Metadata)
	var lowerCaseTableNames int32
	require.NoError(t, container.GetDB().QueryRowContext(ctx, "SELECT @@lower_case_table_names").Scan(&lowerCaseTableNames))
	require.Equal(t, lowerCaseTableNames, instance.Metadata.GetMysqlLowerCaseTableNames())

	var names []string
	for _, db := range instance.Databases {
		names = append(names, db.GetName())
		require.Empty(t, db.GetSchemas(), "an instance sync lists databases without their schemas")
	}
	require.Contains(t, names, database)
	// A MySQL-family system database is not a user database.
	require.NotContains(t, names, "information_schema")
	require.NotContains(t, names, "metrics_schema")
}

// TestSyncDBSchemaReadsATiDBTable pins the TiDB-specific facts the old sync read:
// the sharding info, the primary key type, a generated column, a check
// constraint, an expression index and a partition's ordinal order.
func TestSyncDBSchemaReadsATiDBTable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.sharded (
			id BIGINT PRIMARY KEY AUTO_RANDOM,
			note VARCHAR(64) NOT NULL DEFAULT 'n',
			upper_note VARCHAR(64) AS (UPPER(note)) VIRTUAL
		);
		CREATE INDEX sharded_note ON %[1]s.sharded ((LOWER(note)));
		CREATE TABLE %[1]s.parted (
			id INT NOT NULL,
			PRIMARY KEY (id)
		) PARTITION BY RANGE (id) (
			PARTITION p10 VALUES LESS THAN (10),
			PARTITION p2 VALUES LESS THAN (20),
			PARTITION p1 VALUES LESS THAN (30)
		);
	`, database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	tables := map[string]*metadatapb.TableMetadata{}
	for _, table := range metadata.GetSchemas()[0].GetTables() {
		tables[table.GetName()] = table
	}

	sharded := tables["sharded"]
	require.NotNil(t, sharded)
	require.Contains(t, sharded.GetShardingInfo(), "PK_AUTO_RANDOM_BITS")
	require.Equal(t, "CLUSTERED", sharded.GetPrimaryKeyType())
	var virtual string
	for _, column := range sharded.GetColumns() {
		if column.GetName() == "id" {
			// The default TiDB reports for a bare AUTO_RANDOM primary key.
			require.Equal(t, "AUTO_RANDOM(5)", column.GetDefault())
		}
		if column.GetName() == "upper_note" {
			require.NotNil(t, column.GetGeneration())
			require.Equal(t, metadatapb.GenerationMetadata_TYPE_VIRTUAL, column.GetGeneration().GetType())
			virtual = column.GetGeneration().GetExpression()
		}
	}
	require.Equal(t, "upper(`note`)", virtual)
	// TiDB parses a CHECK constraint but does not expose or enforce one, so the
	// sync has none to read on this server.
	require.Empty(t, sharded.GetCheckConstraints())
	var expressions []string
	for _, index := range sharded.GetIndexes() {
		if index.GetName() == "sharded_note" {
			expressions = index.GetExpressions()
		}
	}
	require.Equal(t, []string{"(lower(`note`))"}, expressions)

	parted := tables["parted"]
	require.NotNil(t, parted)
	var partitions []string
	for _, partition := range parted.GetPartitions() {
		partitions = append(partitions, partition.GetName())
	}
	// Known difference: the sync reads partitions in ordinal order, where the
	// old driver sorted them by name (p1, p10, p2) and a dump of that order
	// could not be restored.
	require.Equal(t, []string{"p10", "p2", "p1"}, partitions)
}

// TestSyncDBSchemaReportsAMissingDatabase keeps the product's not-found error for
// a database the server no longer has.
func TestSyncDBSchemaReportsAMissingDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriverOn(ctx, t, container, database+"_missing")
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	_, err := driver.SyncDBSchema(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("database %q not found", database+"_missing"))
}

// TestSyncDBSchemaReadsColumnDefaults ports the expectations the old driver's
// setColumnMetadataDefault unit test held, read through SyncDBSchema now that the
// mapping is omni's: a nullable column with no default is NULL, a literal keeps
// its unquoted text, a CURRENT_TIMESTAMP default keeps its precision, and an
// ON UPDATE clause is reported beside the default.
func TestSyncDBSchemaReadsColumnDefaults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.hello (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name0 varchar(20),
			name1 varchar(20) DEFAULT '0',
			name2 varchar(20) DEFAULT 'hello',
			age0 int NOT NULL,
			age1 tinyint NOT NULL DEFAULT '0',
			price double(16,2) DEFAULT '0.00',
			time0 datetime,
			time1 datetime NOT NULL,
			time2 datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			time3 datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			time4 datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			time5 datetime(6) DEFAULT CURRENT_TIMESTAMP(6),
			time6 datetime(6) DEFAULT NOW(6),
			time7 datetime DEFAULT CURRENT_TIMESTAMP()
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='a comment';
	`, database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)

	table := metadata.GetSchemas()[0].GetTables()[0]
	require.Equal(t, "hello", table.GetName())
	// The charset is the collation's prefix, and a table comment loses the
	// quotes the sync query put around it.
	require.Equal(t, "utf8mb4_bin", table.GetCollation())
	require.Equal(t, "utf8mb4", table.GetCharset())
	require.Equal(t, "a comment", table.GetComment())

	// A literal default keeps the quotes QUOTE() put around it, which is what
	// the old driver stored too: the old unit test fed setColumnMetadataDefault an
	// already unquoted value, so its "0" was not what the sync query produces.
	want := map[string]struct{ defaultText, onUpdate string }{
		"id":    {defaultText: "AUTO_INCREMENT"},
		"name0": {defaultText: "NULL"},
		"name1": {defaultText: "'0'"},
		"name2": {defaultText: "'hello'"},
		"age0":  {},
		"age1":  {defaultText: "'0'"},
		"price": {defaultText: "'0'"},
		"time0": {defaultText: "NULL"},
		"time1": {},
		"time2": {defaultText: "CURRENT_TIMESTAMP"},
		"time3": {defaultText: "CURRENT_TIMESTAMP", onUpdate: "CURRENT_TIMESTAMP"},
		"time4": {defaultText: "CURRENT_TIMESTAMP", onUpdate: "CURRENT_TIMESTAMP"},
		"time5": {defaultText: "CURRENT_TIMESTAMP(6)"},
		"time6": {defaultText: "CURRENT_TIMESTAMP(6)"},
		"time7": {defaultText: "CURRENT_TIMESTAMP"},
	}
	require.Len(t, table.GetColumns(), len(want))
	for _, column := range table.GetColumns() {
		expected, ok := want[column.GetName()]
		require.True(t, ok, column.GetName())
		require.Equal(t, expected.defaultText, column.GetDefault(), column.GetName())
		require.Equal(t, expected.onUpdate, column.GetOnUpdate(), column.GetName())
	}
}
