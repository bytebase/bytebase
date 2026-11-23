package oceanbase

import (
	"context"
	"errors"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestProbeRefusesPreExecutionsOutsideATransaction pins the contract's rule:
// Autocommit exists for a statement the engine refuses inside a transaction, so
// pre-executions, which need one, are refused rather than run beside it.
func TestProbeRefusesPreExecutionsOutsideATransaction(t *testing.T) {
	_, err := (&Driver{}).Probe(context.Background(), db.ProbeContext{
		Autocommit:    true,
		PreExecutions: []string{"SET NAMES utf8mb4"},
	}, "SELECT 1")
	require.ErrorContains(t, err, "cannot run outside a transaction")
}

type recordingProbeSession struct {
	statements   []string
	statementErr error
	closed       bool
}

func (s *recordingProbeSession) Query(_ context.Context, script string, _ odriver.QueryOptions) ([]*odriver.Result, error) {
	s.statements = append(s.statements, script)
	switch script {
	case "START TRANSACTION", "ROLLBACK":
		return []*odriver.Result{{}}, nil
	case "SET @probe_value = 7":
		return []*odriver.Result{{}}, nil
	default:
		if s.statementErr != nil {
			//nolint:nilerr // a refused statement's error rides on its result.
			return []*odriver.Result{{Err: s.statementErr}}, nil
		}
		return []*odriver.Result{{
			Columns: []odriver.Column{{Name: "Query Plan", TypeName: "VARCHAR"}},
			Rows:    [][]odriver.Value{{{Kind: odriver.String, String: `{"EST.ROWS":2}`}}},
		}}, nil
	}
}

func (s *recordingProbeSession) Close() error {
	s.closed = true
	return nil
}

type fakeProbeDriver struct {
	session      *recordingProbeSession
	affectedRows int64
	affectedErr  error
	columnExtra  string
	columnErr    error
	gotCountStmt string
	gotExtraArgs [3]string
}

func (*fakeProbeDriver) Ping(context.Context) error { return nil }
func (*fakeProbeDriver) Close() error               { return nil }
func (f *fakeProbeDriver) Session(context.Context) (odriver.Session, error) {
	return f.session, nil
}

func (f *fakeProbeDriver) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	f.gotCountStmt = statement
	return f.affectedRows, f.affectedErr
}

func (f *fakeProbeDriver) ColumnExtra(_ context.Context, database, table, column string) (string, error) {
	f.gotExtraArgs = [3]string{database, table, column}
	return f.columnExtra, f.columnErr
}

func TestProbeTransactionAndAutocommit(t *testing.T) {
	ctx := context.Background()

	t.Run("transactional probe wraps pre-executions and rolls back", func(t *testing.T) {
		sess := &recordingProbeSession{}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeProbeDriver{session: sess}}}

		res, err := d.Probe(ctx, db.ProbeContext{
			PreExecutions: []string{"", "SET @probe_value = 7"},
		}, "EXPLAIN format=json UPDATE t SET name = 'x'")
		require.NoError(t, err)
		require.True(t, sess.closed)
		require.Equal(t, []string{
			"START TRANSACTION",
			"SET @probe_value = 7",
			"EXPLAIN format=json UPDATE t SET name = 'x'",
			"ROLLBACK",
		}, sess.statements)
		require.Equal(t, []string{"Query Plan"}, res.Columns)
		plan, err := res.Plan()
		require.NoError(t, err)
		require.Equal(t, `{"EST.ROWS":2}`, plan)
	})

	t.Run("failed probe statement still rolls back", func(t *testing.T) {
		boom := errors.New("table doesn't exist")
		sess := &recordingProbeSession{statementErr: boom}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeProbeDriver{session: sess}}}

		_, err := d.Probe(ctx, db.ProbeContext{}, "EXPLAIN UPDATE missing SET a = 1")
		require.ErrorIs(t, err, boom)
		require.Equal(t, []string{
			"START TRANSACTION",
			"EXPLAIN UPDATE missing SET a = 1",
			"ROLLBACK",
		}, sess.statements)
	})

	t.Run("autocommit runs the statement alone", func(t *testing.T) {
		sess := &recordingProbeSession{}
		d := &Driver{OmniAdapter: util.OmniAdapter{Driver: &fakeProbeDriver{session: sess}}}

		_, err := d.Probe(ctx, db.ProbeContext{Autocommit: true}, "EXPLAIN SELECT 1")
		require.NoError(t, err)
		require.Equal(t, []string{"EXPLAIN SELECT 1"}, sess.statements)
	})
}

func TestProbeCell(t *testing.T) {
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123000000, time.UTC)
	require.Equal(t, "raw", probeCell(odriver.Value{Kind: odriver.Bytes, Bytes: []byte("raw")}))
	require.Equal(t, int32(7), probeCell(odriver.Value{Kind: odriver.Int32, Int64: 7}))
	require.Equal(t, uint32(9), probeCell(odriver.Value{Kind: odriver.Uint32, Uint64: 9}))
	require.Nil(t, probeCell(odriver.Value{Kind: odriver.Structured, Structured: map[string]any{"a": 1}}))
	require.Equal(t, ts.Format(time.RFC3339Nano), probeCell(odriver.Value{Kind: odriver.Timestamp, Time: ts}))
	require.Equal(t, int64(42), probeCell(odriver.Value{Kind: odriver.Int64, Int64: 42}))
}

func TestRowEstimatorAndColumnExtraDelegation(t *testing.T) {
	ctx := context.Background()
	impl := &fakeProbeDriver{
		affectedRows: 2,
		columnExtra:  "STORED GENERATED",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

	rows, err := d.CountAffectedRows(ctx, "UPDATE t SET name = 'x'")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, "UPDATE t SET name = 'x'", impl.gotCountStmt)

	extra, err := d.ColumnExtra(ctx, "app", "t", "generated_col")
	require.NoError(t, err)
	require.Equal(t, "STORED GENERATED", extra)
	require.Equal(t, [3]string{"app", "t", "generated_col"}, impl.gotExtraArgs)
}
