// Package sampleprojectinstance periodically expires Cloud sample Project
// Instances. Every replica runs this worker; row locking makes passes safe.
package sampleprojectinstance

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const interval = time.Hour

// Runner performs startup and hourly sample Project Instance cleanup.
type Runner struct {
	manager cleanupManager
	clock   func() time.Time
	logger  *slog.Logger
}

type cleanupManager interface {
	Cleanup(context.Context, time.Time) error
}

// Options configures a cleanup runner.
type Options struct {
	Clock  func() time.Time
	Logger *slog.Logger
}

// NewRunner creates a cleanup runner.
func NewRunner(manager cleanupManager, options ...Options) *Runner {
	option := Options{}
	if len(options) > 0 {
		option = options[0]
	}
	if option.Clock == nil {
		option.Clock = time.Now
	}
	if option.Logger == nil {
		option.Logger = slog.Default()
	}
	return &Runner{
		manager: manager,
		clock:   option.Clock,
		logger:  option.Logger,
	}
}

// Run performs cleanup immediately, then once per hour until shutdown.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	r.runOnceAndLog(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnceAndLog(ctx)
		}
	}
}

// RunOnce performs one Sample Project Instance cleanup pass.
func (r *Runner) RunOnce(ctx context.Context) error {
	return r.manager.Cleanup(ctx, r.clock())
}

func (r *Runner) runOnceAndLog(ctx context.Context) {
	if err := r.RunOnce(ctx); err != nil {
		r.logger.ErrorContext(ctx, "Sample Project Instance cleanup failed", "error", err)
	}
}
