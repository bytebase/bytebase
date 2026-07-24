// Package leader coordinates singleton component work across replicas.
package leader

import (
	"context"
	"errors"
	"log/slog"
	"time"

	pkgerrors "github.com/pkg/errors"
)

const (
	leaseTTL          = 30 * time.Second
	renewalInterval   = 10 * time.Second
	renewalRetry      = time.Second
	leaseSafetyMargin = 10 * time.Second
	cleanupTimeout    = 5 * time.Second
)

// Type identifies the work guarded by a leader lease.
type Type string

const (
	// SchemaSync guards schema synchronization work.
	SchemaSync Type = "SCHEMA_SYNC"
)

// ErrLeaseLost reports that this replica can no longer safely act as leader.
var ErrLeaseLost = errors.New("leader lease lost")

var errReleaseNotOwned = errors.New("leader lease release was not accepted")

// Persistence stores leader leases.
type Persistence interface {
	TryAcquireLeaderLease(ctx context.Context, leaderType string, replicaID string, ttl time.Duration) (generation int64, acquired bool, err error)
	RenewLeaderLease(ctx context.Context, leaderType string, replicaID string, generation int64, ttl time.Duration) (renewed bool, err error)
	ReleaseLeaderLease(ctx context.Context, leaderType string, replicaID string, generation int64) (released bool, err error)
}

// Manager runs component work only while this replica holds its lease.
type Manager struct {
	persistence Persistence
	replicaID   string
	options     managerOptions
}

type managerOptions struct {
	ttl             time.Duration
	renewalInterval time.Duration
	retryInterval   time.Duration
	safetyMargin    time.Duration
	cleanupTimeout  time.Duration
	clock           clock
	logger          *slog.Logger
}

type clock interface {
	Now() time.Time
	NewTimer(time.Duration) timer
}

type timer interface {
	Chan() <-chan time.Time
	Stop() bool
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) NewTimer(d time.Duration) timer {
	return systemTimer{Timer: time.NewTimer(d)}
}

type systemTimer struct {
	*time.Timer
}

func (t systemTimer) Chan() <-chan time.Time {
	return t.C
}

// NewManager creates a leader manager for a replica.
func NewManager(p Persistence, replicaID string) *Manager {
	return newManagerWithOptions(p, replicaID, managerOptions{})
}

// newManagerWithOptions allows deterministic timing and log tests without exposing timing
// as user configuration.
func newManagerWithOptions(p Persistence, replicaID string, options managerOptions) *Manager {
	if options.ttl == 0 {
		options.ttl = leaseTTL
	}
	if options.renewalInterval == 0 {
		options.renewalInterval = renewalInterval
	}
	if options.retryInterval == 0 {
		options.retryInterval = renewalRetry
	}
	if options.safetyMargin == 0 {
		options.safetyMargin = leaseSafetyMargin
	}
	if options.cleanupTimeout == 0 {
		options.cleanupTimeout = cleanupTimeout
	}
	if options.clock == nil {
		options.clock = systemClock{}
	}
	if options.logger == nil {
		options.logger = slog.Default()
	}
	return &Manager{persistence: p, replicaID: replicaID, options: options}
}

// TryRun acquires the lease for typ and runs callback while it remains valid.
func (m *Manager) TryRun(ctx context.Context, typ Type, callback func(context.Context) error) (bool, error) {
	if typ != SchemaSync {
		return false, pkgerrors.Errorf("unsupported leader type %q", typ)
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}

	acquireStarted := m.options.clock.Now()
	generation, acquired, err := m.persistence.TryAcquireLeaderLease(ctx, string(typ), m.replicaID, m.options.ttl)
	if err != nil {
		return false, pkgerrors.Wrap(err, "acquire leader lease")
	}
	if !acquired {
		return false, nil
	}

	validUntil := acquireStarted.Add(m.usableLease())
	authorityStarted, validUntil, preCallbackErr := m.establishInitialAuthority(ctx, typ, generation, acquireStarted, validUntil)
	if preCallbackErr != nil {
		return false, errors.Join(preCallbackErr, m.release(ctx, typ, generation))
	}
	if err := ctx.Err(); err != nil {
		return false, errors.Join(err, m.release(ctx, typ, generation))
	}

	leaderCtx, cancelLeader := context.WithCancel(ctx)
	defer cancelLeader()
	renewStop, stopRenew := context.WithCancel(context.Background())
	renewDone := make(chan renewalOutcome, 1)
	go func() {
		renewDone <- m.renewUntilStopped(ctx, renewStop.Done(), cancelLeader, typ, generation, authorityStarted, validUntil)
	}()

	callbackErr := callback(leaderCtx)
	stopRenew()
	renewal := <-renewDone
	releaseErr := m.release(ctx, typ, generation)

	var parentErr error
	if ctx.Err() != nil && !renewal.lost {
		parentErr = ctx.Err()
	}
	return true, errors.Join(callbackErr, renewal.err, parentErr, releaseErr)
}

