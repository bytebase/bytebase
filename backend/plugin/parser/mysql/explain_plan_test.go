package mysql

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/stretchr/testify/require"
)

// The fixtures are real `EXPLAIN FORMAT=JSON` outputs. Plans over target_table (400k rows) and
// related_table (100k rows) come from MySQL 8.0 and MariaDB 11. The others come from MySQL 8.0.33
// (unprefixed), MySQL 5.7, and MariaDB 11.8 against t (1000 rows), s (100 rows, 10 flagged), big
// (10000 rows, 100 per s row), small (3 rows), and the empty copies t2 and big2.
func TestAffectedRowsQuery(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		want      string
	}{
		{
			name:      "update with filter, order, and limit",
			statement: "UPDATE t SET v = CONCAT(v, 'y'), c = c + 1 WHERE c < 10 ORDER BY id LIMIT 3;",
			want:      "SELECT 1 FROM t WHERE c < 10 ORDER BY id LIMIT 3",
		},
		{
			name:      "update of an aliased table with an index hint",
			statement: "UPDATE t AS tt FORCE INDEX (idx_c) SET tt.v = 'y' WHERE tt.c < 10",
			want:      "SELECT 1 FROM t AS tt FORCE INDEX (idx_c) WHERE tt.c < 10",
		},
		{
			name:      "update of every row",
			statement: "UPDATE t SET d = CASE WHEN c > 1 THEN 1 ELSE 2 END;",
			want:      "SELECT 1 FROM t",
		},
		{
			name:      "update with a comment before WHERE",
			statement: "UPDATE t SET v = (SELECT MAX(v) FROM s) /* latest */ WHERE id = 1",
			want:      "SELECT 1 FROM t WHERE id = 1",
		},
		{
			name:      "delete with a comment before WHERE",
			statement: "DELETE FROM t -- ticket 42\nWHERE id = 1;",
			want:      "SELECT 1 FROM t -- ticket 42\nWHERE id = 1",
		},
		{
			name:      "update with a block comment before WHERE",
			statement: "UPDATE t SET c = 1 /* ticket 42 */ WHERE id = 1;",
			want:      "SELECT 1 FROM t WHERE id = 1",
		},
		{
			name:      "delete with an executable comment",
			statement: "DELETE FROM t /*!80000 WHERE id = 1 */;",
			want:      "DELETE FROM t /*!80000 WHERE id = 1 */;",
		},
		{
			name:      "update with an optimizer hint",
			statement: "UPDATE /*+ SET_VAR(optimizer_switch='condition_fanout_filter=off') */ t SET c = 1 WHERE id > 5;",
			want:      "UPDATE /*+ SET_VAR(optimizer_switch='condition_fanout_filter=off') */ t SET c = 1 WHERE id > 5;",
		},
		{
			name:      "delete with an optimizer hint",
			statement: "DELETE /*+ NO_RANGE_OPTIMIZATION(t) */ FROM t WHERE id > 5;",
			want:      "DELETE /*+ NO_RANGE_OPTIMIZATION(t) */ FROM t WHERE id > 5;",
		},
		{
			// omni positions the text after an executable comment as if its markers were removed.
			name:      "delete with an executable comment in WHERE",
			statement: "DELETE FROM big WHERE /*!50700 v = 0 AND */ s_id < 10000000000;",
			want:      "DELETE FROM big WHERE /*!50700 v = 0 AND */ s_id < 10000000000;",
		},
		{
			name:      "update with a comment before the table",
			statement: "UPDATE /* ticket 42 */ t SET c = 1 WHERE id > 5;",
			want:      "SELECT 1 FROM t WHERE id > 5",
		},
		{
			name:      "update with a CTE",
			statement: "WITH x AS (SELECT id FROM s WHERE flag = 1) UPDATE t SET v = 'y' WHERE grp IN (SELECT id FROM x);",
			want:      "WITH x AS (SELECT id FROM s WHERE flag = 1) SELECT 1 FROM t WHERE grp IN (SELECT id FROM x)",
		},
		{
			name:      "delete from a partition",
			statement: "DELETE LOW_PRIORITY QUICK IGNORE FROM db1.t PARTITION (p0) WHERE c = 1 ORDER BY id DESC LIMIT 5;",
			want:      "SELECT 1 FROM db1.t PARTITION (p0) WHERE c = 1 ORDER BY id DESC LIMIT 5",
		},
		{
			name:      "delete from an aliased table",
			statement: "DELETE FROM t AS a WHERE a.c = 1;",
			want:      "SELECT 1 FROM t AS a WHERE a.c = 1",
		},
		{
			name:      "delete of every row",
			statement: "DELETE FROM big;",
			want:      "SELECT 1 FROM big",
		},
		{
			name:      "delete from a partition of an aliased table",
			statement: "DELETE FROM t AS a PARTITION (p0) WHERE a.c = 1;",
			want:      "DELETE FROM t AS a PARTITION (p0) WHERE a.c = 1;",
		},
		{
			name:      "multi-table update",
			statement: "UPDATE big JOIN s ON big.s_id = s.id SET big.v = 1 WHERE s.flag = 1;",
			want:      "UPDATE big JOIN s ON big.s_id = s.id SET big.v = 1 WHERE s.flag = 1;",
		},
		{
			name:      "update of a table list",
			statement: "UPDATE t, s SET t.v = 'a' WHERE t.id = s.id;",
			want:      "UPDATE t, s SET t.v = 'a' WHERE t.id = s.id;",
		},
		{
			name:      "multi-table delete",
			statement: "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			want:      "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
		},
		{
			name:      "delete with USING",
			statement: "DELETE FROM big USING big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			want:      "DELETE FROM big USING big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
		},
		{
			name:      "insert",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			want:      "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, AffectedRowsQuery(parseSingleStatement(t, tc.statement), tc.statement))
		})
	}
}

func TestEstimateAffectedRowsFromExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		fixture   string
		statement string
		want      int64
	}{
		{
			// The UPDATE target is the last node of a duplicate-weedout semijoin (BYT-9858).
			fixture:   "semijoin_update.json",
			statement: "UPDATE target_table o SET o.target_flag = 1 WHERE o.target_flag IS NULL AND EXISTS (SELECT 1 FROM related_table t WHERE t.join_key = o.join_key AND t.filter_column_1 = 'VALUE_A');",
			want:      1144,
		},
		{
			// Single-table plans have no rows_produced_per_join; the scan estimate is scaled by filtered.
			fixture:   "single_delete.json",
			statement: "DELETE FROM target_table WHERE join_key < 50;",
			want:      200,
		},
		{
			fixture:   "order_limit_update.json",
			statement: "UPDATE target_table SET target_flag = 1 WHERE join_key < 50 ORDER BY id LIMIT 500;",
			want:      200,
		},
		{
			fixture:   "not_exists_update.json",
			statement: "UPDATE target_table o SET o.target_flag = 1 WHERE o.target_flag IS NULL AND NOT EXISTS (SELECT 1 FROM related_table t WHERE t.join_key = o.join_key AND t.filter_column_1 = 'VALUE_A');",
			want:      39964,
		},
		{
			// The subquery plan in attached_subqueries does not contribute.
			fixture:   "in_materialized_update.json",
			statement: "UPDATE target_table o SET o.target_flag = 1 WHERE o.join_key IN (SELECT t.join_key FROM related_table t WHERE t.filter_column_1 = 'VALUE_A' GROUP BY t.join_key HAVING COUNT(*) > 0);",
			want:      399648,
		},
		{
			// Both targets are flagged: min(10023, 38163) + min(38163, 38163).
			fixture:   "multi_target_delete.json",
			statement: "DELETE o, t FROM target_table o JOIN related_table t ON t.join_key = o.join_key WHERE t.filter_column_1 = 'VALUE_A';",
			want:      48186,
		},
		{
			fixture:   "delete_limit.json",
			statement: "DELETE FROM target_table LIMIT 10;",
			want:      10,
		},
		{
			fixture:   "mariadb_update.json",
			statement: "UPDATE target_table o SET o.target_flag = 1 WHERE o.target_flag IS NULL AND EXISTS (SELECT 1 FROM related_table t WHERE t.join_key = o.join_key AND t.filter_column_1 = 'VALUE_A');",
			want:      10000,
		},
		{
			// The plan scans the flagged target first; the later join filter bounds it: min(11330, 1133).
			fixture:   "update_target_first_straight_join.json",
			statement: "UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;",
			want:      1133,
		},
		{
			// The derived table's fanout does not raise the target past its own estimate: min(1000, 10000).
			fixture:   "update_derived_join.json",
			statement: "UPDATE t JOIN (SELECT grp, COUNT(*) cnt FROM t GROUP BY grp) g ON g.grp = t.grp SET t.v = 'g' WHERE g.cnt > 50;",
			want:      1000,
		},
		{
			// The materialization scan prints no rows, and the target's 113 is per materialized row: 113 * 10.
			fixture:   "delete_cte_semijoin_materialization_scan.json",
			statement: "WITH x AS (SELECT id FROM s WHERE flag = 1) DELETE FROM big WHERE s_id IN (SELECT id FROM x);",
			want:      1130,
		},
		{
			// The plans of the SELECT that AffectedRowsQuery explains for a single-table change.
			fixture:   "select_for_update_filter.json",
			statement: "UPDATE t SET v = CONCAT(v, 'y') WHERE d = 3;",
			want:      100,
		},
		{
			fixture:   "mariadb_select_for_delete_range_filter.json",
			statement: "DELETE FROM t WHERE c < 10 AND d = 3;",
			want:      14,
		},
		{
			fixture:   "mariadb_select_for_delete_all_rows.json",
			statement: "DELETE FROM big;",
			want:      10000,
		},
		{
			// A target the plan does not name, such as a view, counts the final estimate.
			fixture:   "select_only.json",
			statement: "UPDATE other_table SET target_flag = 1;",
			want:      200,
		},
		{
			// UNION ALL of 3 joined rows and the first EXCEPT branch's 500.
			fixture:   "insert_select_union_except.json",
			statement: "INSERT INTO t2 SELECT t.* FROM small JOIN t USING (id) UNION ALL (SELECT * FROM t WHERE c < 50 EXCEPT SELECT * FROM t WHERE d = 1);",
			want:      503,
		},
		{
			// UNION ALL of 3 joined rows and the smaller INTERSECT branch's 100.
			fixture:   "insert_select_union_intersect.json",
			statement: "INSERT INTO t2 SELECT t.* FROM small JOIN t USING (id) UNION ALL (SELECT * FROM t WHERE c < 50 INTERSECT SELECT * FROM t WHERE d = 1);",
			want:      103,
		},
		{
			fixture:   "insert_select_parenthesized_limit.json",
			statement: "INSERT INTO t2 (SELECT * FROM t ORDER BY c LIMIT 500) ORDER BY id LIMIT 400;",
			want:      400,
		},
		{
			fixture:   "update_impossible_where.json",
			statement: "UPDATE t SET v = 'y' WHERE 1 = 0;",
			want:      0,
		},
		{
			// MySQL 5.7 prints the row estimate beside the "Deleting all rows" message.
			fixture:   "mysql57_delete_all_rows.json",
			statement: "DELETE FROM big;",
			want:      10195,
		},
		{
			fixture:   "insert_select_filter.json",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			want:      100,
		},
		{
			fixture:   "insert_select_join.json",
			statement: "INSERT INTO big2 SELECT big.* FROM s JOIN big ON big.s_id = s.id WHERE s.flag = 1;",
			want:      1133,
		},
		{
			// Only the first branch sits under insert_from: 100 + 899.
			fixture:   "insert_select_union_all.json",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE c < 10 UNION ALL SELECT * FROM t WHERE c >= 10;",
			want:      999,
		},
		{
			fixture:   "replace_select_join.json",
			statement: "REPLACE INTO big2 SELECT big.* FROM s JOIN big ON big.s_id = s.id WHERE s.flag = 1;",
			want:      1133,
		},
		{
			fixture:   "insert_table.json",
			statement: "INSERT INTO t2 TABLE t;",
			want:      1000,
		},
		{
			fixture:   "insert_select_join.json",
			statement: "INSERT INTO big2 SELECT big.* FROM s JOIN big ON big.s_id = s.id WHERE s.flag = 1 LIMIT 30;",
			want:      30,
		},
		{
			// The lookup into the materialized subquery has no per-join estimate.
			fixture:   "insert_select_semijoin_materialization_lookup.json",
			statement: "INSERT INTO t2 WITH x AS (SELECT id FROM s WHERE flag = 1) SELECT * FROM t WHERE grp IN (SELECT id FROM x);",
			want:      1000,
		},
		{
			fixture:   "insert_select_buffer_result.json",
			statement: "INSERT INTO t2 SELECT t.* FROM t LEFT JOIN t2 ON t2.id = t.id WHERE t2.id IS NULL;",
			want:      1000,
		},
		{
			fixture:   "insert_select_impossible_where.json",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE 1 = 0;",
			want:      0,
		},
		{
			// MariaDB marks no multi-table target, so the target is found by name.
			fixture:   "mariadb_delete_join.json",
			statement: "DELETE big FROM big JOIN s ON big.s_id = s.id WHERE s.flag = 1;",
			want:      1000,
		},
		{
			fixture:   "mariadb_delete_semijoin.json",
			statement: "DELETE FROM big WHERE s_id IN (SELECT id FROM s WHERE flag = 1);",
			want:      1000,
		},
		{
			// The lookup into the materialized subquery prints no loops.
			fixture:   "mariadb_delete_semijoin_materialization.json",
			statement: "DELETE FROM s WHERE id IN (SELECT s_id FROM big WHERE v = 0);",
			want:      100,
		},
		{
			// min(10000, 1000) for the target big.
			fixture:   "mariadb_update_straight_join.json",
			statement: "UPDATE big STRAIGHT_JOIN s ON big.s_id = s.id SET big.v = big.v + 1 WHERE s.flag = 1;",
			want:      1000,
		},
		{
			// big: min(1000, 1000); s: min(10, 1000).
			fixture:   "mariadb_update_two_targets.json",
			statement: "UPDATE big JOIN s ON big.s_id = s.id SET big.v = 1, s.flag = 1 WHERE s.flag = 1;",
			want:      1010,
		},
		{
			fixture:   "mariadb_update_join_unqualified_column.json",
			statement: "UPDATE big AS b JOIN s AS x ON b.s_id = x.id SET v = 1 WHERE x.flag = 1;",
			want:      1000,
		},
		{
			// Either joined table may own each unqualified column: 2 * 1000.
			fixture:   "mariadb_update_join_unqualified_column.json",
			statement: "UPDATE big AS b JOIN s AS x ON b.s_id = x.id SET v = 1, flag = 0 WHERE x.flag = 1;",
			want:      2000,
		},
		{
			fixture:   "mariadb_delete_impossible_where.json",
			statement: "DELETE FROM t WHERE 1 = 0;",
			want:      0,
		},
		{
			// 1000 rows filtered to 14.3%.
			fixture:   "mariadb_insert_select_filter.json",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE d = 3;",
			want:      143,
		},
		{
			fixture:   "mariadb_insert_select_join.json",
			statement: "INSERT INTO big2 SELECT big.* FROM s JOIN big ON big.s_id = s.id WHERE s.flag = 1;",
			want:      1000,
		},
		{
			fixture:   "mariadb_insert_select_union_all.json",
			statement: "INSERT INTO t2 SELECT * FROM t WHERE c < 10 UNION ALL SELECT * FROM t WHERE c >= 10;",
			want:      1000,
		},
		{
			fixture:   "mariadb_insert_select_block_nl_join.json",
			statement: "INSERT INTO t2 SELECT t.* FROM t JOIN s ON t.d = s.flag WHERE s.id < 50;",
			want:      49000,
		},
		{
			fixture:   "mariadb_insert_select_order_limit.json",
			statement: "INSERT INTO t2 SELECT * FROM t ORDER BY d LIMIT 50;",
			want:      50,
		},
		{
			fixture:   "mariadb_insert_select_window.json",
			statement: "INSERT INTO t2 (id, c, d, grp, v) SELECT id, ROW_NUMBER() OVER (PARTITION BY grp ORDER BY id), d, grp, v FROM t WHERE c < 10;",
			want:      100,
		},
		{
			fixture:   "mariadb_insert_select_no_tables.json",
			statement: "INSERT INTO t2 SELECT 1, 1, 1, 1, 'a';",
			want:      1,
		},
		{
			// MySQL prints only the INSERT target for a source that reads no table.
			fixture:   "insert_select_from_dual_not_exists.json",
			statement: "INSERT INTO t2 SELECT 1, 1, 1, 1, 'a' FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t2 WHERE id = 1);",
			want:      1,
		},
		{
			fixture:   "insert_select_union_without_tables.json",
			statement: "INSERT INTO t2 SELECT 1, 1, 1, 1, 'a' UNION ALL SELECT 2, 2, 2, 2, 'b';",
			want:      2,
		},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			got, err := EstimateAffectedRowsFromExplainJSON(parseSingleStatement(t, tc.statement), readExplainPlanFixture(t, tc.fixture))
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTableFreeSelectRows(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      int
		wantOK    bool
	}{
		{statement: "INSERT INTO t SELECT 1 UNION ALL SELECT 2 UNION SELECT 3", want: 3, wantOK: true},
		{statement: "INSERT INTO t SELECT 1 INTERSECT SELECT 1 FROM DUAL", want: 1, wantOK: true},
		{statement: "INSERT INTO t SELECT 1 EXCEPT SELECT 2", want: 1, wantOK: true},
		{statement: "INSERT INTO t SELECT 1 UNION ALL SELECT id FROM s"},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			insert, ok := parseSingleStatement(t, tc.statement).(*ast.InsertStmt)
			require.True(t, ok)
			got, ok := tableFreeSelectRows(insert.Select)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDMLTargetCount(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      int
	}{
		{statement: "UPDATE t SET a = 1, b = 2 WHERE id = 1", want: 1},
		{statement: "UPDATE a JOIN b ON a.id = b.id SET a.x = 1, b.y = 1", want: 2},
		{statement: "UPDATE a JOIN b ON a.id = b.id SET x = 1, y = 2, z = 3", want: 2},
		{statement: "DELETE a, b FROM a JOIN b ON a.id = b.id", want: 2},
		{statement: "DELETE FROM t WHERE id = 1", want: 1},
		{statement: "INSERT INTO t SELECT * FROM s", want: 1},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			require.Equal(t, tc.want, DMLTargetCount(parseSingleStatement(t, tc.statement), 2))
		})
	}
}

