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
		var holder SessionHolder

		var wg sync.WaitGroup
		for range 6 {
			wg.Go(func() {
				res, err := holder.Query(ctx, func(context.Context) (omnidriver.Session, error) {
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
		require.NoError(t, holder.Close())
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())
	})

	t.Run("error discards session by default and keeps partial results", func(t *testing.T) {
		s1 := &fakeOmniSession{
			results: []*omnidriver.Result{{Statement: "SELECT 1"}},
			err:     errors.New("connection lost"),
		}
		holder := NewSessionHolder(s1)
		res, err := holder.Query(ctx, nil, SessionQueryOptions{}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
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
		holder := NewSessionHolder(s1)
		opts := SessionQueryOptions{
			KeepOnError: func(_ context.Context, err error) bool {
				return errors.Is(err, recErr)
			},
		}

		_, err := holder.Query(ctx, nil, opts, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(ctx, "BAD", omnidriver.QueryOptions{})
		})
		require.ErrorIs(t, err, recErr)
		require.False(t, s1.closed)
		require.Same(t, s1, holder.Session())

		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()
		_, err = holder.Query(canceledCtx, nil, opts, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
			return s.Query(canceledCtx, "BAD", omnidriver.QueryOptions{})
		})
		require.ErrorIs(t, err, recErr)
		require.True(t, s1.closed)
		require.Nil(t, holder.Session())
	})

	t.Run("ShouldDiscardResults and context cancel discard session", func(t *testing.T) {
		s1 := &fakeOmniSession{results: []*omnidriver.Result{{Statement: "SELECT 1", Err: errors.New("bad")}}}
		holder := NewSessionHolder(s1)
		_, err := holder.Query(ctx, nil, SessionQueryOptions{
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
		_, err = holder.Query(canceledCtx, nil, SessionQueryOptions{}, func(s omnidriver.Session) ([]*omnidriver.Result, error) {
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
	adapter.AddCloser(closerFunc(func() error {
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
