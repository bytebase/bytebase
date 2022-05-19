package reporter

import (
	"time"

	"github.com/bytebase/bytebase/api"

	"github.com/segmentio/analytics-go"
	"go.uber.org/zap"
)

// Segment is the metrics collector https://segment.com/.
type segment struct {
	l          *zap.Logger
	identifier string
	client     analytics.Client
}

const (
	// identifyTraitForPlan is the trait key for subscription plan.
	identifyTraitForPlan = "plan"
	// metricValueField is the property key for value
	metricValueField = "value"
)

// NewSegmentReporter creates a new instance of segment
func NewSegmentReporter(l *zap.Logger, key string, identifier string) api.MetricReporter {
	client := analytics.New(key)

	return &segment{
		l:          l,
		identifier: identifier,
		client:     client,
	}
}

// Close will close the segment client.
func (s *segment) Close() {
	s.client.Close()
}

// Report will exec all the segment reporter.
func (s *segment) Report(metric *api.Metric) error {
	properties := analytics.NewProperties().
		Set(metricValueField, metric.Value)

	for key, value := range metric.Labels {
		properties.Set(key, value)
	}

	return s.client.Enqueue(analytics.Track{
		Event:      string(metric.Name),
		UserId:     s.identifier,
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}

// Identify will identify the workspace with license.
func (s *segment) Identify(workspace *api.Workspace) error {
	return s.client.Enqueue(analytics.Identify{
		UserId:    s.identifier,
		Traits:    analytics.NewTraits().Set(identifyTraitForPlan, workspace.Plan),
		Timestamp: time.Now().UTC(),
	})
}
