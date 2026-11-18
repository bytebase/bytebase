package tidb

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

func TestCountAffectedRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)

	driver := openTestDriver(ctx, t, container)
	t.Cleanup(func() {
		require.NoError(t, driver.Close(ctx))
	})

	// By default ANALYZE collects only columns that earlier queries filtered on,
	// which would leave c without statistics.
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE %[1]s.t (id INT PRIMARY KEY, c INT);
		CREATE TABLE %[1]s.archive (id INT PRIMARY KEY, c INT);
		INSERT INTO %[1]s.t
			WITH RECURSIVE seq (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < 1000)
			SELECT n, n %% 10 FROM seq;
		ANALYZE TABLE %[1]s.t ALL COLUMNS;
	`, database))
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		statement string
	}{
		{name: "update", statement: fmt.Sprintf("UPDATE %s.t SET c = 0 WHERE c = 1", database)},
		{name: "delete", statement: fmt.Sprintf("DELETE FROM %s.t WHERE c = 1", database)},
		{name: "insert_select", statement: fmt.Sprintf("INSERT INTO %[1]s.archive SELECT * FROM %[1]s.t WHERE c = 1", database)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := driver.CountAffectedRows(ctx, tc.statement)
			require.NoError(t, err)
			// 100 of the 1000 rows match c = 1, while the table scan beneath the
			// filter estimates all 1000.
			require.Positive(t, got)
			require.Less(t, got, int64(1000))
		})
	}
}

// TestCountAffectedRowsRefusesANonDMLStatement pins the known difference in the
// message: the plan estimate is taken for exactly one INSERT, UPDATE or DELETE,
// so a reader is refused here rather than by the server.
func TestCountAffectedRowsRefusesANonDMLStatement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	driver := openTestDriver(ctx, t, container)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	_, err := driver.CountAffectedRows(ctx, fmt.Sprintf("SELECT * FROM %s.t", database))
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected an INSERT, UPDATE or DELETE statement")
}
