package oceanbase

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

// TestSyncTableNamespace covers the OceanBase objects that share the table namespace
// with base tables. A consumer can only conclude that a table name is free if sync lists every
// one of them.
func TestSyncTableNamespace(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	startNano := time.Now().UnixNano()
	database := fmt.Sprintf("sync_namespace_%d", startNano)
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, fmt.Sprintf("CREATE DATABASE `%s`", database))
	t.Cleanup(func() {
		exec(t, admin, fmt.Sprintf("DROP DATABASE `%s`", database))
	})

	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT PRIMARY KEY, a INT) COMMENT 'base'",
		"CREATE VIEW v AS SELECT id, a FROM t",
		"CREATE EXTERNAL TABLE ext (c1 INT, c2 VARCHAR(10)) LOCATION = '/tmp' FORMAT = (TYPE = 'CSV')",
		"CREATE MATERIALIZED VIEW mv COMMENT 'materialized' AS SELECT id, a FROM t",
		"CREATE INDEX idx_mv_a ON mv (a)",
		"CREATE MATERIALIZED VIEW LOG ON t WITH PRIMARY KEY (a) INCLUDING NEW VALUES",
		"CREATE SEQUENCE seq",
	)
	var container string
	require.NoError(t, raw.QueryRowContext(ctx,
		"SELECT CONTAINER_NAME FROM oceanbase.DBA_MVIEWS WHERE OWNER = ? AND MVIEW_NAME = 'mv'", database,
	).Scan(&container))

	census := oceanBaseObjectCensus(ctx, t, raw, database)
	require.ElementsMatch(t, []string{"t", "v", "ext", "mv", "mlog$_t", container}, census)
	for _, name := range census {
		_, err := raw.ExecContext(ctx, fmt.Sprintf("CREATE TABLE `%s` (z INT)", name))
		requireMySQLErrorNumber(t, err, 1050, name)
	}
	// A sequence has a namespace of its own.
	exec(t, raw, "CREATE TABLE seq (z INT)")
	exec(t, raw, "DROP TABLE seq")

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	// The container table holds a materialized view's rows; it is the view's
	// storage, so the sync lists the view and not the table.
	require.ElementsMatch(t, []string{"t", "v", "ext", "mv", "mlog$_t"}, syncedRelationNames(metadata.Schemas[0]))
	for _, table := range metadata.Schemas[0].Tables {
		require.NotEqual(t, container, table.Name)
	}
}

// TestSyncEvents pins what the sync records about an event: OceanBase rejects
// SHOW CREATE EVENT, so the driver builds the statement from
// information_schema.EVENTS and the dump writes it back.
func TestSyncEvents(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := fmt.Sprintf("sync_events_%d", time.Now().UnixNano())

	admin := server.openRaw(ctx, t, "")
	exec(t, admin, fmt.Sprintf("CREATE DATABASE `%s`", database))
	t.Cleanup(func() { exec(t, admin, fmt.Sprintf("DROP DATABASE `%s`", database)) })

	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT)",
		"CREATE EVENT e_every ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' ON COMPLETION PRESERVE DISABLE DO INSERT INTO t VALUES (1)",
		"CREATE EVENT e_at ON SCHEDULE AT '2030-06-01 12:00:00' DO BEGIN INSERT INTO t VALUES (2); DELETE FROM t WHERE id = 1; END",
	)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	schema := metadata.Schemas[0]
	require.Equal(t, []string{"t"}, syncedRelationNames(schema))
	events := schema.Events
	require.Len(t, events, 2)
	// The old sync recorded the name alone, so the dump had no event to write.
	require.Equal(t, "e_at", events[0].Name)
	require.Equal(t, "CREATE EVENT `e_at` ON SCHEDULE AT '2030-06-01 12:00:00' ON COMPLETION NOT PRESERVE DO INSERT INTO t VALUES (2);DELETE FROM t WHERE id = 1", events[0].Definition)
	require.Equal(t, "e_every", events[1].Name)
	require.Equal(t, "CREATE EVENT `e_every` ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' ON COMPLETION PRESERVE DISABLE DO INSERT INTO t VALUES (1)", events[1].Definition)
	require.Equal(t, "SYSTEM", events[1].TimeZone)
	require.Contains(t, events[1].SqlMode, "STRICT_TRANS_TABLES")
}

