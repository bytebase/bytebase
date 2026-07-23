package leader

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type testPersistence struct {
	acquire func(context.Context, string, string, time.Duration) (int64, bool, error)
	renew   func(context.Context, string, string, int64, time.Duration) (bool, error)
	release func(context.Context, string, string, int64) (bool, error)
}

func (p testPersistence) TryAcquireLeaderLease(ctx context.Context, typ, replica string, ttl time.Duration) (int64, bool, error) {
	return p.acquire(ctx, typ, replica, ttl)
}

func (p testPersistence) RenewLeaderLease(ctx context.Context, typ, replica string, generation int64, ttl time.Duration) (bool, error) {
	return p.renew(ctx, typ, replica, generation, ttl)
}

func (p testPersistence) ReleaseLeaderLease(ctx context.Context, typ, replica string, generation int64) (bool, error) {
	return p.release(ctx, typ, replica, generation)
}

func successPersistence() testPersistence {
	return testPersistence{
		acquire: func(context.Context, string, string, time.Duration) (int64, bool, error) { return 7, true, nil },
		renew:   func(context.Context, string, string, int64, time.Duration) (bool, error) { return true, nil },
		release: func(context.Context, string, string, int64) (bool, error) { return true, nil },
	}
}

func TestManagerContentionDoesNotRunCallback(t *testing.T) {
	p := successPersistence()
	p.acquire = func(context.Context, string, string, time.Duration) (int64, bool, error) { return 0, false, nil }
	p.release = func(context.Context, string, string, int64) (bool, error) {
		t.Fatal("ReleaseLeaderLease called after contention")
		return false, nil
	}
	called := false
	ran, err := NewManager(p, "replica-1").TryRun(context.Background(), SchemaSync, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || ran || called {
		t.Fatalf("TryRun() = (%v, %v), called=%v", ran, err, called)
	}
}

func TestManagerAcquireErrorDoesNotRunCallback(t *testing.T) {
	want := errors.New("database unavailable")
	p := successPersistence()
	p.acquire = func(context.Context, string, string, time.Duration) (int64, bool, error) { return 0, false, want }
	ran, err := NewManager(p, "replica-1").TryRun(context.Background(), SchemaSync, func(context.Context) error {
		t.Fatal("callback called after acquire failure")
		return nil
	})
	if ran || !errors.Is(err, want) {
		t.Fatalf("TryRun() = (%v, %v), want false and %v", ran, err, want)
	}
}

func TestManagerReturnsCallbackErrorAndReleases(t *testing.T) {
	want := errors.New("callback failed")
	released := false
	p := successPersistence()
	p.release = func(_ context.Context, typ, replica string, generation int64) (bool, error) {
		released = true
		if typ != string(SchemaSync) || replica != "replica-1" || generation != 7 {
			t.Fatalf("release = (%q, %q, %d)", typ, replica, generation)
		}
		return true, nil
	}
	ran, err := NewManager(p, "replica-1").TryRun(context.Background(), SchemaSync, func(context.Context) error { return want })
	if !ran || !released || !errors.Is(err, want) {
		t.Fatalf("TryRun() = (%v, %v), released=%v", ran, err, released)
	}
}

func TestManagerRejectsUnsupportedTypeBeforePersistence(t *testing.T) {
	p := successPersistence()
	p.acquire = func(context.Context, string, string, time.Duration) (int64, bool, error) {
		t.Fatal("persistence called for unsupported type")
		return 0, false, nil
	}
	ran, err := NewManager(p, "replica-1").TryRun(context.Background(), Type("OTHER"), func(context.Context) error { return nil })
	if ran || err == nil {
		t.Fatalf("TryRun() = (%v, %v), want validation failure", ran, err)
	}
}

func TestManagerDelayedAcquireRenewsBeforeCallback(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	p := successPersistence()
	p.acquire = func(context.Context, string, string, time.Duration) (int64, bool, error) {
		clock.Advance(21 * time.Second)
		return 7, true, nil
	}
	called := false
	renewed := false
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) {
		renewed = true
		return true, nil
	}
	ran, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(context.Context) error {
		called = true
		return nil
	})
	if !ran || err != nil || !renewed || !called {
		t.Fatalf("TryRun() = (%v, %v), renewed=%v, called=%v", ran, err, renewed, called)
	}
}

