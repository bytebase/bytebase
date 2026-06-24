package mariadb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// mariaDBOnlySQL are statements valid in MariaDB but rejected by the omni/mysql
// parser that backed Engine_MARIADB before this carve-out. Each omni add-surface
// closed a Diagnose false-flag; this list grows with the bump it ships in.
var mariaDBOnlySQL = []string{
	// carve-out baseline + UUID/INET4/INET6 (omni #318)
	"CREATE SEQUENCE s",
	"INSERT INTO t (id) VALUES (1) RETURNING id",
	"REPLACE INTO t (id) VALUES (1) RETURNING id",
	"DELETE FROM t RETURNING id",
	"SELECT NEXT VALUE FOR s",
	"CREATE TABLE t (a UUID)",
	"CREATE TABLE t (a INET4, b INET6)",
	// system-versioned tables (omni #319)
	"CREATE TABLE t (id INT) WITH SYSTEM VERSIONING",
	"SELECT * FROM t FOR SYSTEM_TIME AS OF NOW()",
	// application-time periods (omni #324-#328)
	"CREATE TABLE t (s DATE, e DATE, PERIOD FOR app_time(s, e))",
	"UPDATE t FOR PORTION OF app_time FROM '2020-01-01' TO '2021-01-01' SET id = 1",
	// parenthesized row constructors (omni #330)
	"SELECT * FROM t WHERE (a, b) IN ((1, 2), (3, 4))",
	"SELECT * FROM t WHERE (a, b) > (1, 2)",
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

	// Inverse fidelity case: FOR UPDATE ... OF is valid MySQL but MariaDB removed
	// it in 11.4 (omni #318). The bump makes MariaDB Diagnose correctly flag it,
	// while omni/mysql still accepts it.
	const forUpdateOf = "SELECT a FROM t FOR UPDATE OF t"
	mysqlFU, err := mysql.Diagnose(ctx, base.DiagnoseContext{}, forUpdateOf)
	require.NoError(t, err)
	require.Empty(t, mysqlFU, "omni/mysql accepts FOR UPDATE OF (valid MySQL)")
	mariadbFU, err := Diagnose(ctx, base.DiagnoseContext{}, forUpdateOf)
	require.NoError(t, err)
	require.NotEmpty(t, mariadbFU, "omni/mariadb must flag FOR UPDATE OF (removed in MariaDB 11.4)")

	// Diagnose must still report a genuine syntax error — proving it actually
	// validates, not just "returns empty".
	badDiags, err := Diagnose(ctx, base.DiagnoseContext{}, "SELECT FROM WHERE )(")
	require.NoError(t, err)
	require.NotEmpty(t, badDiags, "a real syntax error must still be reported")
}
