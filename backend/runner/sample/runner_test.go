package sample

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
	manager := &cleanupManagerStub{calls: make(chan struct{}, 1)}
	runner := NewRunner(manager)

	var wg sync.WaitGroup
	wg.Add(1)
	go runner.Run(ctx, &wg)

	<-manager.calls
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

func TestRunnerRunOnceCallsCleanup(t *testing.T) {
	manager := &cleanupManagerStub{calls: make(chan struct{}, 1)}
	runner := NewRunner(manager)

	require.NoError(t, runner.RunOnce(context.Background()))
	<-manager.calls
}

type cleanupManagerStub struct {
	calls chan struct{}
}

func (m *cleanupManagerStub) Cleanup(context.Context) error {
	m.calls <- struct{}{}
	return nil
}
