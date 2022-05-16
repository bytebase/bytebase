package segment

import (
	"context"
	"reflect"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"

	"go.uber.org/zap"
)

// EventType is the segment track event name.
type EventType string

var (
	// InstanceEventType is the track event for instance.
	InstanceEventType EventType = "bb_instance"
	// WorkspaceEventType is the track event for workspace.
	WorkspaceEventType EventType = "bb.workspace"
)

const (
	// IdentifyTraitForPlan is the trait key for subscription plan.
	IdentifyTraitForPlan = "plan"
)

// Segment is the metrics collector https://segment.com/.
type segment struct {
	l            *zap.Logger
	identifier   string
	client       analytics.Client
	store        *store.Store
	reporterList []Reporter
}

// NewService creates a new instance of Segment
func NewService(l *zap.Logger, key string, identifier string, store *store.Store) api.MetricService {
	client := analytics.New(key)

	return &segment{
		l:          l,
		identifier: identifier,
		client:     client,
		store:      store,
		reporterList: []Reporter{
			&InstanceReporter{},
		},
	}
}

// Close will close the segment client.
func (s *segment) Close() {
	s.client.Close()
}

// Identify will identify the workspace with license.
func (s *segment) Identify(workspace *api.Workspace) {
	if err := s.client.Enqueue(analytics.Identify{
		UserId:    s.identifier,
		Traits:    analytics.NewTraits().Set(IdentifyTraitForPlan, workspace.Plan),
		Timestamp: time.Now().UTC(),
	}); err != nil {
		s.l.Debug("segment identify failed", zap.Error(err))
	}
}

// Report will exec all the segment reporter.
func (s *segment) Report(ctx context.Context) {
	for _, reporter := range s.reporterList {
		reporterType := reflect.TypeOf(reporter).String()
		s.l.Debug("Run segment reporter", zap.String("reporter", reporterType))
		if err := reporter.Report(ctx, s.store, s); err != nil {
			s.l.Info(
				"Failed to report to segment",
				zap.String("reporter", reporterType),
				zap.Error(err),
			)
		}
	}
}
