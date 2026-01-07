package heartbeat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	heartbeatInterval = 10 * time.Second
)

// Runner sends periodic heartbeats to indicate this replica is alive.
type Runner struct {
	store   *store.Store
	profile *config.Profile
}

// NewRunner creates a new heartbeat runner.
func NewRunner(store *store.Store, profile *config.Profile) *Runner {
	return &Runner{
		store:   store,
		profile: profile,
	}
}

// Run starts the heartbeat runner.
func (r *Runner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	slog.Debug("Heartbeat runner started", slog.String("replicaID", r.profile.DeployID))

	// Send heartbeat immediately on startup
	r.sendHeartbeat(ctx)

	for {
		select {
		case <-ticker.C:
			r.sendHeartbeat(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) sendHeartbeat(ctx context.Context) {
	if err := r.store.UpsertReplicaHeartbeat(ctx, r.profile.DeployID); err != nil {
		slog.Error("Failed to send heartbeat", log.BBError(err))
	}
}
