package reporter

import (
	"time"

	"github.com/bytebase/bytebase/plugin/metric"

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
	// metricValueField is the property key for value
	metricValueField = "value"
)

// NewSegmentReporter creates a new instance of segment
func NewSegmentReporter(l *zap.Logger, key string, identifier string) MetricReporter {
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
func (s *segment) Report(metric *metric.Metric) error {
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
func (s *segment) Identify(identifier *metric.Identifier) error {
	traits := analytics.NewTraits()
	for key, value := range identifier.Labels {
		traits.Set(key, value)
	}

	return s.client.Enqueue(analytics.Identify{
		UserId:    s.identifier,
		Traits:    traits,
		Timestamp: time.Now().UTC(),
	})
}
