package tidb

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
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

// The plans are EXPLAIN output captured from TiDB v8.5.0.
func TestGetAffectedRowsFromPlan(t *testing.T) {
	for _, tc := range []struct {
		name    string
		plan    []planRow
		want    int64
		wantErr bool
	}{
		{
			name: "join_estimates_joined_rows_not_its_inputs",
			plan: []planRow{
				{"Update_8", "N/A"},
				{"└─HashJoin_27", "800.00"},
				{"  ├─TableReader_44(Build)", "0.40"},
				{"  │ └─Selection_43", "0.40"},
				{"  │   └─TableFullScan_42", "400.00"},
				{"  └─TableReader_38(Probe)", "1000.00"},
				{"    └─Selection_37", "1000.00"},
				{"      └─TableFullScan_36", "1000.00"},
			},
			want: 800,
		},
		{
			name: "limit_is_part_of_the_plan",
			plan: []planRow{
				{"Update_6", "N/A"},
				{"└─TopN_9", "20.00"},
				{"  └─TableReader_17", "20.00"},
				{"    └─TopN_16", "20.00"},
				{"      └─Selection_15", "500.00"},
				{"        └─TableFullScan_14", "1000.00"},
			},
			want: 20,
		},
		{
			name: "foreign_key_cascade_follows_the_input",
			plan: []planRow{
				{"Delete_3", "N/A"},
				{"├─TableReader_8", "9.00"},
				{"│ └─TableRangeScan_7", "9.00"},
				{"└─Foreign_Key_Cascade_4", "0.00"},
			},
			want: 9,
		},
		{
			name: "insert_set_has_no_input",
			plan: []planRow{
				{"Insert_1", "N/A"},
			},
			want: 0,
		},
		{
			// With tidb_opt_enable_non_eval_scalar_subquery, a subquery in SET is
			// planned as a second root rather than evaluated during planning.
			name: "subquery_root_is_not_an_input",
			plan: []planRow{
				{"Insert_1", "N/A"},
				{"ScalarSubQuery_36", "N/A"},
				{"└─MaxOneRow_11", "1.00"},
				{"  └─Projection_12", "1.00"},
				{"    └─StreamAgg_14", "1.00"},
				{"      └─Limit_30", "1.00"},
				{"        └─TableReader_35", "1.00"},
				{"          └─Limit_34", "1.00"},
				{"            └─TableFullScan_24", "1.00"},
			},
			want: 0,
		},
		{
			name: "select_root_is_rejected",
			plan: []planRow{
				{"TableReader_7", "100.00"},
				{"└─Selection_6", "100.00"},
				{"  └─TableFullScan_5", "1000.00"},
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getAffectedRowsFromPlan(tc.plan)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, common.RoundRows(got))
		})
	}
}

func TestGetAffectedRows(t *testing.T) {
	plan := []planRow{
		{"Delete_7", "N/A"},
		{"└─HashJoin_20", "0.40"},
	}
	// 0.4 joined rows change a row in each of two tables.
	got, err := getAffectedRows(plan, "DELETE a, b FROM a JOIN b ON a.id = b.id")
	require.NoError(t, err)
	require.Equal(t, int64(1), got)
}

func TestCountDMLTargets(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      int
	}{
		{statement: "UPDATE t SET a = 1, b = 2 WHERE id = 1", want: 1},
		{statement: "UPDATE a JOIN b ON a.id = b.id SET a.c = 1, a.d = 2", want: 1},
		{statement: "UPDATE a JOIN b ON a.id = b.id SET a.c = 1, b.c = 1", want: 2},
		{statement: "UPDATE a AS x, b AS y SET x.c = 1, y.c = 1 WHERE x.id = y.id", want: 2},
		// Either joined table may own each unqualified column, up to the two joined tables.
		{statement: "UPDATE a JOIN b ON a.id = b.id SET c = 1, d = 2, e = 3", want: 2},
		{statement: "DELETE a, b FROM a JOIN b ON a.id = b.id", want: 2},
		{statement: "DELETE a FROM a JOIN b ON a.id = b.id", want: 1},
		{statement: "DELETE FROM t WHERE id = 1", want: 1},
		{statement: "INSERT INTO t SELECT * FROM s", want: 1},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			require.Equal(t, tc.want, countDMLTargets(tc.statement))
		})
	}
}
