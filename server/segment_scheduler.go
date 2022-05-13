package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/segment"

	"go.uber.org/zap"
)

const (
	segmentSchedulerInterval = time.Duration(1) * time.Second
)

// NewSegmentScheduler creates a new segment scheduler.
func NewSegmentScheduler(logger *zap.Logger, server *Server) (*SegmentScheduler, error) {
	segmentService, err := segment.NewService(logger, server.profile.SegmentKey, server.store)
	if err != nil {
		return nil, err
	}

	license := api.FREE.String()
	if server.subscription != nil {
		license = server.subscription.Plan.String()
	}
	segmentService.Identify(&segment.Workspace{
		License: license,
	})

	return &SegmentScheduler{
		l:       logger,
		server:  server,
		segment: segmentService,
	}, nil
}

// SegmentScheduler is the segment scheduler.
type SegmentScheduler struct {
	l *zap.Logger

	server  *Server
	segment segment.Service
}

// Run will run the task scheduler.
func (s *SegmentScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(segmentSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug(fmt.Sprintf("Task scheduler started and will run every %v", segmentSchedulerInterval))

	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Task scheduler PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
					s.segment.Close()
				}()

				ctx := context.Background()
				s.segment.Report(ctx)
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}
