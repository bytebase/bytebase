package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

	// Register the doris + starrocks editor validators.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/starrocks"
)

// TestValidateSQLForEditor_EngineDispatch guards the #20562 P2 fix: QueryConn
// must split/validate via the connection's engine (d.dbType), not a hardcoded
// Engine_DORIS. On a StarRocks-only statement the two engines must disagree —
// doris rejects it (→ allQuery forced true, DDL wrongly routed through
// QueryContext), starrocks accepts it (→ correct allQuery=false).
func TestValidateSQLForEditor_EngineDispatch(t *testing.T) {
	// StarRocks generated column AS (expr): doris wants GENERATED ALWAYS AS.
	const ddl = "CREATE TABLE t (a INT, b INT AS (a + 1)) DUPLICATE KEY(a) DISTRIBUTED BY HASH(a)"

	_, srAllQuery, srErr := base.ValidateSQLForEditor(storepb.Engine_STARROCKS, ddl)
	require.NoError(t, srErr, "starrocks parser should accept StarRocks DDL")
	require.False(t, srAllQuery, "a CREATE TABLE is DDL, not a read-only query")

	_, _, dorisErr := base.ValidateSQLForEditor(storepb.Engine_DORIS, ddl)
	require.Error(t, dorisErr, "doris parser rejects StarRocks AS(expr) DDL — why d.dbType matters")
}