func TestManagerDelayedAcquireRenewsUntilItHasACompleteWindow(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	p := successPersistence()
	p.acquire = func(context.Context, string, string, time.Duration) (int64, bool, error) {
		clock.Advance(21 * time.Second)
		return 7, true, nil
	}
	renewals := 0
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) {
		renewals++
		if renewals == 1 {
			// This successful operation consumed its whole local usable window.
			clock.Advance(20 * time.Second)
		}
		return true, nil
	}
	ran, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(context.Context) error {
		if renewals != 2 {
			t.Fatalf("callback started after %d renewals, want 2", renewals)
		}
		return nil
	})
	if !ran || err != nil || renewals != 2 {
		t.Fatalf("TryRun() = (%v, %v), renewals=%d", ran, err, renewals)
	}
}

func TestManagerRenewFalseCancelsCallbackAndReleasesAfterDrain(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	started := make(chan struct{})
	finished := make(chan struct{})
	released := make(chan struct{})
	p := successPersistence()
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) { return false, nil }
	p.release = func(context.Context, string, string, int64) (bool, error) {
		select {
		case <-finished:
		default:
			t.Error("released before callback drained")
		}
		close(released)
		return true, nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(finished)
			return ctx.Err()
		})
		result <- err
	}()
	<-started
	advanceAfterTimer(t, clock, 10*time.Second)
	if err := <-result; !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("TryRun error = %v, want ErrLeaseLost", err)
	}
	<-released
}

func TestManagerRenewFailureRetriesAndLogsRecoveryOnce(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	started := make(chan struct{})
	stop := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	p := successPersistence()
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls < 3 {
			return false, errors.New("temporary")
		}
		return true, nil
	}
	options := testOptions(clock)
	options.logger = logger
	result := make(chan error, 1)
	go func() {
		_, err := newManagerWithOptions(p, "replica-1", options).TryRun(context.Background(), SchemaSync, func(context.Context) error {
			close(started)
			<-stop
			return nil
		})
		result <- err
	}()
	<-started
	advanceAfterTimer(t, clock, 10*time.Second)
	advanceAfterTimer(t, clock, time.Second)
	advanceAfterTimer(t, clock, time.Second)
	clock.WaitForActiveTimer(t, 10*time.Second)
	close(stop)
	if err := <-result; err != nil {
		t.Fatalf("TryRun error = %v", err)
	}
	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 3 || bytes.Count(logs.Bytes(), []byte("renewal failed")) != 1 || bytes.Count(logs.Bytes(), []byte("renewal recovered")) != 1 {
		t.Fatalf("calls=%d logs=%q", gotCalls, logs.String())
	}
}

func TestManagerParentCancellationIsNotLeaseLoss(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	p := successPersistence()
	result := make(chan error, 1)
	go func() {
		_, err := NewManager(p, "replica-1").TryRun(ctx, SchemaSync, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return nil
		})
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) || errors.Is(err, ErrLeaseLost) {
		t.Fatalf("TryRun error = %v", err)
	}
}

func TestManagerParentCancellationUsesUncanceledCleanupContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	released := make(chan struct{})
	p := successPersistence()
	p.release = func(cleanupCtx context.Context, _ string, _ string, _ int64) (bool, error) {
		if err := cleanupCtx.Err(); err != nil {
			t.Errorf("cleanup context is canceled: %v", err)
		}
		close(released)
		return true, nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := NewManager(p, "replica-1").TryRun(ctx, SchemaSync, func(callbackCtx context.Context) error {
			close(started)
			<-callbackCtx.Done()
			return nil
		})
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) || errors.Is(err, ErrLeaseLost) {
		t.Fatalf("TryRun error = %v", err)
	}
	<-released
}

func TestManagerReleaseFailureJoinsCallbackAndLeaseErrors(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	callbackErr := errors.New("callback")
	releaseErr := errors.New("release")
	started := make(chan struct{})
	p := successPersistence()
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) { return false, nil }
	p.release = func(context.Context, string, string, int64) (bool, error) { return false, releaseErr }
	result := make(chan error, 1)
	go func() {
		_, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return callbackErr
		})
		result <- err
	}()
	<-started
	advanceAfterTimer(t, clock, 10*time.Second)
	if err := <-result; !errors.Is(err, callbackErr) || !errors.Is(err, ErrLeaseLost) || !errors.Is(err, releaseErr) {
		t.Fatalf("TryRun error = %v", err)
	}
}

