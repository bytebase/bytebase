package starrocks

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// BYT-9689: StarRocks reports materialized views in information_schema.tables with
// TABLE_TYPE='VIEW' and omits them from information_schema.views (verified on a live
// 3.4.10 BE-alive smoke test), while Doris reports them as 'BASE TABLE'. The shared
// driver previously only matched Doris's 'BASE TABLE' rows, so StarRocks MVs were
// dropped. Membership in the materialized-view set is the authoritative signal.
func TestIsMaterializedView(t *testing.T) {
	mvMap := map[db.TableKey]*metadatapb.MaterializedViewMetadata{
		{Schema: "", Table: "mv_async"}: {Name: "mv_async"},
	}
	mvKey := db.TableKey{Schema: "", Table: "mv_async"}

	// StarRocks reports the MV as VIEW; Doris as BASE TABLE; both map to an MV.
	require.True(t, isMaterializedView(viewTableType, mvKey, mvMap))
	require.True(t, isMaterializedView(baseTableType, mvKey, mvMap))
	require.True(t, isMaterializedView(materializedViewType, mvKey, mvMap))

	// A name absent from the MV set is never a materialized view.
	require.False(t, isMaterializedView(viewTableType, db.TableKey{Table: "v_regular"}, mvMap))
	require.False(t, isMaterializedView(baseTableType, db.TableKey{Table: "sales"}, mvMap))
}

// BYT-9689: StarRocks synchronous rollups are listed in
// information_schema.materialized_views with REFRESH_TYPE='ROLLUP' but, unlike async
// MVs, have no information_schema.tables row (they are rollup indexes on the base
// table). Only confirmed rollups are excluded; async MVs (ASYNC/MANUAL/...) are kept, so
// a future refresh type isn't silently dropped. The query coalesces SQL NULL to the
// empty string via IFNULL (database/sql cannot scan NULL into a string), so a NULL
// refresh type reaches this check as the empty string, which must also be kept.
func TestIsSyncRollup(t *testing.T) {
	require.True(t, isSyncRollup("ROLLUP"))
	require.False(t, isSyncRollup("ASYNC"))
	require.False(t, isSyncRollup("MANUAL"))
	require.False(t, isSyncRollup("")) // IFNULL-coalesced NULL or a genuinely empty type.
}

// #20566 review (P2): SyncInstance's system-database exclusion must match the
// parser's per-engine system set — StarRocks's read-only `sys` metadatabase
// must be excluded so it isn't imported as a user database. Doris has no `sys`,
// so excluding it there would risk hiding a user db named sys (engine-aware).
func TestSystemDatabaseExclusion(t *testing.T) {
	sr := (&Driver{dbType: storepb.Engine_STARROCKS}).systemDatabaseExclusion()
	require.Contains(t, sr, "'sys'", "StarRocks sync must exclude the sys metadatabase")
	require.Contains(t, sr, "'information_schema'")
	require.Contains(t, sr, "'_statistics_'")

	doris := (&Driver{dbType: storepb.Engine_DORIS}).systemDatabaseExclusion()
	require.NotContains(t, doris, "'sys'", "Doris must not exclude sys (StarRocks-only)")
	require.Contains(t, doris, "'information_schema'")
}
