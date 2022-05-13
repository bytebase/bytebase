package server

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/segment"
	segmentAPI "github.com/bytebase/bytebase/plugin/segment/api"
	"github.com/bytebase/bytebase/plugin/segment/task"
	"go.uber.org/zap"
)

const (
	segmentSchedulerInterval = time.Duration(24) * time.Hour
)

// NewSegmentScheduler creates a new segment scheduler.
func NewSegmentScheduler(logger *zap.Logger, server *Server) (*SegmentScheduler, error) {
	segmentService, err := segment.NewService(logger, server.profile.DataDir, server.profile.SegmentKey)
	if err != nil {
		return nil, err
	}

	license := api.FREE.String()
	if server.subscription != nil {
		license = server.subscription.Plan.String()
	}
	segmentService.Identify(&segmentAPI.WorkspaceIdentify{
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

	server    *Server
	segment   *segment.Segment
	executors []task.Executor
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
				}()

				ctx := context.Background()

				for _, e := range s.executors {
					go func(executor task.Executor) {
						s.l.Info("Run segment task", zap.String("task", reflect.TypeOf(executor).String()))
						if err := executor.Run(ctx, s.server.store, s.segment); err != nil {
							s.l.Info(
								"Failed to run segment task",
								zap.String("task", reflect.TypeOf(executor).String()),
								zap.Error(err),
							)
						}
					}(e)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Register will register a task executor.
func (s *SegmentScheduler) Register(executor task.Executor) {
	if executor == nil {
		panic("segment scheduler: Register executor is nil")
	}
	s.executors = append(s.executors, executor)
}
