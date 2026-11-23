package tidb

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"

	// The review check parses with Bytebase's TiDB parser and runs the engine's
	// registered advisors, neither of which this package otherwise imports.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

type fakeProbeDriver struct {
	sessions []*recordingProbeSession
	handle   func(stmt string) ([]*omnidriver.Result, error)
}

func (*fakeProbeDriver) Ping(context.Context) error { return nil }
func (*fakeProbeDriver) Close() error               { return nil }

func (f *fakeProbeDriver) Session(context.Context) (omnidriver.Session, error) {
	sess := &recordingProbeSession{handle: f.handle}
	f.sessions = append(f.sessions, sess)
	return sess, nil
}

type recordingProbeSession struct {
	statements  []string
	rollbackCtx context.Context
	closed      bool
	handle      func(stmt string) ([]*omnidriver.Result, error)
}

func (s *recordingProbeSession) Query(ctx context.Context, script string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	trimmed := strings.TrimSpace(script)
	s.statements = append(s.statements, trimmed)
	if trimmed == "ROLLBACK" {
		s.rollbackCtx = ctx
		return []*omnidriver.Result{{}}, nil
	}
	if trimmed == "START TRANSACTION" {
		return []*omnidriver.Result{{}}, nil
	}
	if s.handle != nil {
		return s.handle(trimmed)
	}
	return []*omnidriver.Result{{}}, nil
}

func (s *recordingProbeSession) Close() error {
	s.closed = true
	return nil
}

// TestDryRunRuleProbesThroughTheAdapter pins the advisor-to-adapter seam:
// STATEMENT_DML_DRY_RUN probes through Driver.Probe in a rolled-back
// transaction for regular DML and in autocommit mode for BATCH statements.
func TestDryRunRuleProbesThroughTheAdapter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for _, tc := range []struct {
		name           string
		statement      string
		probeErr       error
		wantAdvice     bool
		wantAutocommit bool
	}{
		{
			name:      "a dry run the server accepts gives no advice",
			statement: "DELETE FROM app.t WHERE id = 1;",
		},
		{
			name:       "a dry run the server refuses gives advice",
			statement:  "DELETE FROM app.t WHERE missing_column = 1;",
			probeErr:   errors.New("Unknown column 'missing_column' in 'where clause'"),
			wantAdvice: true,
		},
		{
			name:           "a batch dry run outside a transaction gives no advice",
			statement:      "BATCH ON id LIMIT 2 DELETE FROM app.t WHERE id > 0;",
			wantAutocommit: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeProbeDriver{
				handle: func(_ string) ([]*omnidriver.Result, error) {
					if tc.probeErr != nil {
						//nolint:nilerr // a refused statement's error rides on Result.Err, as the contract specifies.
						return []*omnidriver.Result{{Err: tc.probeErr}}, nil
					}
					return []*omnidriver.Result{{
						Columns: []omnidriver.Column{{Name: "id", TypeName: "VARCHAR"}},
						Rows:    [][]omnidriver.Value{{{Kind: omnidriver.String, String: "Delete_1"}}},
					}}, nil
				},
			}
			driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

			adviceList, err := advisor.SQLReviewCheck(ctx, sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_TIDB,
			}}, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Prober:          driver,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			if tc.wantAutocommit {
				// BATCH dry-run probes the BATCH ... DRY RUN statement in autocommit
				// mode and then EXPLAINs the inner DML inside a rolled-back transaction.
				require.Len(t, fake.sessions, 2)
				require.True(t, fake.sessions[0].closed)
				require.Len(t, fake.sessions[0].statements, 1)
				require.Contains(t, fake.sessions[0].statements[0], "DRY RUN")

				require.True(t, fake.sessions[1].closed)
				require.Len(t, fake.sessions[1].statements, 3)
				require.Equal(t, "START TRANSACTION", fake.sessions[1].statements[0])
				require.True(t, strings.HasPrefix(fake.sessions[1].statements[1], "EXPLAIN "))
				require.Equal(t, "ROLLBACK", fake.sessions[1].statements[2])
			} else {
				require.Len(t, fake.sessions, 1)
				require.True(t, fake.sessions[0].closed)
				require.Len(t, fake.sessions[0].statements, 3)
				require.Equal(t, "START TRANSACTION", fake.sessions[0].statements[0])
				require.True(t, strings.HasPrefix(fake.sessions[0].statements[1], "EXPLAIN "))
				require.Equal(t, "ROLLBACK", fake.sessions[0].statements[2])
			}

			if !tc.wantAdvice {
				require.Empty(t, adviceList)
				return
			}
			require.Len(t, adviceList, 1)
			require.Contains(t, adviceList[0].GetContent(), "dry runs failed")
		})
	}
}

