// Package sample periodically reconciles stale or expired sample instance
// setups. Every replica runs this worker; row locking makes passes safe.
package sample

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const interval = time.Hour

// Runner performs startup and hourly sample instance cleanup.
type Runner struct {
	manager cleanupManager
	logger  *slog.Logger
}

type cleanupManager interface {
	Cleanup(context.Context) error
}

// Options configures a cleanup runner.
type Options struct {
	Logger *slog.Logger
}

// NewRunner creates a cleanup runner.
func NewRunner(manager cleanupManager, options ...Options) *Runner {
	option := Options{}
	if len(options) > 0 {
		option = options[0]
	}
	if option.Logger == nil {
		option.Logger = slog.Default()
	}
	return &Runner{
		manager: manager,
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

// RunOnce performs one Sample instance cleanup pass.
func (r *Runner) RunOnce(ctx context.Context) error {
	return r.manager.Cleanup(ctx)
}

func (r *Runner) runOnceAndLog(ctx context.Context) {
	if err := r.RunOnce(ctx); err != nil {
		r.logger.ErrorContext(ctx, "Sample instance cleanup failed", "error", err)
	}
}
