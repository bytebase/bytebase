// Package slowquerysync is a runner that synchronize slow query logs.
package slowquerysync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	schemaSyncInterval = 12 * time.Hour
)

// NewSyncer creates a new slow query syncer.
func NewSyncer(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile config.Profile) *Syncer {
	return &Syncer{
		store:     store,
		dbFactory: dbFactory,
		stateCfg:  stateCfg,
		profile:   profile,
	}
}

// Syncer is the slow query syncer.
type Syncer struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	stateCfg  *state.State
	profile   config.Profile
}

// Run will run the slow query syncer.
func (s *Syncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Slow query syncer started and will run every %s", schemaSyncInterval.String()))
	for {
		select {
		case <-ctx.Done():
			log.Debug("Slow query syncer received context cancellation")
			return
		case <-ticker.C:
			log.Debug("Slow query syncer received tick")
			s.syncSlowQuery(ctx)
		}
	}
}

func (s *Syncer) syncSlowQuery(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("slow query syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()
}
