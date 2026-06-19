package mariadb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// mariaDBOnlySQL are statements valid in MariaDB but rejected by the omni/mysql
// parser that backed Engine_MARIADB before this carve-out: SEQUENCE objects,
// NEXT VALUE FOR, and RETURNING on INSERT/REPLACE/DELETE.
var mariaDBOnlySQL = []string{
	"CREATE SEQUENCE s",
	"INSERT INTO t (id) VALUES (1) RETURNING id",
	"REPLACE INTO t (id) VALUES (1) RETURNING id",
	"DELETE FROM t RETURNING id",
	"SELECT NEXT VALUE FOR s",
}

// TestMariaDBDiagnoseKeystone is the carve-out keystone. Before the re-point,
// Engine_MARIADB Diagnose ran through omni/mysql and reported a *false* syntax
// error on MariaDB-only SQL. After re-pointing to omni/mariadb it parses cleanly.
// The test contrasts both backends so the green state can't be accidental: the
// mysql backend must still flag these (the bug), the mariadb backend must not.
func TestMariaDBDiagnoseKeystone(t *testing.T) {
	ctx := context.Background()
	for _, stmt := range mariaDBOnlySQL {
		// RED baseline: the old omni/mysql backend reports a false syntax error.
		mysqlDiags, err := mysql.Diagnose(ctx, base.DiagnoseContext{}, stmt)
		require.NoError(t, err)
		require.NotEmpty(t, mysqlDiags, "omni/mysql should still (wrongly) flag %q — the false error being fixed", stmt)

		// GREEN: the omni/mariadb backend accepts it (no diagnostics).
		mariadbDiags, err := Diagnose(ctx, base.DiagnoseContext{}, stmt)
		require.NoError(t, err)
		require.Empty(t, mariadbDiags, "omni/mariadb should accept %q with no diagnostic", stmt)
	}

	// Diagnose must still report a genuine syntax error — proving it actually
	// validates, not just "returns empty".
	badDiags, err := Diagnose(ctx, base.DiagnoseContext{}, "SELECT FROM WHERE )(")
	require.NoError(t, err)
	require.NotEmpty(t, badDiags, "a real syntax error must still be reported")
}
