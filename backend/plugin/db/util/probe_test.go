//nolint:revive
package util

import (
	"context"
	"strings"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestProbeResultPlan(t *testing.T) {
	// OceanBase splits its plan across rows; PostgreSQL prints it as one cell.
	plan, err := db.ProbeResult{Rows: [][]any{{`{"ID":0,`}, {`"OPERATOR":"X"}`}}}.Plan()
	require.NoError(t, err)
	require.Equal(t, `{"ID":0,"OPERATOR":"X"}`, plan)

	_, err = db.ProbeResult{}.Plan()
	require.Error(t, err)
}

func TestProbeOmni(t *testing.T) {
	ctx := context.Background()

	t.Run("transactional probe prepares, runs pre-executions, rolls back on error", func(t *testing.T) {
		var ran []string
		s := &fakeOmniSession{
			queryFunc: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				ran = append(ran, stmt)
				if strings.HasPrefix(stmt, "EXPLAIN") {
					return []*omnidriver.Result{{
						Columns: []omnidriver.Column{{Name: "PLAN", TypeName: "TEXT"}},
						Rows:    [][]omnidriver.Value{{{Kind: omnidriver.String, String: "ok"}}},
					}}, nil
				}
				return []*omnidriver.Result{{}}, nil
			},
		}
		q := &fakeOmniDriver{sessions: []*fakeOmniSession{s}}
		adapter := &OmniAdapter{Driver: q}

		res, err := adapter.ProbeOmni(ctx, db.ProbeContext{PreExecutions: []string{"", "SET a = 1"}}, "EXPLAIN SELECT 1", OmniProbeOptions{
			BeginStatement:           "START TRANSACTION",
			RollbackStatement:        "ROLLBACK",
			RollbackWithoutCancel:    true,
			SkipEmptyPreExecutions:   true,
			CheckPreExecutionResults: true,
			PrepareTx: func(context.Context, omnidriver.Session) ([]string, error) {
				return []string{"SET ROLE owner"}, nil
			},
		})
		require.NoError(t, err)
		require.True(t, s.closed)
		require.Equal(t, []string{"START TRANSACTION", "SET ROLE owner", "SET a = 1", "EXPLAIN SELECT 1", "ROLLBACK"}, ran)
		require.Equal(t, [][]any{{"ok"}}, res.Rows)
	})

	t.Run("autocommit only and pre-execution validation", func(t *testing.T) {
		q := &fakeOmniDriver{}
		adapter := &OmniAdapter{Driver: q}
		_, err := adapter.ProbeOmni(ctx, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET a = 1"}}, "SELECT 1", OmniProbeOptions{})
		require.ErrorContains(t, err, "cannot run outside a transaction")

		_, err = adapter.ProbeOmni(ctx, db.ProbeContext{}, "SELECT 1", OmniProbeOptions{AutocommitOnly: true})
		require.ErrorContains(t, err, "autocommit-only")
		require.Equal(t, 0, q.sessionCalls)
	})

	t.Run("result error triggers rollback and surfaces error", func(t *testing.T) {
		var ran []string
		s := &fakeOmniSession{
			queryFunc: func(_ context.Context, stmt string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
				ran = append(ran, stmt)
				if stmt == "EXPLAIN BAD" {
					return []*omnidriver.Result{{Err: errors.New("bad sql")}}, nil
				}
				return []*omnidriver.Result{{}}, nil
			},
		}
		adapter := &OmniAdapter{Driver: &fakeOmniDriver{sessions: []*fakeOmniSession{s}}}
		_, err := adapter.ProbeOmni(ctx, db.ProbeContext{}, "EXPLAIN BAD", OmniProbeOptions{
			BeginStatement:    "BEGIN;",
			RollbackStatement: "ROLLBACK;",
		})
		require.ErrorContains(t, err, "bad sql")
		require.Equal(t, []string{"BEGIN;", "EXPLAIN BAD", "ROLLBACK;"}, ran)
	})
}