// TestProbeHonoursAutocommit pins transactional pre-executions, rollback on a
// canceled context, autocommit validation, and probeCell timestamp formatting.
func TestProbeHonoursAutocommit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ts := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)
	tsNoFrac := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	fake := &fakeProbeDriver{
		handle: func(stmt string) ([]*omnidriver.Result, error) {
			if strings.HasPrefix(stmt, "INSERT ") {
				return []*omnidriver.Result{{}}, nil
			}
			return []*omnidriver.Result{{
				Columns: []omnidriver.Column{
					{Name: "cnt", TypeName: "BIGINT"},
					{Name: "ts3", TypeName: "DATETIME"},
					{Name: "ts0", TypeName: "TIMESTAMP"},
					{Name: "nul", TypeName: "VARCHAR"},
				},
				Rows: [][]omnidriver.Value{{
					{Kind: omnidriver.Int64, Int64: 1},
					{Kind: omnidriver.Timestamp, Time: ts, Precision: 3},
					{Kind: omnidriver.TimestampTZ, Time: tsNoFrac, Precision: 0},
					{Kind: omnidriver.Null},
				}},
			}}, nil
		},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
	require.Implements(t, (*db.RowEstimator)(nil), driver)

	// Transactional probe skips empty pre-executions, runs non-empty ones,
	// formats temporal cells via probeCell, and rolls back with a detached ctx.
	probeCtx, cancel := context.WithCancel(ctx)
	cancel()
	probe, err := driver.Probe(probeCtx, db.ProbeContext{
		PreExecutions: []string{"", "INSERT INTO app.t VALUES (42)"},
	}, "SELECT COUNT(*) FROM t")
	require.NoError(t, err)
	require.Equal(t, []string{"cnt", "ts3", "ts0", "nul"}, probe.Columns)
	require.Equal(t, [][]any{{int64(1), "2026-03-04 05:06:07.123", "2026-03-04 05:06:07", nil}}, probe.Rows)
	require.Len(t, fake.sessions, 1)
	require.Equal(t, []string{
		"START TRANSACTION",
		"INSERT INTO app.t VALUES (42)",
		"SELECT COUNT(*) FROM t",
		"ROLLBACK",
	}, fake.sessions[0].statements)
	require.NoError(t, fake.sessions[0].rollbackCtx.Err(), "rollback must ignore caller cancellation")

	// Autocommit runs the statement directly without opening a transaction.
	_, err = driver.Probe(ctx, db.ProbeContext{Autocommit: true}, "BATCH ON id LIMIT 2 DRY RUN DELETE FROM app.t WHERE id > 0")
	require.NoError(t, err)
	require.Len(t, fake.sessions, 2)
	require.Equal(t, []string{"BATCH ON id LIMIT 2 DRY RUN DELETE FROM app.t WHERE id > 0"}, fake.sessions[1].statements)

	// Pre-executions cannot run outside a transaction.
	_, err = driver.Probe(ctx, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET @x = 1"}}, "SELECT 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-executions cannot run outside a transaction")
	require.Len(t, fake.sessions, 2, "invalid probe must fail before opening a session")
}