func TestSyncForeignKeyReferencedDatabase(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := fmt.Sprintf("sync_fk_%d", time.Now().UnixNano())
	other := database + "_other"

	admin := server.openRaw(ctx, t, "")
	for _, name := range []string{database, other} {
		exec(t, admin, fmt.Sprintf("CREATE DATABASE `%s`", name))
	}
	t.Cleanup(func() {
		for _, name := range []string{database, other} {
			exec(t, admin, fmt.Sprintf("DROP DATABASE `%s`", name))
		}
	})

	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		fmt.Sprintf("CREATE TABLE `%s`.parent (id INT PRIMARY KEY)", other),
		"CREATE TABLE parent (id INT PRIMARY KEY)",
		fmt.Sprintf(`CREATE TABLE child (
	id INT PRIMARY KEY,
	local_id INT,
	qualified_id INT,
	remote_id INT,
	CONSTRAINT fk_local FOREIGN KEY (local_id) REFERENCES parent (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
	CONSTRAINT fk_qualified FOREIGN KEY (qualified_id) REFERENCES `+"`%s`"+`.parent (id) ON DELETE SET NULL ON UPDATE RESTRICT,
	CONSTRAINT fk_remote FOREIGN KEY (remote_id) REFERENCES `+"`%s`"+`.parent (id) ON DELETE CASCADE ON UPDATE RESTRICT
)`, database, other),
	)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	var child *metadatapb.TableMetadata
	for _, table := range metadata.Schemas[0].Tables {
		if table.Name == "child" {
			child = table
		}
	}
	require.NotNil(t, child)
	require.Empty(t, cmp.Diff([]*metadatapb.ForeignKeyMetadata{
		{Name: "fk_local", Columns: []string{"local_id"}, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "RESTRICT", OnUpdate: "RESTRICT", MatchType: "NONE"},
		{Name: "fk_qualified", Columns: []string{"qualified_id"}, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "SET NULL", OnUpdate: "RESTRICT", MatchType: "NONE"},
		{Name: "fk_remote", Columns: []string{"remote_id"}, ReferencedSchema: other, ReferencedTable: "parent", ReferencedColumns: []string{"id"}, OnDelete: "CASCADE", OnUpdate: "RESTRICT", MatchType: "NONE"},
	}, child.ForeignKeys, protocmp.Transform()))
}

// OceanBase keeps listing a dropped database's events under its name, so a database created
// again under that name lists an event created again twice.
func TestSyncEventsOfRecreatedDatabase(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := fmt.Sprintf("sync_recreated_%d", time.Now().UnixNano())

	admin := server.openRaw(ctx, t, "")
	t.Cleanup(func() { exec(t, admin, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", database)) })
	var listed int
	for range 2 {
		exec(t, admin, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", database))
		exec(t, admin, fmt.Sprintf("CREATE DATABASE `%s`", database))
		driver := server.open(ctx, t, database)
		raw := server.openRaw(ctx, t, database)
		exec(t, raw,
			"CREATE TABLE t (id INT)",
			"CREATE EVENT e ON SCHEDULE EVERY 1 DAY STARTS '2030-01-01 00:00:00' DO DELETE FROM t",
		)
		require.NoError(t, admin.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.EVENTS WHERE EVENT_SCHEMA = ? AND EVENT_NAME = 'e'", database,
		).Scan(&listed))

		metadata, err := driver.SyncDBSchema(ctx)
		require.NoError(t, err)
		// The event is listed twice and synced once.
		require.Len(t, metadata.Schemas[0].Events, 1)
		require.Equal(t, "e", metadata.Schemas[0].Events[0].Name)
		require.Contains(t, metadata.Schemas[0].Events[0].Definition, "CREATE EVENT `e` ON SCHEDULE EVERY 1 DAY")
	}
	require.Equal(t, 2, listed)
}

// oceanBaseObjectCensus lists every object the catalog holds in the database except the
// database itself and its indexes.
func oceanBaseObjectCensus(ctx context.Context, t *testing.T, connection *sql.DB, database string) []string {
	t.Helper()
	rows, err := connection.QueryContext(ctx,
		"SELECT OBJECT_NAME FROM oceanbase.DBA_OBJECTS WHERE OWNER = ? AND OBJECT_TYPE NOT IN ('DATABASE', 'INDEX')", database)
	require.NoError(t, err)
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	return names
}

func syncedRelationNames(schema *metadatapb.SchemaMetadata) []string {
	var names []string
	for _, table := range schema.Tables {
		names = append(names, table.Name)
	}
	for _, view := range schema.Views {
		names = append(names, view.Name)
	}
	for _, table := range schema.ExternalTables {
		names = append(names, table.Name)
	}
	for _, view := range schema.MaterializedViews {
		names = append(names, view.Name)
	}
	slices.Sort(names)
	return names
}

func requireMySQLErrorNumber(t *testing.T, err error, number uint16, msgAndArgs ...any) {
	t.Helper()
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr, msgAndArgs...)
	require.Equal(t, number, mysqlErr.Number, msgAndArgs...)
}
