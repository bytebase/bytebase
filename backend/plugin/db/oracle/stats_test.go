package oracle

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountAffectedRows(t *testing.T) {
	const hint = "/*+ OPT_PARAM('optimizer_dynamic_sampling' 11) */"
	for _, tc := range []struct {
		name      string
		statement string
		// plan holds the ID, PARENT_ID, OPERATION, and CARDINALITY of each PLAN_TABLE row.
		plan      [][]driver.Value
		explained string
		want      int64
		wantErr   bool
	}{
		{
			name:      "large_cardinality",
			statement: "UPDATE huge SET c = 1 WHERE c < 10",
			// go-ora returns NUMBER columns as strings.
			plan: [][]driver.Value{
				{"0", nil, "UPDATE STATEMENT", "5000000"},
				{"1", "0", "UPDATE", nil},
				{"2", "1", "TABLE ACCESS", "5000000"},
			},
			explained: "UPDATE " + hint + " huge SET c = 1 WHERE c < 10",
			want:      5000000,
		},
		{
			name:      "trailing_semicolon",
			statement: "DELETE FROM t WHERE c < 10;\n",
			plan:      [][]driver.Value{{"0", nil, "DELETE STATEMENT", "101"}},
			explained: "DELETE " + hint + " FROM t WHERE c < 10",
			want:      101,
		},
		{
			// Captured from Oracle 23 with dynamic sampling, where the statement row estimates 1 row
			// for a MERGE that inserts 100.
			name:      "merge_reads_the_rows_below_the_merge_operation",
			statement: "MERGE INTO t2 USING (SELECT * FROM t WHERE c < 10) src ON (t2.id = src.id) WHEN NOT MATCHED THEN INSERT VALUES (src.id, src.c, src.d, src.grp, src.v)",
			plan: [][]driver.Value{
				{"0", nil, "MERGE STATEMENT", "1"},
				{"1", "0", "MERGE", nil},
				{"2", "1", "VIEW", nil},
				{"3", "2", "NESTED LOOPS", "100"},
				{"4", "3", "TABLE ACCESS", "100"},
				{"5", "3", "TABLE ACCESS", "1"},
				{"6", "5", "INDEX", "1"},
			},
			explained: "MERGE " + hint + " INTO t2 USING (SELECT * FROM t WHERE c < 10) src ON (t2.id = src.id) WHEN NOT MATCHED THEN INSERT VALUES (src.id, src.c, src.d, src.grp, src.v)",
			want:      100,
		},
		{
			// Captured from Oracle 23, where the statement row estimates the 1000 rows an INSERT ALL reads
			// and writes to three tables.
			name:      "insert_all_writes_each_row_to_every_into",
			statement: "INSERT ALL INTO t1 VALUES (id) INTO t2 VALUES (id) INTO t3 VALUES (id) SELECT id FROM src",
			plan: [][]driver.Value{
				{"0", nil, "INSERT STATEMENT", "1000"},
				{"1", "0", "MULTI-TABLE INSERT", nil},
				{"2", "1", "TABLE ACCESS", "1000"},
				{"3", "1", "INTO", nil},
				{"4", "1", "INTO", nil},
				{"5", "1", "INTO", nil},
			},
			explained: "INSERT " + hint + " ALL INTO t1 VALUES (id) INTO t2 VALUES (id) INTO t3 VALUES (id) SELECT id FROM src",
			want:      3000,
		},
		{
			name:      "insert_first_writes_each_row_once",
			statement: "INSERT /* move */ FIRST WHEN id < 10 THEN INTO t1 VALUES (id) ELSE INTO t2 VALUES (id) SELECT id FROM src",
			plan: [][]driver.Value{
				{"0", nil, "INSERT STATEMENT", "1000"},
				{"1", "0", "MULTI-TABLE INSERT", nil},
				{"2", "1", "TABLE ACCESS", "1000"},
				{"3", "1", "INTO", nil},
				{"4", "1", "INTO", nil},
			},
			explained: "INSERT " + hint + " /* move */ FIRST WHEN id < 10 THEN INTO t1 VALUES (id) ELSE INTO t2 VALUES (id) SELECT id FROM src",
			want:      1000,
		},
		{
			name:      "merge_without_cardinality",
			statement: "MERGE INTO t2 USING t ON (t2.id = t.id) WHEN MATCHED THEN UPDATE SET t2.v = t.v",
			plan: [][]driver.Value{
				{"0", nil, "MERGE STATEMENT", "1000"},
				{"1", "0", "MERGE", nil},
				{"2", "1", "VIEW", nil},
			},
			explained: "MERGE " + hint + " INTO t2 USING t ON (t2.id = t.id) WHEN MATCHED THEN UPDATE SET t2.v = t.v",
			wantErr:   true,
		},
		{
			name:      "null_cardinality",
			statement: "DELETE FROM t;",
			plan:      [][]driver.Value{{"0", nil, "DELETE STATEMENT", nil}},
			explained: "DELETE " + hint + " FROM t",
			wantErr:   true,
		},
		{
			name:      "missing_statement_row",
			statement: "DELETE FROM t;",
			explained: "DELETE " + hint + " FROM t",
			wantErr:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connector := &fakePlanConnector{plan: tc.plan}
			db := sql.OpenDB(connector)
			defer db.Close()
			// Without idle connections, every statement that is not pinned to one connection opens a new one.
			db.SetMaxIdleConns(0)

			got, err := (&Driver{db: db}).CountAffectedRows(context.Background(), tc.statement)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}

			require.NotEmpty(t, connector.queries)
			match := regexp.MustCompile(`^EXPLAIN PLAN SET STATEMENT_ID = '(\d+)' FOR `).FindStringSubmatch(connector.queries[0].text)
			require.NotNil(t, match, connector.queries[0].text)
			id := match[1]
			require.Equal(t, []fakePlanQuery{
				{conn: 1, text: "EXPLAIN PLAN SET STATEMENT_ID = '" + id + "' FOR " + tc.explained},
				{conn: 1, text: "SELECT ID, PARENT_ID, OPERATION, CARDINALITY FROM PLAN_TABLE WHERE STATEMENT_ID = '" + id + "' ORDER BY ID"},
				{conn: 1, text: "DELETE FROM PLAN_TABLE WHERE STATEMENT_ID = '" + id + "'"},
			}, connector.queries)
		})
	}
}