func TestManagerDeadlineExhaustionJoinsFinalRenewalError(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	started := make(chan struct{})
	finalRenewalErr := errors.New("renewal timed out")
	p := successPersistence()
	p.renew = func(ctx context.Context, _ string, _ string, _ int64, _ time.Duration) (bool, error) {
		<-ctx.Done()
		return false, finalRenewalErr
	}
	result := make(chan error, 1)
	go func() {
		_, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return nil
		})
		result <- err
	}()
	<-started
	advanceAfterTimer(t, clock, 10*time.Second)
	advanceAfterTimer(t, clock, 10*time.Second)
	if err := <-result; !errors.Is(err, ErrLeaseLost) || !errors.Is(err, finalRenewalErr) {
		t.Fatalf("TryRun error = %v, want lease loss and %v", err, finalRenewalErr)
	}
}

func TestManagerDrainsInFlightRenewalBeforeRelease(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	renewStarted := make(chan struct{})
	allowRenewReturn := make(chan struct{})
	callbackStarted := make(chan struct{})
	released := make(chan struct{})
	p := successPersistence()
	p.renew = func(context.Context, string, string, int64, time.Duration) (bool, error) {
		close(renewStarted)
		<-allowRenewReturn
		return false, errors.New("late renewal")
	}
	p.release = func(context.Context, string, string, int64) (bool, error) {
		close(released)
		return true, nil
	}
	result := make(chan error, 1)
	go func() {
		_, err := newManagerWithOptions(p, "replica-1", testOptions(clock)).TryRun(context.Background(), SchemaSync, func(ctx context.Context) error {
			close(callbackStarted)
			<-ctx.Done()
			return nil
		})
		result <- err
	}()
	<-callbackStarted
	advanceAfterTimer(t, clock, 10*time.Second)
	<-renewStarted
	advanceAfterTimer(t, clock, 10*time.Second)
	select {
	case <-released:
		t.Fatal("release raced with in-flight renewal")
	default:
	}
	close(allowRenewReturn)
	if err := <-result; !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("TryRun error = %v, want ErrLeaseLost", err)
	}
	<-released
}

func testOptions(clock *testClock) managerOptions {
	return managerOptions{
		ttl:             30 * time.Second,
		renewalInterval: 10 * time.Second,
		retryInterval:   time.Second,
		safetyMargin:    10 * time.Second,
		cleanupTimeout:  time.Second,
		clock:           clock,
	}
}

func advanceAfterTimer(t *testing.T, clock *testClock, d time.Duration) {
	t.Helper()
	clock.WaitForActiveTimer(t, d)
	clock.Advance(d)
}

type testClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*testTimer
	added  chan struct{}
}

func newTestClock(now time.Time) *testClock {
	return &testClock{now: now, added: make(chan struct{}, 32)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) NewTimer(d time.Duration) timer {
	c.mu.Lock()
	t := &testTimer{due: c.now.Add(d), c: make(chan time.Time, 1)}
	c.timers = append(c.timers, t)
	c.mu.Unlock()
	c.added <- struct{}{}
	return t
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	var due []*testTimer
	for _, t := range c.timers {
		if !t.stopped && !now.Before(t.due) {
			t.stopped = true
			due = append(due, t)
		}
	}
	c.mu.Unlock()
	for _, t := range due {
		t.c <- now
	}
}

func (c *testClock) WaitForActiveTimer(t *testing.T, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		c.mu.Lock()
		want := c.now.Add(d)
		for _, timer := range c.timers {
			if !timer.stopped && timer.due.Equal(want) {
				c.mu.Unlock()
				return
			}
		}
		c.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("manager did not schedule active timer")
		}
		time.Sleep(time.Millisecond)
	}
}

type testTimer struct {
	mu      sync.Mutex
	due     time.Time
	c       chan time.Time
	stopped bool
}

func (t *testTimer) Chan() <-chan time.Time { return t.c }

func (t *testTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}
