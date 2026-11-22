package starrocks

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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

type fakeStarRocksSession struct {
	gotOpts omnidriver.QueryOptions
	closed  bool
}

func (s *fakeStarRocksSession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotOpts = opts
	return []*omnidriver.Result{{Statement: script}}, nil
}

func (s *fakeStarRocksSession) Close() error {
	s.closed = true
	return nil
}

type fakeStarRocksDriver struct {
	session       *fakeStarRocksSession
	sessionCalls  int
	executeScript string
	executeOpts   omnidriver.ExecuteOptions
	instance      *omnidriver.Instance
	dumpDDL       string
}

func (*fakeStarRocksDriver) Ping(context.Context) error { return nil }
func (*fakeStarRocksDriver) Close() error               { return nil }

func (f *fakeStarRocksDriver) Session(context.Context) (omnidriver.Session, error) {
	f.sessionCalls++
	return f.session, nil
}

func (f *fakeStarRocksDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	return 2, nil
}

func (f *fakeStarRocksDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (*fakeStarRocksDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return &metadatapb.DatabaseSchemaMetadata{Name: "analytics"}, nil
}

func (f *fakeStarRocksDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func TestStarRocksAdapterSeam(t *testing.T) {
	ctx := context.Background()
	sess := &fakeStarRocksSession{}
	impl := &fakeStarRocksDriver{
		session: sess,
		instance: &omnidriver.Instance{
			Version:   "3.4.10",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "analytics"}},
		},
		dumpDDL: "CREATE TABLE `t` (`id` INT);\n",
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl, WithInstanceMetadata: true}}

	// Held session is opened lazily and reused across calls.
	_, err := d.QueryConn(ctx, "SET @x = 1", db.QueryContext{Limit: 3, MaskingEnabled: true, MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.Equal(t, 1, impl.sessionCalls)
	require.Equal(t, 3, sess.gotOpts.Limit)
	require.True(t, sess.gotOpts.MaskingEnabled)
	require.Same(t, sess, d.Session())

	_, err = d.QueryConn(ctx, "SELECT @x", db.QueryContext{MaximumSQLResultSize: 1024})
	require.NoError(t, err)
	require.Equal(t, 1, impl.sessionCalls)

	// Canceled context discards the held session.
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = d.QueryConn(canceledCtx, "SELECT 1", db.QueryContext{})
	require.NoError(t, err)
	require.True(t, sess.closed)
	require.Nil(t, d.Session())

	// Execute strips txn-mode, ignores isolation, and forwards options.
	affected, err := d.Execute(ctx, "-- txn-mode=off\n-- txn-isolation=serializable\nINSERT INTO t VALUES (1)", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)
	require.Equal(t, "INSERT INTO t VALUES (1)", impl.executeScript)
	require.Equal(t, omnidriver.AutoCommit, impl.executeOpts.Transaction)
	require.Empty(t, impl.executeOpts.Isolation)
	require.True(t, impl.executeOpts.CreateDatabase)
	require.Equal(t, 2, impl.executeOpts.MaxRetries)

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "3.4.10", inst.Version)

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, nil))
	require.Contains(t, out.String(), "CREATE TABLE `t`")
}
