package tidb

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// overlapSession records whether two statements ran on it at the same time,
// which is what one server session must never do.
type overlapSession struct {
	inFlight   atomic.Int32
	overlapped atomic.Bool
}

func (s *overlapSession) Query(context.Context, string, omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	if s.inFlight.Add(1) > 1 {
		s.overlapped.Store(true)
	}
	time.Sleep(5 * time.Millisecond)
	s.inFlight.Add(-1)
	return nil, nil
}

func (*overlapSession) Close() error { return nil }

// TestQueryConnSerializesStatementsOnTheSession pins that the lock covers the
// statement, not just the session's acquisition: two callers sharing a driver
// share one server session, and their statements must not interleave on it.
func TestQueryConnSerializesStatementsOnTheSession(t *testing.T) {
	session := &overlapSession{}
	d := &Driver{}
	d.SetSession(session)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			_, _ = d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
		})
	}
	wg.Wait()

	require.False(t, session.overlapped.Load(), "two statements ran on the one session at once")
}
