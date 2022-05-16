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
	metricSchedulerInterval = time.Duration(24) * time.Hour
)

// MetricScheduler is the metric scheduler.
type MetricScheduler struct {
	l *zap.Logger

	server  *Server
	metrics api.MetricService
}

// NewMetricScheduler creates a new metric scheduler.
func NewMetricScheduler(logger *zap.Logger, server *Server, deploymentID string) *MetricScheduler {
	segmentService := segment.NewService(logger, server.profile.SegmentKey, deploymentID, server.store)

	plan := api.FREE.String()
	if server.subscription != nil {
		plan = server.subscription.Plan.String()
	}

	segmentService.Identify(&api.Workspace{
		Plan:         plan,
		DeploymentID: deploymentID,
	})

	return &MetricScheduler{
		l:       logger,
		server:  server,
		metrics: segmentService,
	}
}

// Run will run the metric scheduler.
func (m *MetricScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(metricSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	m.l.Debug(fmt.Sprintf("Metrics scheduler started and will run every %v", metricSchedulerInterval))

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
						m.l.Error("Metrics scheduler PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()
				m.metrics.Report(ctx)
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Close will close the metric client.
func (m *MetricScheduler) Close() {
	m.metrics.Close()
}
