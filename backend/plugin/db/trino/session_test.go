package trino

import (
	"context"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type countingQuerierDriver struct {
	sessionOpens int
	driverClosed bool
	session      *recordingSession
}

func (*countingQuerierDriver) Ping(context.Context) error { return nil }
func (d *countingQuerierDriver) Close() error {
	d.driverClosed = true
	return nil
}

func (d *countingQuerierDriver) Session(context.Context) (omnidriver.Session, error) {
	d.sessionOpens++
	return d.session, nil
}

type recordingSession struct {
	scripts []string
	closed  bool
}

func (s *recordingSession) Query(_ context.Context, script string, _ omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.scripts = append(s.scripts, script)
	if len(s.scripts) == 1 {
		return []*omnidriver.Result{{Statement: script}}, nil
	}
	return []*omnidriver.Result{{
		Statement: script,
		Columns:   []omnidriver.Column{{Name: "Name", TypeName: "VARCHAR"}, {Name: "Value", TypeName: "VARCHAR"}},
		Rows: [][]omnidriver.Value{{
			{Kind: omnidriver.String, String: "query_max_run_time"},
			{Kind: omnidriver.String, String: "1m"},
		}},
	}}, nil
}

func (s *recordingSession) Close() error {
	s.closed = true
	return nil
}

// TestQueryConnKeepsTheSessionForTheDriver pins that separate QueryConn calls
// reuse one lazily opened omni Session until Close releases both the session
// and the driver.
func TestQueryConnKeepsTheSessionForTheDriver(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sess := &recordingSession{}
	fake := &countingQuerierDriver{session: sess}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 20}

	results, err := driver.QueryConn(ctx, "SET SESSION query_max_run_time = '1m'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)

	results, err = driver.QueryConn(ctx, "SHOW SESSION LIKE 'query_max_run_time'", queryContext)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Empty(t, results[0].Error)
	require.Len(t, results[0].Rows, 1)
	require.Equal(t, "1m", results[0].Rows[0].Values[1].GetStringValue())

	require.Equal(t, 1, fake.sessionOpens, "both QueryConn calls must share one session")
	require.Equal(t, []string{
		"SET SESSION query_max_run_time = '1m'",
		"SHOW SESSION LIKE 'query_max_run_time'",
	}, sess.scripts)

	require.NoError(t, driver.Close(ctx))
	require.True(t, sess.closed)
	require.True(t, fake.driverClosed)
}
