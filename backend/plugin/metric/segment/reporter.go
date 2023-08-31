// Package segment implements the reporter for segment.
package segment

import (
	"time"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/metric"

	"github.com/segmentio/analytics-go"
)

var _ metric.Reporter = (*reporter)(nil)

// reporter is the metrics collector https://segment.com/.
type reporter struct {
	client analytics.Client
}

const (
	// metricValueField is the property key for value.
	metricValueField = "value"
)

// NewReporter creates a new instance of segment.
func NewReporter(key string) metric.Reporter {
	client, err := analytics.NewWithConfig(key, analytics.Config{
		Logger: &sinkLogger{},
	})
	if err != nil {
		log.Error("failed to create reporter", zap.Error(err))
		client = analytics.New(key)
	}

	return &reporter{
		client: client,
	}
}

// Close will close the segment client.
func (r *reporter) Close() {
	r.client.Close()
}

// Report will exec all the segment reporter.
func (r *reporter) Report(id string, metric *metric.Metric) error {
	properties := analytics.NewProperties().
		Set(metricValueField, metric.Value)

	for key, value := range metric.Labels {
		properties.Set(key, value)
	}

	return r.client.Enqueue(analytics.Track{
		Event:      string(metric.Name),
		UserId:     id,
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}

// Identify will identify the workspace with license.
func (r *reporter) Identify(identifier *metric.Identifier) error {
	traits := analytics.NewTraits()
	traits.SetEmail(identifier.Email)
	traits.SetName(identifier.Name)
	for key, value := range identifier.Labels {
		traits.Set(key, value)
	}

	return r.client.Enqueue(analytics.Identify{
		UserId:    identifier.ID,
		Traits:    traits,
		Timestamp: time.Now().UTC(),
	})
}

type sinkLogger struct {
}

func (sinkLogger) Logf(string, ...any) {
}

func (sinkLogger) Errorf(string, ...any) {
}
