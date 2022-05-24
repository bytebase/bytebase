package segment

import (
	"time"

	"github.com/bytebase/bytebase/plugin/metric"

	"github.com/segmentio/analytics-go"
	"go.uber.org/zap"
)

var _ metric.Reporter = (*reporter)(nil)

// reporter is the metrics collector https://segment.com/.
type reporter struct {
	l          *zap.Logger
	identifier string
	client     analytics.Client
}

const (
	// metricValueField is the property key for value
	metricValueField = "value"
)

// NewReporter creates a new instance of segment
func NewReporter(l *zap.Logger, key string, identifier string) metric.Reporter {
	client := analytics.New(key)

	return &reporter{
		l:          l,
		identifier: identifier,
		client:     client,
	}
}

// Close will close the segment client.
func (r *reporter) Close() {
	r.client.Close()
}

// Report will exec all the segment reporter.
func (r *reporter) Report(metric metric.Metric) error {
	properties := analytics.NewProperties().
		Set(metricValueField, metric.Value())

	for key, value := range metric.Labels() {
		properties.Set(key, value)
	}

	return r.client.Enqueue(analytics.Track{
		Event:      string(metric.Name()),
		UserId:     r.identifier,
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}

// Identify will identify the workspace with license.
func (r *reporter) Identify(identifier *metric.Identifier) error {
	traits := analytics.NewTraits()
	for key, value := range identifier.Labels {
		traits.Set(key, value)
	}

	return r.client.Enqueue(analytics.Identify{
		UserId:    r.identifier,
		Traits:    traits,
		Timestamp: time.Now().UTC(),
	})
}
