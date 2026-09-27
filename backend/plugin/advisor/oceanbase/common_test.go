package oceanbase

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	// testOceanBaseExplainPlan is the plan the fake OceanBase server returns for every EXPLAIN.
	testOceanBaseExplainPlan string
	// testOceanBaseExplainCount counts the EXPLAIN statements the fake server receives.
	testOceanBaseExplainCount int
)

func init() {
	sql.Register("test_oceanbase_advisor_explain", testOceanBaseExplainDriver{})
}

type testOceanBaseExplainDriver struct{}

func (testOceanBaseExplainDriver) Open(string) (driver.Conn, error) {
	return testOceanBaseExplainConn{}, nil
}

type testOceanBaseExplainConn struct{}

func (testOceanBaseExplainConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (testOceanBaseExplainConn) Close() error {
	return nil
}

func (testOceanBaseExplainConn) Begin() (driver.Tx, error) {
	return testOceanBaseExplainTx{}, nil
}

func (testOceanBaseExplainConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.HasPrefix(query, "EXPLAIN ") {
		testOceanBaseExplainCount++
	}
	return &testOceanBaseExplainRows{rows: [][]driver.Value{{testOceanBaseExplainPlan}}}, nil
}

type testOceanBaseExplainTx struct{}

func (testOceanBaseExplainTx) Commit() error {
	return nil
}

func (testOceanBaseExplainTx) Rollback() error {
	return nil
}

type testOceanBaseExplainRows struct {
	rows [][]driver.Value
	idx  int
}

func (*testOceanBaseExplainRows) Columns() []string {
	return []string{"Query Plan"}
}

func (*testOceanBaseExplainRows) Close() error {
	return nil
}

func (r *testOceanBaseExplainRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

// checkTestRowLimitRule runs one OceanBase row limit rule with a limit of 5 against the fake
// server, which answers every EXPLAIN with plan.
func checkTestRowLimitRule(t *testing.T, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, plan, statement string) []*storepb.Advice {
	t.Helper()
	testOceanBaseExplainPlan = plan
	testOceanBaseExplainCount = 0
	db, err := sql.Open("test_oceanbase_advisor_explain", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{
		{
			Type:    ruleType,
			Level:   level,
			Engine:  storepb.Engine_OCEANBASE,
			Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
		},
	}, advisor.Context{
		DBType:          storepb.Engine_OCEANBASE,
		Driver:          db,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}

func TestGetEstimatedRowsFromJSON(t *testing.T) {
	// OceanBase splits the JSON plan across rows, and EST.ROWS may be fractional.
	rows, err := getEstimatedRowsFromJSON([]any{
		[]string{"Query Plan"},
		[]string{"VARCHAR"},
		[]any{
			[]any{`{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE",`},
			[]any{`"NAME":"","EST.ROWS":41.6,"output":""}`},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), rows)

	_, err = getEstimatedRowsFromJSON([]any{
		[]string{"Query Plan"},
		[]string{"VARCHAR"},
		[]any{[]any{`{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE","NAME":""}`}},
	})
	require.ErrorContains(t, err, `operator "DISTRIBUTED UPDATE" has no EST.ROWS`)
}
