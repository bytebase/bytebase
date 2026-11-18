package oceanbase

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDumpRendersTheSyncedMetadata proves the adapter's Dump hands the synced
// metadata to omni's OceanBase catalog: the objects the driver synced are the
// objects the dump states. What the dump says about an object is the engine's
// DDL fidelity, which omni's own corpus owns (oceanbase/driver's sync golden and
// its dump round trip), not this package's.
func TestDumpRendersTheSyncedMetadata(t *testing.T) {
	server := newOceanBaseTestServer(t)
	ctx := context.Background()
	database := "dump_renders"
	admin := server.openRaw(ctx, t, "")
	exec(t, admin, "DROP DATABASE IF EXISTS "+database, "CREATE DATABASE "+database)
	t.Cleanup(func() { exec(t, admin, "DROP DATABASE IF EXISTS "+database) })
	driver := server.open(ctx, t, database)
	raw := server.openRaw(ctx, t, database)
	exec(t, raw,
		"CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(32) NOT NULL DEFAULT 'anon') COMMENT 'a table'",
		"CREATE VIEW v AS SELECT id, name FROM t",
	)

	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	var dump strings.Builder
	require.NoError(t, driver.Dump(ctx, &dump, metadata))
	text := dump.String()

	require.Contains(t, text, "CREATE TABLE `t` (")
	require.Contains(t, text, "`name` varchar(32) NOT NULL DEFAULT 'anon'")
	require.Contains(t, text, "PRIMARY KEY (`id`)")
	require.Contains(t, text, "COMMENT='a table'")
	require.Contains(t, text, "CREATE OR REPLACE VIEW `v` AS select")

	// The dump renders what it is given: a database the caller left out is not
	// in it.
	metadata.Schemas[0].Tables = nil
	dump.Reset()
	require.NoError(t, driver.Dump(ctx, &dump, metadata))
	require.NotContains(t, dump.String(), "CREATE TABLE `t`")
}
