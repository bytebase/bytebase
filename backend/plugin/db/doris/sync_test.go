package doris

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// BYT-9689: Doris reports materialized views in information_schema.tables with
// TABLE_TYPE='BASE TABLE', and some setups with 'MATERIALIZED VIEW', so the table type
// alone cannot identify one. Membership in the materialized-view set is the
// authoritative signal, whatever type the row carries.
func TestIsMaterializedView(t *testing.T) {
	mvMap := map[db.TableKey]*metadatapb.MaterializedViewMetadata{
		{Schema: "", Table: "mv_async"}: {Name: "mv_async"},
	}
	mvKey := db.TableKey{Schema: "", Table: "mv_async"}

	require.True(t, isMaterializedView(baseTableType, mvKey, mvMap))
	require.True(t, isMaterializedView(materializedViewType, mvKey, mvMap))
	require.True(t, isMaterializedView(viewTableType, mvKey, mvMap))

	// A name absent from the MV set is never a materialized view.
	require.False(t, isMaterializedView(viewTableType, db.TableKey{Table: "v_regular"}, mvMap))
	require.False(t, isMaterializedView(baseTableType, db.TableKey{Table: "sales"}, mvMap))
}

// #20566 review (P2): SyncInstance's system-database exclusion must match the parser's
// system set. Doris has no `sys` metadatabase, so excluding it would risk hiding a user
// database named sys.
func TestSystemDatabaseExclusion(t *testing.T) {
	got := systemDatabaseExclusion()
	require.Contains(t, got, "'information_schema'")
	require.Contains(t, got, "'_statistics_'")
	require.NotContains(t, got, "'sys'")
}
