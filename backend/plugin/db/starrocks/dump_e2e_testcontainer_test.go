package starrocks

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestDump_StarRocksMaterializedViewRoundTrip locks the two dump fixes (BYT-9689) against a
// real StarRocks (amd64-only; skipped in -short). A materialized view defined on a regular
// view is dumped, then replayed into a fresh database of the same name, and must come back:
//   - without a name collision (P1: the temporary placeholder is dropped as a view, so the
//     real CREATE MATERIALIZED VIEW does not collide with a leftover placeholder), and
//   - IS_ACTIVE=true (P2: the MV is emitted after its source view, so it is created against
//     the real view, not inactivated when a placeholder is dropped under it).
//
// The dump is schema-only (--no-data), so IS_ACTIVE — not row data — is the P2 signal.
func TestDump_StarRocksMaterializedViewRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping StarRocks testcontainer test in short mode")
	}
	ctx := context.Background()
	c := testcontainer.GetTestStarRocksContainer(ctx, t)
	defer c.Close(ctx)
	db := c.GetDB()

	for _, stmt := range []string{
		"CREATE DATABASE rt",
		"CREATE TABLE rt.t (id INT, amt INT) DISTRIBUTED BY HASH(id) BUCKETS 1 PROPERTIES('replication_num'='1')",
		"INSERT INTO rt.t VALUES (1,10),(2,20)",
		"CREATE VIEW rt.vw AS SELECT id, amt FROM rt.t",
		"CREATE MATERIALIZED VIEW rt.mv REFRESH ASYNC AS SELECT id, sum(amt) AS total FROM rt.vw GROUP BY id",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("setup failed: %s: %v", stmt, err)
		}
	}
	requireMVActive(ctx, t, db, "rt", "mv")

	// Produce the dump through the real driver.
	d := &Driver{dbType: storepb.Engine_STARROCKS, db: db, databaseName: "rt"}
	var buf bytes.Buffer
	require.NoError(t, d.Dump(ctx, &buf, nil))
	dump := buf.String()
	require.Contains(t, dump, "CREATE MATERIALIZED VIEW", "dump should emit the MV")

	// Restore into a fresh same-name database. StarRocks qualifies view/MV references with
	// the original db, so the restore target must reuse the name. DROP+CREATE+USE+replay run
	// in one multi-statement Exec so USE applies to the unqualified statements on the same
	// pooled connection.
	restore := "DROP DATABASE IF EXISTS rt;\nCREATE DATABASE rt;\nUSE rt;\n" + dump
	if _, err := db.ExecContext(ctx, restore); err != nil {
		t.Fatalf("dump replay failed (P1 collision or unreplayable DDL): %v", err)
	}

	// P2: the restored MV must be active, not inactivated by the placeholder dance.
	requireMVActive(ctx, t, db, "rt", "mv")

	// And it re-syncs cleanly: mv as a materialized view, no stale placeholder.
	meta, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, meta.Schemas, 1)
	require.Len(t, meta.Schemas[0].MaterializedViews, 1)
	require.Equal(t, "mv", meta.Schemas[0].MaterializedViews[0].Name)
}

// requireMVActive waits until the named materialized view reports IS_ACTIVE=true.
func requireMVActive(ctx context.Context, t *testing.T, db *sql.DB, schema, name string) {
	t.Helper()
	q := fmt.Sprintf("SELECT IS_ACTIVE FROM information_schema.materialized_views WHERE TABLE_SCHEMA='%s' AND TABLE_NAME='%s'", schema, name)
	require.Eventually(t, func() bool {
		var active string
		if err := db.QueryRowContext(ctx, q).Scan(&active); err != nil {
			return false
		}
		return active == "true"
		// information_schema.materialized_views (listMaterializedViewStatus, an FE->BE RPC)
		// is unavailable for ~tens of seconds after a fresh allin1 boot ("BE host unknown")
		// before it settles, so allow a generous window.
	}, 180*time.Second, 3*time.Second, "MV %s.%s did not become active", schema, name)
}
