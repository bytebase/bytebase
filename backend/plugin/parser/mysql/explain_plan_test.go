package mysql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fixtures are real `EXPLAIN FORMAT=JSON` outputs captured from MySQL 8.0 and
// MariaDB 11 against a 400k-row target_table joined with a 100k-row related_table.
func TestGetEstimatedAffectedRowsFromExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		fixture   string
		wantRows  int64
		wantFound bool
	}{
		{
			// UPDATE ... WHERE ... EXISTS rewritten to a semijoin; the target table is
			// the last nested_loop node. The first tabular EXPLAIN row would report the
			// 100232-row driving-table scan (BYT-9858).
			fixture:   "semijoin_update.json",
			wantRows:  1144,
			wantFound: true,
		},
		{
			// Single-table DELETE with a range condition: no rows_produced_per_join,
			// estimate is rows_examined_per_scan scaled by filtered.
			fixture:   "single_delete.json",
			wantRows:  200,
			wantFound: true,
		},
		{
			// UPDATE ... ORDER BY ... LIMIT wraps the target node in ordering_operation.
			fixture:   "order_limit_update.json",
			wantRows:  200,
			wantFound: true,
		},
		{
			// NOT EXISTS antijoin: the target table is the first nested_loop node.
			fixture:   "not_exists_update.json",
			wantRows:  39964,
			wantFound: true,
		},
		{
			// IN (SELECT ...) materialization: the subquery plan lives in
			// attached_subqueries and must not contribute to the estimate.
			fixture:   "in_materialized_update.json",
			wantRows:  399648,
			wantFound: true,
		},
		{
			// DELETE o, t FROM ... flags both target nodes; estimates are summed.
			fixture:   "multi_target_delete.json",
			wantRows:  48186,
			wantFound: true,
		},
		{
			fixture:   "delete_limit.json",
			wantRows:  10,
			wantFound: true,
		},
		{
			// MariaDB marks the target with "update": 1 and names the estimate "rows".
			fixture:   "mariadb_update.json",
			wantRows:  10000,
			wantFound: true,
		},
		{
			// SELECT plans have no update/delete node; callers must fall back.
			fixture:   "select_only.json",
			wantRows:  0,
			wantFound: false,
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			plan, err := os.ReadFile(filepath.Join("test-data", "explain-plan", tc.fixture))
			require.NoError(t, err)
			rows, found := GetEstimatedAffectedRowsFromExplainJSON(string(plan))
			require.Equal(t, tc.wantFound, found)
			require.Equal(t, tc.wantRows, rows)
		})
	}
}

func TestGetEstimatedAffectedRowsFromExplainJSONInvalidInput(t *testing.T) {
	for _, plan := range []string{"", "not json", "[]", `{"query_block":{}}`} {
		rows, found := GetEstimatedAffectedRowsFromExplainJSON(plan)
		require.False(t, found, plan)
		require.Zero(t, rows, plan)
	}
}
