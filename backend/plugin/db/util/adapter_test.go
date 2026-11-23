//nolint:revive
package util

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

type fakeOmniDriver struct {
	pingErr         error
	closeErr        error
	closed          bool
	syncInstanceOut *omnidriver.Instance
	syncInstanceErr error
	syncDBOut       *metadatapb.DatabaseSchemaMetadata
	syncDBErr       error
	dumpErr         error
	dumpText        string
	sessions        []*fakeOmniSession
	sessionErr      error
	sessionCalls    int
	execScript      string
	execOpts        omnidriver.ExecuteOptions
	execAffected    int64
	execErr         error
}

func (f *fakeOmniDriver) Ping(context.Context) error { return f.pingErr }
func (f *fakeOmniDriver) Close() error {
	f.closed = true
	return f.closeErr
}
func (f *fakeOmniDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.syncInstanceOut, f.syncInstanceErr
}
func (f *fakeOmniDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.syncDBOut, f.syncDBErr
}
func (f *fakeOmniDriver) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := io.WriteString(w, f.dumpText)
	return err
}
func (f *fakeOmniDriver) Session(context.Context) (omnidriver.Session, error) {
	f.sessionCalls++
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}
	if len(f.sessions) == 0 {
		return &fakeOmniSession{}, nil
	}
	s := f.sessions[0]
	f.sessions = f.sessions[1:]
	return s, nil
}
func (f *fakeOmniDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if opts.Log != nil {
		opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
		opts.Log.CommandResponse(f.execAffected, []int64{f.execAffected}, nil)
	}
	return f.execAffected, f.execErr
}

type fakeOmniSession struct {
	results   []*omnidriver.Result
	err       error
	closeErr  error
	closed    bool
	inFlight  atomic.Int32
	overlap   atomic.Bool
	queryFunc func(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error)
}

func (s *fakeOmniSession) Query(ctx context.Context, stmt string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	if s.inFlight.Add(1) > 1 {
		s.overlap.Store(true)
	}
	defer s.inFlight.Add(-1)
	if s.queryFunc != nil {
		return s.queryFunc(ctx, stmt, opts)
	}
	time.Sleep(2 * time.Millisecond)
	return s.results, s.err
}

func (s *fakeOmniSession) Close() error {
	s.closed = true
	return s.closeErr
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

func TestSessionHolderLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("lazy open reuses and serializes calls", func(t *testing.T) {
		s1 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1"}}}
		opens := 0
		var holder sessionHolder

		var wg sync.WaitGroup
		for range 6 {
			wg.Go(func() {
				res, err := holder.query(ctx, func(context.Context) (omnidriver.Session, error) {
					opens++
					return s1, nil
				}, SessionQueryOptions{}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
					return s.Query(ctx, "SELECT 1", omnidriver.QueryOptions{})
				})
				require.NoError(t, err)
				require.Len(t, res, 1)
			})
		}
		wg.Wait()
		require.Equal(t, 1, opens)
		require.False(t, s1.overlap.Load())
		require.Same(t, s1, holder.Session())
		require.NoError(t, holder.close())
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())
	})

	t.Run("error discards session by default and keeps partial results", func(t *testing.T) {
		s1 := &fakeOmniSession{
			results: []*omnidriver.Result{{Statement: "SELECT 1"}},
			err:     errors.New("connection lost"),
		}
		holder := sessionHolder{session: s1}
		res, err := holder.query(ctx, nil, SessionQueryOptions{}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(ctx, "SELECT 1", omnidriver.QueryOptions{})
		})
		require.ErrorContains(t, err, "connection lost")
		require.Len(t, res, 1)
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())
	})

	t.Run("KeepOnError preserves session unless context is canceled", func(t *testing.T) {
		recErr := errors.New("syntax error")
		s1 := &fakeOmniSession{err: recErr}
		holder := sessionHolder{session: s1}
		opts := SessionQueryOptions{
			KeepOnError: func(_ context.Context, err error) bool {
				return errors.Is(err, recErr)
			},
		}

		_, err := holder.query(ctx, nil, opts, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(ctx, "BAD", omnidriver.QueryOptions{})
		})
		require.ErrorIs(t, err, recErr)
		require.False(t, s1.closed)
		require.Same(t, s1, holder.Session())

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = holder.query(canceledCtx, nil, opts, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(canceledCtx, "BAD", omnidriver.QueryOptions{})
		})
		require.ErrorIs(t, err, recErr)
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())
	})

	t.Run("ShouldDiscardResults and context cancel discard session", func(t *testing.T) {
		s1 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1", Err: errors.New("bad")}}}
		holder := sessionHolder{session: s1}
		_, err := holder.query(ctx, nil, SessionQueryOptions{
			ShouldDiscardResults: func(_ context.Context, _ omnidriver.Session, results []*omnidriver.Result) bool {
				return len(results) > 0 && results[0].Err != nil
			},
		}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(ctx, "SELECT 1", omnidriver.QueryOptions{})
		})
		require.NoError(t, err)
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())

		s2 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1"}}}
		holder.SetSession(s2)
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = holder.query(canceledCtx, nil, SessionQueryOptions{}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(canceledCtx, "SELECT 1", omnidriver.QueryOptions{})
		})
		require.NoError(t, err)
		require.True(t, s2.closed)
		require.Nil(t, holder.Session())
	})
}