func TestCapAffectedRowsByLimit(t *testing.T) {
	for _, tc := range []struct {
		statement string
		rows      float64
		want      int64
	}{
		{statement: "INSERT INTO t2 SELECT t.id FROM t, big, big AS big3", rows: 1e30, want: math.MaxInt64},
		{statement: "INSERT INTO t2 SELECT t.id FROM t, big, big AS big3 LIMIT 10", rows: 1e30, want: 10},
		{statement: "DELETE FROM t WHERE id > 0 LIMIT 5", rows: 2.5, want: 3},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			require.Equal(t, tc.want, CapAffectedRowsByLimit(parseSingleStatement(t, tc.statement), tc.rows))
		})
	}
}

func TestEstimateAffectedRowsFromExplainJSONError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		plan      string
		statement string
		wantErr   string
	}{
		{
			// MySQL 8.0 omits insert_from when the SELECT computes window functions.
			name:      "insert without a source plan",
			plan:      readExplainPlanFixture(t, "insert_select_window_no_source.json"),
			statement: "INSERT INTO t2 (id, c, d, grp, v) SELECT id, ROW_NUMBER() OVER (PARTITION BY grp ORDER BY id), d, grp, v FROM t WHERE c < 10;",
			wantErr:   `table "t2" in the plan has no row estimate`,
		},
		{
			name:      "MariaDB delete of all rows",
			plan:      readExplainPlanFixture(t, "mariadb_delete_all_rows.json"),
			statement: "DELETE FROM big;",
			wantErr:   "the plan has no row estimate: Deleting all rows",
		},
		{
			name:      "format version 2",
			plan:      `{"query": "/* select#1 */ delete from t", "operation": "Delete from t", "access_type": "table", "estimated_rows": 1000}`,
			statement: "DELETE FROM t;",
			wantErr:   "no query_block",
		},
		{
			name:      "unrecognized plan node",
			plan:      `{"query_block": {"select_id": 1, "future_operation": {"table": {"delete": true, "table_name": "t", "rows_examined_per_scan": 10}}}}`,
			statement: "DELETE FROM t;",
			wantErr:   "unrecognized EXPLAIN FORMAT=JSON plan node",
		},
		{
			name:      "not JSON",
			plan:      "-> Delete from t",
			statement: "DELETE FROM t;",
			wantErr:   "failed to parse EXPLAIN FORMAT=JSON output",
		},
		{
			name:      "unsupported statement",
			plan:      readExplainPlanFixture(t, "select_only.json"),
			statement: "SELECT * FROM target_table;",
			wantErr:   "unsupported statement type",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EstimateAffectedRowsFromExplainJSON(parseSingleStatement(t, tc.statement), tc.plan)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestEstimateAffectedRowsFromOceanBaseExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		plan string
		want int64
	}{
		{
			name: "root operator",
			plan: `{"ID":0,"OPERATOR":"UPDATE","NAME":"","EST.ROWS":1000,"EST.TIME(us)":7680,"output":"","CHILD_1":{"ID":1,"OPERATOR":"TABLE RANGE SCAN","NAME":"dba_test_1","EST.ROWS":1200,"EST.TIME(us)":91}}`,
			want: 1000,
		},
		{
			name: "fractional estimate",
			plan: `{"ID":0,"OPERATOR":"DELETE","NAME":"","EST.ROWS":2.6,"EST.TIME(us)":20}`,
			want: 3,
		},
		{
			name: "largest child operator without a root operator",
			plan: `{"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t1","EST.ROWS":40},"CHILD_2":{"ID":2,"OPERATOR":"TABLE FULL SCAN","NAME":"t2","EST.ROWS":70}}`,
			want: 70,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EstimateAffectedRowsFromOceanBaseExplainJSON(tc.plan)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct {
		name    string
		plan    string
		wantErr string
	}{
		{
			name:    "root operator without an estimate",
			plan:    `{"ID":0,"OPERATOR":"UPDATE","NAME":""}`,
			wantErr: `operator "UPDATE" has no EST.ROWS`,
		},
		{
			name:    "child operator without an estimate",
			plan:    `{"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t1"}}`,
			wantErr: `operator "TABLE FULL SCAN" has no EST.ROWS`,
		},
		{
			name:    "no operator",
			plan:    `{}`,
			wantErr: "the plan has no operator",
		},
		{
			name:    "empty output",
			plan:    "",
			wantErr: "failed to parse EXPLAIN FORMAT=JSON output",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EstimateAffectedRowsFromOceanBaseExplainJSON(tc.plan)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func readExplainPlanFixture(t *testing.T, fixture string) string {
	t.Helper()
	plan, err := os.ReadFile(filepath.Join("test-data", "explain-plan", fixture))
	require.NoError(t, err)
	return string(plan)
}

func parseSingleStatement(t *testing.T, statement string) ast.Node {
	t.Helper()
	list, err := ParseMySQL(statement)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	return list.Items[0]
}
