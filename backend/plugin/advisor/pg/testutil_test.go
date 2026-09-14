package pg

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// explainFake stands in for PostgreSQL behind advisor.Query. It answers `EXPLAIN (FORMAT JSON)` or
// `EXPLAIN` with the plan registered for the statement and fails to explain any other statement.
type explainFake struct {
	plans map[string]string

	mu        sync.Mutex
	explained []string
}

func newExplainFakeDB(t *testing.T, plans map[string]string) (*sql.DB, *explainFake) {
	t.Helper()
	fake := &explainFake{plans: plans}
	db := sql.OpenDB(fake)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, fake
}

func (f *explainFake) Connect(context.Context) (driver.Conn, error) {
	return explainFakeConn{fake: f}, nil
}

func (f *explainFake) Driver() driver.Driver {
	return explainFakeDriver{fake: f}
}

func (f *explainFake) explainedStatements() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.explained
}

type explainFakeDriver struct{ fake *explainFake }

func (d explainFakeDriver) Open(string) (driver.Conn, error) {
	return explainFakeConn(d), nil
}

type explainFakeConn struct{ fake *explainFake }

func (explainFakeConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.Errorf("unexpected prepare of %q", query)
}

func (explainFakeConn) Close() error { return nil }

func (explainFakeConn) Begin() (driver.Tx, error) { return explainFakeTx{}, nil }

func (explainFakeConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.ResultNoRows, nil
}

func (c explainFakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	statement, ok := strings.CutPrefix(query, "EXPLAIN (FORMAT JSON) ")
	if !ok {
		statement, ok = strings.CutPrefix(query, "EXPLAIN ")
	}
	if !ok {
		return nil, errors.Errorf("unexpected query %q", query)
	}
	c.fake.mu.Lock()
	c.fake.explained = append(c.fake.explained, statement)
	c.fake.mu.Unlock()
	plan, ok := c.fake.plans[statement]
	if !ok {
		return nil, errors.Errorf("relation in %q does not exist", statement)
	}
	return &explainFakeRows{plan: plan}, nil
}

type explainFakeTx struct{}

func (explainFakeTx) Commit() error   { return nil }
func (explainFakeTx) Rollback() error { return nil }

type explainFakeRows struct {
	plan string
	done bool
}

func (*explainFakeRows) Columns() []string { return []string{"QUERY PLAN"} }

func (*explainFakeRows) Close() error { return nil }

func (r *explainFakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = r.plan
	r.done = true
	return nil
}

// modifyTablePlan returns a minimal `EXPLAIN (FORMAT JSON)` plan whose ModifyTable node is fed rows.
func modifyTablePlan(rows int) string {
	return fmt.Sprintf(`[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 0, "Plans": [{"Node Type": "Seq Scan", "Parent Relationship": "Outer", "Plan Rows": %d}]}}]`, rows)
}

func runRowLimitRule(t *testing.T, db *sql.DB, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, limit int32, statement string) []*storepb.Advice {
	t.Helper()
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{{
		Type:    ruleType,
		Level:   level,
		Engine:  storepb.Engine_POSTGRES,
		Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: limit}},
	}}, advisor.Context{
		DBType:          storepb.Engine_POSTGRES,
		Driver:          db,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}
