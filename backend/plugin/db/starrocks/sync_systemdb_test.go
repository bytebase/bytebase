package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

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