func (m *Manager) usableLease() time.Duration {
	return m.options.ttl - m.options.safetyMargin
}

func (m *Manager) establishInitialAuthority(ctx context.Context, typ Type, generation int64, authorityStarted, validUntil time.Time) (time.Time, time.Time, error) {
	for !m.options.clock.Now().Before(validUntil) {
		if err := ctx.Err(); err != nil {
			return authorityStarted, validUntil, err
		}
		started := m.options.clock.Now()
		renewed, err := m.persistence.RenewLeaderLease(ctx, string(typ), m.replicaID, generation, m.options.ttl)
		if err != nil {
			return authorityStarted, validUntil, pkgerrors.Wrap(err, "renew leader lease before callback")
		}
		if !renewed {
			return authorityStarted, validUntil, ErrLeaseLost
		}
		authorityStarted = started
		validUntil = started.Add(m.usableLease())
	}
	return authorityStarted, validUntil, nil
}

type renewalOutcome struct {
	lost bool
	err  error
}

func (m *Manager) renewUntilStopped(parent context.Context, stop <-chan struct{}, cancelLeader context.CancelFunc, typ Type, generation int64, authorityStarted, validUntil time.Time) renewalOutcome {
	nextRenewal := authorityStarted.Add(m.options.renewalInterval)
	loggedFailure := false
	for {
		now := m.options.clock.Now()
		if !now.Before(validUntil) {
			cancelLeader()
			return renewalOutcome{lost: true, err: ErrLeaseLost}
		}
		wakeAt := nextRenewal
		if validUntil.Before(wakeAt) {
			wakeAt = validUntil
		}
		wait := m.options.clock.NewTimer(wakeAt.Sub(now))
		select {
		case <-stop:
			wait.Stop()
			return renewalOutcome{}
		case <-parent.Done():
			wait.Stop()
			return renewalOutcome{}
		case <-wait.Chan():
		}

		if parent.Err() != nil {
			return renewalOutcome{}
		}
		if !m.options.clock.Now().Before(validUntil) {
			cancelLeader()
			return renewalOutcome{lost: true, err: ErrLeaseLost}
		}
		started := m.options.clock.Now()
		renewCtx, cancelRenew := context.WithCancel(parent)
		result := make(chan renewResult, 1)
		go func() {
			renewed, err := m.persistence.RenewLeaderLease(renewCtx, string(typ), m.replicaID, generation, m.options.ttl)
			result <- renewResult{renewed: renewed, err: err}
		}()

		deadline := m.options.clock.NewTimer(validUntil.Sub(m.options.clock.Now()))
		var renewal renewResult
		select {
		case <-stop:
			deadline.Stop()
			cancelRenew()
			<-result
			return renewalOutcome{}
		case <-parent.Done():
			deadline.Stop()
			cancelRenew()
			<-result
			return renewalOutcome{}
		case renewal = <-result:
			deadline.Stop()
			cancelRenew()
		case <-deadline.Chan():
			cancelRenew()
			renewal = <-result
			if parent.Err() != nil {
				return renewalOutcome{}
			}
			cancelLeader()
			return renewalOutcome{lost: true, err: errors.Join(ErrLeaseLost, renewal.err)}
		}

		if parent.Err() != nil {
			return renewalOutcome{}
		}
		if !m.options.clock.Now().Before(validUntil) {
			cancelLeader()
			return renewalOutcome{lost: true, err: errors.Join(ErrLeaseLost, renewal.err)}
		}
		if renewal.err != nil {
			if !loggedFailure {
				m.options.logger.Warn("leader lease renewal failed; retrying", "leader_type", typ, "error", renewal.err)
				loggedFailure = true
			}
			nextRenewal = m.options.clock.Now().Add(m.options.retryInterval)
			continue
		}
		if !renewal.renewed {
			cancelLeader()
			return renewalOutcome{lost: true, err: ErrLeaseLost}
		}
		if loggedFailure {
			m.options.logger.Info("leader lease renewal recovered", "leader_type", typ)
			loggedFailure = false
		}
		validUntil = started.Add(m.usableLease())
		nextRenewal = started.Add(m.options.renewalInterval)
	}
}

type renewResult struct {
	renewed bool
	err     error
}

func (m *Manager) release(ctx context.Context, typ Type, generation int64) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.options.cleanupTimeout)
	defer cancel()
	released, err := m.persistence.ReleaseLeaderLease(cleanupCtx, string(typ), m.replicaID, generation)
	if err != nil {
		if !released {
			return errors.Join(errReleaseNotOwned, pkgerrors.Wrap(err, "release leader lease"))
		}
		return pkgerrors.Wrap(err, "release leader lease")
	}
	if !released {
		return errReleaseNotOwned
	}
	return nil
}
