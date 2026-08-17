package sampleprojectinstance

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunnerCleansUpImmediatelyAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	manager := &cleanupManagerStub{calls: make(chan time.Time, 1)}
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	runner := NewRunner(manager, Options{Clock: func() time.Time { return now }})

	var wg sync.WaitGroup
	wg.Add(1)
	go runner.Run(ctx, &wg)

	require.Equal(t, now, <-manager.calls)
	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestRunnerRunOnceUsesInjectedClock(t *testing.T) {
	manager := &cleanupManagerStub{calls: make(chan time.Time, 1)}
	now := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	runner := NewRunner(manager, Options{Clock: func() time.Time { return now }})

	require.NoError(t, runner.RunOnce(context.Background()))
	require.Equal(t, now, <-manager.calls)
}

type cleanupManagerStub struct {
	calls chan time.Time
}

func (m *cleanupManagerStub) Cleanup(_ context.Context, now time.Time) error {
	m.calls <- now
	return nil
}