func TestOmniAdapterMethods(t *testing.T) {
	ctx := context.Background()
	s1 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1"}}}
	fake := &fakeOmniDriver{
		syncInstanceOut: &omnidriver.Instance{Version: "1.0", LowerCaseTableNames: 1},
		syncDBOut:       &metadatapb.DatabaseSchemaMetadata{Name: "app"},
		dumpText:        "CREATE TABLE t (id INT);",
		sessions:        []*fakeOmniSession{s1},
		execAffected:    3,
	}
	var closedOrder []string
	adapter := &OmniAdapter{
		Driver:               fake,
		WithInstanceMetadata: true,
		NotFoundResource:     "database",
		DatabaseName:         "app",
	}
	adapter.addCloser(closerFunc(func() error {
		closedOrder = append(closedOrder, "ssh")
		return nil
	}))
	adapter.AddCloseFunc(func() error {
		closedOrder = append(closedOrder, "dialer")
		return nil
	})

	require.NoError(t, adapter.Ping(ctx))

	inst, err := adapter.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "1.0", inst.Version)
	require.NotNil(t, inst.Metadata)

	schema, err := adapter.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "app", schema.Name)

	fake.syncDBErr = omnidriver.ErrNotFound
	_, err = adapter.SyncDBSchema(ctx)
	require.ErrorContains(t, err, `database "app" not found`)
	require.Equal(t, common.NotFound, common.ErrorCode(err))

	var dump bytes.Buffer
	require.NoError(t, adapter.Dump(ctx, &dump, schema))
	require.Equal(t, "CREATE TABLE t (id INT);", dump.String())

	res, err := adapter.QueryHeldSession(ctx, "SELECT 1", omnidriver.QueryOptions{}, SessionQueryOptions{})
	require.NoError(t, err)
	require.Len(t, res, 1)

	var ephemeralClosed bool
	fake.sessions = append(fake.sessions, &fakeOmniSession{})
	require.NoError(t, adapter.QueryEphemeral(ctx, func(s omnidriver.Session) error {
		ephemeralClosed = s.(*fakeOmniSession).closed
		return nil
	}))
	require.False(t, ephemeralClosed)

	var logs []*storepb.TaskRunLog
	affected, err := adapter.ExecuteScript(ctx, "-- txn-mode=off\nINSERT INTO t VALUES (1)", db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}, ExecuteConfig{
		ParseTransactionDirective: true,
		PassIsolation:             true,
		PassCreateDatabase:        true,
		PassMaxRetries:            true,
		ShiftCommandRanges:        true,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), affected)
	require.Equal(t, "INSERT INTO t VALUES (1)", fake.execScript)
	require.Equal(t, omnidriver.AutoCommit, fake.execOpts.Transaction)
	require.True(t, fake.execOpts.CreateDatabase)
	require.Equal(t, 2, fake.execOpts.MaxRetries)
	require.Len(t, logs, 2)
	// Shifted range points into the original script with the directive line.
	require.Equal(t, int32(len("-- txn-mode=off\n")), logs[0].GetCommandExecute().GetRange().GetStart())

	require.NoError(t, adapter.Close(ctx))
	require.True(t, s1.closed)
	require.True(t, fake.closed)
	require.Equal(t, []string{"ssh", "dialer"}, closedOrder)
	// Second Close does not re-run closers.
	closedOrder = nil
	require.NoError(t, adapter.Close(ctx))
	require.Empty(t, closedOrder)
}