func TestWithDynamicSamplingHint(t *testing.T) {
	for _, tc := range []struct {
		statement string
		want      string
	}{
		{
			statement: "update t set c = 1",
			want:      "update /*+ OPT_PARAM('optimizer_dynamic_sampling' 11) */ t set c = 1",
		},
		{
			statement: "INSERT INTO t2 SELECT * FROM t",
			want:      "INSERT /*+ OPT_PARAM('optimizer_dynamic_sampling' 11) */ INTO t2 SELECT * FROM t",
		},
		{
			statement: "-- clean up\n/* ticket 42 */ DELETE FROM t",
			want:      "-- clean up\n/* ticket 42 */ DELETE /*+ OPT_PARAM('optimizer_dynamic_sampling' 11) */ FROM t",
		},
		{
			statement: "UPDATE /*+ INDEX(t t_c) */ t SET v = 1 WHERE c = 1",
			want:      "UPDATE /*+ OPT_PARAM('optimizer_dynamic_sampling' 11)  INDEX(t t_c) */ t SET v = 1 WHERE c = 1",
		},
		{
			statement: "DELETE --+ FULL(t)\nFROM t",
			want:      "DELETE --+ OPT_PARAM('optimizer_dynamic_sampling' 11)  FULL(t)\nFROM t",
		},
		{
			statement: "SELECT * FROM t",
			want:      "SELECT * FROM t",
		},
	} {
		t.Run(tc.statement, func(t *testing.T) {
			require.Equal(t, tc.want, withDynamicSamplingHint(tc.statement))
		})
	}
}

// fakePlanConnector returns the plan rows for every query and records each statement with the
// number of the connection that ran it.
type fakePlanConnector struct {
	plan    [][]driver.Value
	opened  int
	queries []fakePlanQuery
}

type fakePlanQuery struct {
	conn int
	text string
}

func (c *fakePlanConnector) Connect(context.Context) (driver.Conn, error) {
	c.opened++
	return &fakePlanConn{connector: c, id: c.opened}, nil
}

func (*fakePlanConnector) Driver() driver.Driver {
	return nil
}

type fakePlanConn struct {
	connector *fakePlanConnector
	id        int
}

func (*fakePlanConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*fakePlanConn) Close() error {
	return nil
}

func (*fakePlanConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *fakePlanConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.connector.queries = append(c.connector.queries, fakePlanQuery{conn: c.id, text: query})
	return driver.RowsAffected(0), nil
}

func (c *fakePlanConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.connector.queries = append(c.connector.queries, fakePlanQuery{conn: c.id, text: query})
	return &fakePlanRows{rows: c.connector.plan}, nil
}

type fakePlanRows struct {
	rows [][]driver.Value
}

func (*fakePlanRows) Columns() []string {
	return []string{"ID", "PARENT_ID", "OPERATION", "CARDINALITY"}
}

func (*fakePlanRows) Close() error {
	return nil
}

func (r *fakePlanRows) Next(dest []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(dest, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}
