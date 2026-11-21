package trino

import (
	"context"
	"errors"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// brokenSession is a session that cannot run a statement, as a connection the
// server closed under a canceled statement is.
type brokenSession struct{}

func (brokenSession) Query(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	return nil, errors.New("the session is gone")
}

func (brokenSession) Close() error { return nil }

// TestQueryConnDropsTheSessionAScriptCouldNotRun pins that a session a failed
// call took with it does not outlive the call: the editor's next statement must
// not run on a connection the server has closed.
func TestQueryConnDropsTheSessionAScriptCouldNotRun(t *testing.T) {
	d := &Driver{}
	d.SetSession(brokenSession{})
	_, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
	require.Error(t, err)
	require.Nil(t, d.Session(), "the next statement opens a new session")
}
