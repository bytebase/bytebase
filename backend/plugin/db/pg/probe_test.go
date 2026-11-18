package pg

import (
	"context"
	"errors"
	"strings"
	"testing"

	omndriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

var errProbeStatement = errors.New("the statement was refused")

// recordingSession stands in for the omni session a probe runs on: it records
// each script the probe sends, in order, and answers the owner lookup and the
// probed statement.
type recordingSession struct {
	statements []string
	// statementErr is what the probed statement answered, when set.
	statementErr error
}

func (s *recordingSession) Query(_ context.Context, script string, _ omndriver.QueryOptions) ([]*omndriver.Result, error) {
	s.statements = append(s.statements, strings.Join(strings.Fields(script), " "))
	switch {
	case strings.Contains(script, "pg_roles"):
		return []*omndriver.Result{{
			Columns: []omndriver.Column{{Name: "rolname", TypeName: "NAME"}},
			Rows:    [][]omndriver.Value{{{Kind: omndriver.String, String: "app_owner"}}},
		}}, nil
	case strings.HasPrefix(strings.TrimSpace(script), "BEGIN"), strings.HasPrefix(strings.TrimSpace(script), "ROLLBACK"), strings.HasPrefix(strings.TrimSpace(script), "SET"):
		return []*omndriver.Result{{}}, nil
	default:
		if s.statementErr != nil {
			//nolint:nilerr // a refused statement's error rides on its result, as the contract says.
			return []*omndriver.Result{{Err: s.statementErr}}, nil
		}
		return []*omndriver.Result{{
			Columns: []omndriver.Column{{Name: "QUERY PLAN", TypeName: "TEXT"}},
			Rows:    [][]omndriver.Value{{{Kind: omndriver.String, String: "plan"}}},
		}}, nil
	}
}

func (*recordingSession) Close() error { return nil }

// TestProbeTenantMode proves the probe reads the database owner and sets the
// session role before the statement, in the order the tenant-mode prelude has
// always used, and rolls the transaction back — all on the one session it opened.
func TestProbeTenantMode(t *testing.T) {
	session := &recordingSession{}
	driver := &Driver{connectionCtx: db.ConnectionContext{TenantMode: true}}

	_, err := driver.probeOnSession(context.Background(), session,
		db.ProbeContext{PreExecutions: []string{"SET search_path = app"}},
		"EXPLAIN (FORMAT JSON) UPDATE t SET c = 1")
	require.NoError(t, err)

	require.Equal(t, []string{
		"SELECT u.rolname FROM pg_roles AS u JOIN pg_database AS d ON (d.datdba = u.oid) WHERE d.datname = current_database();",
		"BEGIN;",
		"SET SESSION ROLE 'app_owner';",
		"SET search_path = app",
		"EXPLAIN (FORMAT JSON) UPDATE t SET c = 1",
		"ROLLBACK;",
	}, session.statements)
}

// TestProbeWithoutTenantMode proves a non-tenant probe runs the statement alone.
func TestProbeWithoutTenantMode(t *testing.T) {
	session := &recordingSession{}
	driver := &Driver{}

	_, err := driver.probeOnSession(context.Background(), session, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, []string{"BEGIN;", "SELECT CURRENT_USER", "ROLLBACK;"}, session.statements)
}

// TestProbeRollsBackAfterAFailedStatement proves a refused statement still ends
// its transaction, so the session does not return to the pool inside one.
func TestProbeRollsBackAfterAFailedStatement(t *testing.T) {
	session := &recordingSession{statementErr: errProbeStatement}
	driver := &Driver{}

	_, err := driver.probeOnSession(context.Background(), session, db.ProbeContext{}, "EXPLAIN INSERT INTO t VALUES (1)")
	require.ErrorIs(t, err, errProbeStatement)
	require.Equal(t, []string{"BEGIN;", "EXPLAIN INSERT INTO t VALUES (1)", "ROLLBACK;"}, session.statements)
}

// TestProbeAutocommitRefusesPreExecutions proves the contract's one refusal.
func TestProbeAutocommitRefusesPreExecutions(t *testing.T) {
	session := &recordingSession{}
	driver := &Driver{}

	_, err := driver.probeOnSession(context.Background(), session,
		db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET search_path = app"}}, "SELECT 1")
	require.ErrorContains(t, err, "cannot run outside a transaction")
	require.Empty(t, session.statements)
}

// TestProbeResultTypes proves a rule reads a text cell as a string, the shape the
// advisor rules type-assert.
func TestProbeResultTypes(t *testing.T) {
	session := &recordingSession{}
	driver := &Driver{}

	result, err := driver.probeOnSession(context.Background(), session, db.ProbeContext{}, "SELECT CURRENT_USER")
	require.NoError(t, err)
	require.Equal(t, []string{"QUERY PLAN"}, result.Columns)
	require.Equal(t, []string{"TEXT"}, result.ColumnTypes)
	require.Equal(t, [][]any{{"plan"}}, result.Rows)
}