func TestOmniAdapterOpenWithAndSessionHelpers(t *testing.T) {
	ctx := context.Background()

	t.Run("OpenWith initializes base and sets driver on success", func(t *testing.T) {
		var adapter OmniAdapter
		fake := &fakeOmniDriver{}
		require.NoError(t, adapter.OpenWith(ctx, AdapterOptions{
			WithInstanceMetadata: true,
			NotFoundResource:     "database",
			DatabaseName:         "app",
		}, func(*OmniAdapter) (omnidriver.Driver, error) {
			return fake, nil
		}))
		require.Same(t, fake, adapter.Driver)
		require.True(t, adapter.WithInstanceMetadata)
		require.Equal(t, "database", adapter.NotFoundResource)
		require.Equal(t, "app", adapter.DatabaseName)
	})

	t.Run("OpenWith closes registered closers on error", func(t *testing.T) {
		var adapter OmniAdapter
		closed := false
		err := adapter.OpenWith(ctx, AdapterOptions{WithInstanceMetadata: true}, func(a *OmniAdapter) (omnidriver.Driver, error) {
			a.AddCloseFunc(func() error {
				closed = true
				return nil
			})
			return nil, errors.New("dial failed")
		})
		require.EqualError(t, err, "dial failed")
		require.True(t, closed)
		require.Nil(t, adapter.Driver)
	})

	t.Run("QueryHeldSession wraps OpenSession error when WrapOpenError is set", func(t *testing.T) {
		adapter := &OmniAdapter{Driver: &fakeOmniDriver{sessionErr: errors.New("pool exhausted")}}
		_, err := adapter.QueryHeldSession(ctx, "SELECT 1", omnidriver.QueryOptions{}, SessionQueryOptions{
			WrapOpenError: "failed to open a session",
		})
		require.EqualError(t, err, "failed to open a session: pool exhausted")
	})

	t.Run("QueryEphemeralSession runs query and closes session", func(t *testing.T) {
		s := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1"}}}
		adapter := &OmniAdapter{Driver: &fakeOmniDriver{sessions: []*fakeOmniSession{s}}}
		results, err := adapter.QueryEphemeralSession(ctx, "SELECT 1", omnidriver.QueryOptions{Limit: 10})
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.True(t, s.closed)
	})

	t.Run("OpenWith eager session retains session and cleans up on error", func(t *testing.T) {
		s := &fakeOmniSession{}
		fake := &fakeOmniDriver{sessions: []*fakeOmniSession{s}}
		var adapter OmniAdapter
		require.NoError(t, adapter.OpenWith(ctx, AdapterOptions{
			NotFoundResource: "dataset",
			DatabaseName:     "analytics",
			EagerSession:     true,
		}, func(*OmniAdapter) (omnidriver.Driver, error) {
			return fake, nil
		}))
		require.Same(t, fake, adapter.Driver)
		require.Same(t, s, adapter.Session())

		var failed OmniAdapter
		failingDriver := &fakeOmniDriver{sessionErr: errors.New("session failed")}
		closerClosed := false
		err := failed.OpenWith(ctx, AdapterOptions{EagerSession: true}, func(a *OmniAdapter) (omnidriver.Driver, error) {
			a.AddCloseFunc(func() error {
				closerClosed = true
				return nil
			})
			return failingDriver, nil
		})
		require.EqualError(t, err, "session failed")
		require.True(t, failingDriver.closed)
		require.True(t, closerClosed)
		require.Nil(t, failed.Driver)
		require.Nil(t, failed.Session())
	})

	t.Run("QueryHeldConn and QueryEphemeralConn convert results and propagate errors", func(t *testing.T) {
		s1 := &fakeOmniSession{results: []*omnidriver.Result{{
			Statement: "SELECT 1",
			Columns:   []omnidriver.Column{{Name: "n", TypeName: "INT"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 1}}},
		}}}
		s2 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 2"}}, err: errors.New("boom")}
		adapter := &OmniAdapter{Driver: &fakeOmniDriver{sessions: []*fakeOmniSession{s1, s2}}}

		got, err := adapter.QueryHeldConn(ctx, "SELECT 1", omnidriver.QueryOptions{MaxResultBytes: 1024}, SessionQueryOptions{}, QueryResultOptions{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, int64(1), got[0].Rows[0].Values[0].GetInt64Value())

		got, err = adapter.QueryEphemeralConn(ctx, "SELECT 2", omnidriver.QueryOptions{MaxResultBytes: 1024}, QueryResultOptions{})
		require.EqualError(t, err, "boom")
		require.Nil(t, got)
		require.True(t, s2.closed)

		s3 := &fakeOmniSession{results: []*omnidriver.Result{{
			Statement: "SELECT 3",
			Columns:   []omnidriver.Column{{Name: "n", TypeName: "INT"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 3}}},
		}}}
		adapter.Driver = &fakeOmniDriver{sessions: []*fakeOmniSession{s3}}
		got, err = adapter.QueryEphemeralConn(ctx, "SELECT 3", omnidriver.QueryOptions{MaxResultBytes: 1024}, QueryResultOptions{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, int64(3), got[0].Rows[0].Values[0].GetInt64Value())
		require.True(t, s3.closed)
	})

	t.Run("BuildExecuteOptions maps directives and flags", func(t *testing.T) {
		stmt, opts := BuildExecuteOptions("-- txn-mode=off\n-- txn-isolation=serializable\nDELETE FROM t;", db.ExecuteOptions{
			CreateDatabase: true,
			MaximumRetries: 3,
		}, ExecuteConfig{
			ParseTransactionDirective: true,
			PassIsolation:             true,
			PassCreateDatabase:        true,
			PassMaxRetries:            true,
		})
		require.Equal(t, "DELETE FROM t;", stmt)
		require.Equal(t, omnidriver.AutoCommit, opts.Transaction)
		require.Equal(t, omnidriver.IsolationLevel("SERIALIZABLE"), opts.Isolation)
		require.True(t, opts.CreateDatabase)
		require.Equal(t, 3, opts.MaxRetries)
		require.NotNil(t, opts.Log)
	})
}
