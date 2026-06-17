package starrocks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestSyncDBSchema_StarRocksMaterializedView_E2E brings up a real StarRocks (amd64-only;
// runs in CI, skipped in -short) and verifies BYT-9689 end to end: an async materialized
// view is synced under MaterializedViews with its definition, while a synchronous rollup
// (REFRESH_TYPE='ROLLUP', which has no information_schema.tables row) is excluded.
func TestSyncDBSchema_StarRocksMaterializedView_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping StarRocks testcontainer test in short mode")
	}
	ctx := context.Background()
	c := testcontainer.GetTestStarRocksContainer(ctx, t)
	defer c.Close(ctx)
	db := c.GetDB()

	for _, stmt := range []string{
		"CREATE DATABASE mvtest",
		"CREATE TABLE mvtest.sales (id BIGINT, amt INT) DISTRIBUTED BY HASH(id) BUCKETS 1 PROPERTIES('replication_num'='1')",
		"INSERT INTO mvtest.sales VALUES (1,10),(2,20)",
		"CREATE MATERIALIZED VIEW mvtest.mv_async REFRESH ASYNC AS SELECT id, sum(amt) AS total FROM mvtest.sales GROUP BY id",
		"CREATE MATERIALIZED VIEW mvtest.mv_rollup AS SELECT id, sum(amt) FROM mvtest.sales GROUP BY id",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("setup failed: %s: %v", stmt, err)
		}
	}

	// The sync rollup builds asynchronously; wait until it registers in
	// information_schema.materialized_views so the test deterministically exercises the
	// rollup-exclusion path (it appears there as REFRESH_TYPE='ROLLUP' but never as a
	// information_schema.tables row).
	require.Eventually(t, func() bool {
		var n int
		if err := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM information_schema.materialized_views WHERE TABLE_SCHEMA='mvtest' AND REFRESH_TYPE='ROLLUP'").Scan(&n); err != nil {
			return false
		}
		return n == 1
		// information_schema.materialized_views (an FE->BE RPC) is unavailable for tens of
		// seconds after a fresh allin1 boot before it settles; allow a generous window.
	}, 150*time.Second, 3*time.Second, "sync rollup did not register in materialized_views")

	d := &Driver{dbType: storepb.Engine_STARROCKS, db: db, databaseName: "mvtest"}
	meta, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Len(t, meta.Schemas, 1)
	schema := meta.Schemas[0]

	// The async MV is synced under MaterializedViews with its definition.
	require.Len(t, schema.MaterializedViews, 1, "exactly the async MV must be synced")
	require.Equal(t, "mv_async", schema.MaterializedViews[0].Name)
	require.Contains(t, schema.MaterializedViews[0].Definition, "CREATE MATERIALIZED VIEW")

	// The base table is synced as a table; the sync rollup leaks nowhere; the MV is not
	// misfiled as a regular view.
	require.Len(t, schema.Tables, 1)
	require.Equal(t, "sales", schema.Tables[0].Name)
	for _, mv := range schema.MaterializedViews {
		require.NotEqual(t, "mv_rollup", mv.Name, "sync rollup must be excluded from MaterializedViews")
	}
	for _, v := range schema.Views {
		require.NotEqual(t, "mv_async", v.Name, "MV must not be misfiled as a regular view")
	}
}
