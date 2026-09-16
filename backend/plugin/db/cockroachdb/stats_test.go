package cockroachdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// The fixtures are EXPLAIN outputs captured through this driver from CockroachDB v25.2. After ANALYZE,
// t has 1000 rows with an index on c = id % 100 and none on d = id % 7, s has 100 rows (10 flagged),
// big has 100 rows per s id, big2 has 10000 rows and no secondary index, and parent has 100 rows with
// 10 cascading child rows each. t2 is empty and has no statistics.
func TestGetAffectedRowsFromPlan(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		want    int64
	}{
		{fixture: "update_index_join.txt", want: 100},
		// The filter estimates 143 of the 1,000 scanned rows.
		{fixture: "update_filter.txt", want: 143},
		// The limit caps the filter's 143 rows.
		{fixture: "delete_filter_limit.txt", want: 10},
		{fixture: "update_from_lookup_join.txt", want: 1000},
		{fixture: "update_exists_hash_semi_join.txt", want: 1000},
		// The subquery in SET is a sibling of the UPDATE, not its input.
		{fixture: "update_subquery_in_set.txt", want: 100},
		// The anti join against t2 has no estimate, so its input's estimate is used.
		{fixture: "insert_select_on_conflict_do_nothing.txt", want: 100},
		// The foreign key check after the INSERT is not its input.
		{fixture: "insert_select_fk_check.txt", want: 49},
		{fixture: "upsert_select.txt", want: 100},
		{fixture: "upsert_values_select.txt", want: 1},
		{fixture: "insert_fast_path.txt", want: 2},
		// The DELETE returns rows to the CTE, so it has an estimate of its own.
		{fixture: "cte_delete_select.txt", want: 5000},
		// The statement deletes 100 rows and inserts them into t2.
		{fixture: "insert_select_statement_source_delete.txt", want: 200},
		{fixture: "delete_range_fast_path_off.txt", want: 100},
		// The DELETE removes 10 parent rows; the 100 child rows it cascades to are not counted.
		{fixture: "delete_fk_cascade_fast_path_off.txt", want: 10},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			rows, err := getAffectedRowsFromPlan(readPlanFixture(t, tc.fixture))
			require.NoError(t, err)
			require.Equal(t, tc.want, rows)
		})
	}
}

func TestGetAffectedRowsFromPlanError(t *testing.T) {
	for name, plan := range map[string][]string{
		"select":                          readPlanFixture(t, "select.txt"),
		"delete range":                    readPlanFixture(t, "delete_range.txt"),
		"missing stats":                   readPlanFixture(t, "update_missing_stats.txt"),
		"empty":                           nil,
		"unparsable estimate":             {"• update", "│", "└── • scan", "      estimated row count: many"},
		"limit count without an estimate": {"• delete", "│", "└── • limit", "      count: 10"},
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := getAffectedRowsFromPlan(plan)
			require.Error(t, err)
			require.Zero(t, rows)
		})
	}
}

func TestCountAffectedRows(t *testing.T) {
	const statement = "DELETE FROM big2 WHERE id <= 100"
	const setLocal = "SET LOCAL optimizer_use_delete_range_fast_path = off"
	for _, tc := range []struct {
		name string
		// input is the statement passed to CountAffectedRows, when it is not statement.
		input          string
		plans          [][]string
		setLocalErr    error
		want           int64
		wantErr        string
		wantStatements []string
	}{
		{
			name:           "search_path_is_set_for_the_explain",
			input:          base.WithSearchPath(statement, []string{"app"}),
			plans:          [][]string{readPlanFixture(t, "delete_range_fast_path_off.txt")},
			want:           100,
			wantStatements: []string{"BEGIN", `SET LOCAL search_path TO "app"`, "EXPLAIN " + statement, "ROLLBACK"},
		},
		{
			name:           "plan_with_an_estimate",
			plans:          [][]string{readPlanFixture(t, "delete_range_fast_path_off.txt")},
			want:           100,
			wantStatements: []string{"EXPLAIN " + statement},
		},
		{
			name:           "delete_range_is_planned_again_without_the_fast_path",
			plans:          [][]string{readPlanFixture(t, "delete_range.txt"), readPlanFixture(t, "delete_range_fast_path_off.txt")},
			want:           100,
			wantStatements: []string{"EXPLAIN " + statement, "BEGIN", setLocal, "EXPLAIN " + statement, "ROLLBACK"},
		},
		{
			name:           "version_without_the_setting_keeps_the_delete_range_error",
			plans:          [][]string{readPlanFixture(t, "delete_range.txt")},
			setLocalErr:    errors.New(`unrecognized configuration parameter "optimizer_use_delete_range_fast_path"`),
			wantErr:        "the delete range node in the plan has no row count estimate",
			wantStatements: []string{"EXPLAIN " + statement, "BEGIN", setLocal, "ROLLBACK"},
		},
		{
			name:           "plan_without_delete_range_is_not_planned_again",
			plans:          [][]string{readPlanFixture(t, "select.txt")},
			wantErr:        "the plan has no mutation node",
			wantStatements: []string{"EXPLAIN " + statement},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connector := &fakeExplainConnector{plans: tc.plans, setLocalErr: tc.setLocalErr}
			db := sql.OpenDB(connector)
			defer db.Close()

			input := tc.input
			if input == "" {
				input = statement
			}
			got, err := (&Driver{db: db}).CountAffectedRows(context.Background(), input)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
			require.Equal(t, tc.wantStatements, connector.statements)
		})
	}
}

func readPlanFixture(t *testing.T, name string) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("test-data", "explain-plan", name))
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
}

// fakeExplainConnector answers each EXPLAIN with the next plan and records every statement,
// including transaction control.
type fakeExplainConnector struct {
	plans       [][]string
	setLocalErr error
	statements  []string
}

func (c *fakeExplainConnector) Connect(context.Context) (driver.Conn, error) {
	return &fakeExplainConn{connector: c}, nil
}

func (*fakeExplainConnector) Driver() driver.Driver {
	return nil
}

type fakeExplainConn struct {
	connector *fakeExplainConnector
}

func (*fakeExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*fakeExplainConn) Close() error {
	return nil
}

func (c *fakeExplainConn) Begin() (driver.Tx, error) {
	c.connector.statements = append(c.connector.statements, "BEGIN")
	return &fakeExplainTx{connector: c.connector}, nil
}

func (c *fakeExplainConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.connector.statements = append(c.connector.statements, query)
	return driver.ResultNoRows, c.connector.setLocalErr
}

func (c *fakeExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.connector.statements = append(c.connector.statements, query)
	if len(c.connector.plans) == 0 {
		return nil, errors.New("unexpected EXPLAIN")
	}
	plan := c.connector.plans[0]
	c.connector.plans = c.connector.plans[1:]
	return &fakeExplainRows{lines: plan}, nil
}

type fakeExplainTx struct {
	connector *fakeExplainConnector
}

func (tx *fakeExplainTx) Commit() error {
	tx.connector.statements = append(tx.connector.statements, "COMMIT")
	return nil
}

func (tx *fakeExplainTx) Rollback() error {
	tx.connector.statements = append(tx.connector.statements, "ROLLBACK")
	return nil
}

type fakeExplainRows struct {
	lines []string
}

func (*fakeExplainRows) Columns() []string {
	return []string{"info"}
}

func (*fakeExplainRows) Close() error {
	return nil
}

func (r *fakeExplainRows) Next(dest []driver.Value) error {
	if len(r.lines) == 0 {
		return io.EOF
	}
	dest[0] = r.lines[0]
	r.lines = r.lines[1:]
	return nil
}
