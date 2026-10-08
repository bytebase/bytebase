package pg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The fixtures are real `EXPLAIN (FORMAT JSON)` outputs captured from PostgreSQL 17 after ANALYZE:
// t has 1000 rows, s has 100 rows (10 flagged), big has 100 rows per s id, and parted and the
// inheritance parent inh_parent each hold 1000 rows.
func TestGetEstimatedAffectedRowsFromExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		fixture  string
		wantRows int64
	}{
		{fixture: "update.json", wantRows: 100},
		{fixture: "delete_using_join.json", wantRows: 1000},
		{fixture: "update_from_join.json", wantRows: 1000},
		{fixture: "delete_in_subquery.json", wantRows: 1000},
		{fixture: "update_correlated_set_subquery.json", wantRows: 100},
		{fixture: "update_initplan.json", wantRows: 3333},
		{fixture: "update_partitioned.json", wantRows: 100},
		{fixture: "delete_inheritance_parent.json", wantRows: 101},
		{fixture: "insert_select.json", wantRows: 100},
		{fixture: "insert_select_on_conflict_do_nothing.json", wantRows: 1000},
		{fixture: "insert_values_on_conflict.json", wantRows: 2},
		{fixture: "merge.json", wantRows: 10000},
		{fixture: "cte_delete_select.json", wantRows: 5000},
		// The CTE deletes 5000 rows and the outer INSERT inserts them again.
		{fixture: "cte_delete_insert.json", wantRows: 10000},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			rows, err := GetEstimatedAffectedRowsFromExplainJSON(readExplainPlanFixture(t, tc.fixture))
			require.NoError(t, err)
			require.Equal(t, tc.wantRows, rows)
		})
	}
}

func TestGetEstimatedInsertedRowsFromExplainJSON(t *testing.T) {
	for _, tc := range []struct {
		fixture  string
		wantRows int64
	}{
		{fixture: "insert_select.json", wantRows: 100},
		// The outer INSERT adds the 5000 rows its CTE deletes.
		{fixture: "cte_delete_insert.json", wantRows: 5000},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			rows, err := GetEstimatedInsertedRowsFromExplainJSON(readExplainPlanFixture(t, tc.fixture))
			require.NoError(t, err)
			require.Equal(t, tc.wantRows, rows)
		})
	}

	_, err := GetEstimatedInsertedRowsFromExplainJSON(readExplainPlanFixture(t, "select.json"))
	require.EqualError(t, err, "the plan root is not a ModifyTable node")
}

func TestGetEstimatedAffectedRowsFromExplainJSONMemberSubplans(t *testing.T) {
	// PostgreSQL 13 and earlier plan a partitioned UPDATE as one ModifyTable subplan per partition.
	const plan = `[{"Plan": {"Node Type": "ModifyTable", "Operation": "Update", "Plan Rows": 0, "Plans": [
		{"Node Type": "Seq Scan", "Parent Relationship": "Member", "Relation Name": "parted_p1", "Plan Rows": 50},
		{"Node Type": "Seq Scan", "Parent Relationship": "Member", "Relation Name": "parted_p2", "Plan Rows": 50}
	]}}]`
	rows, err := GetEstimatedAffectedRowsFromExplainJSON(plan)
	require.NoError(t, err)
	require.Equal(t, int64(100), rows)
}

func TestGetEstimatedAffectedRowsFromExplainJSONError(t *testing.T) {
	for name, plan := range map[string]string{
		"select":                       readExplainPlanFixture(t, "select.json"),
		"costs off":                    readExplainPlanFixture(t, "update_costs_off.json"),
		"empty":                        "",
		"not json":                     "not json",
		"no statements":                "[]",
		"text format":                  "Update on t  (cost=5.05..13.30 rows=0 width=0)",
		"modify table without subplan": `[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 0}}]`,
		"modify table with only an init plan": `[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 0, "Plans": [
			{"Node Type": "Result", "Parent Relationship": "InitPlan", "Plan Rows": 1}
		]}}]`,
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := GetEstimatedAffectedRowsFromExplainJSON(plan)
			require.Error(t, err)
			require.Zero(t, rows)
		})
	}
}

func readExplainPlanFixture(t *testing.T, name string) string {
	t.Helper()
	plan, err := os.ReadFile(filepath.Join("test-data", "explain-plan", name))
	require.NoError(t, err)
	return string(plan)
}
