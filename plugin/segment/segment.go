package segment

import (
	"context"
	"reflect"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"

	"go.uber.org/zap"
)

// EventType is the segment track event name.
type EventType string

var (
	// InstanceEventType is the track event for instance.
	InstanceEventType EventType = "instance"
)

// Segment is the metrics collector https://segment.com/.
type segment struct {
	l            *zap.Logger
	identify     string
	client       analytics.Client
	store        *store.Store
	reporterList []Reporter
}

// Workspace is the instance for console application.
type Workspace struct {
	License string
}

// Service is the service for Segment.
type Service interface {
	Close()
	Report(ctx context.Context)
	Track(event EventType, properties analytics.Properties)
	Identify(workspace *Workspace)
}

// NewService creates a new instance of Segment
func NewService(l *zap.Logger, key string, store *store.Store) (Service, error) {
	identify, err := initIdentify(store)
	if err != nil {
		return nil, err
	}

	client := analytics.New(key)

	return &segment{
		l:        l,
		identify: identify,
		client:   client,
		store:    store,
		reporterList: []Reporter{
			&InstanceReporter{},
		},
	}, nil
}

func initIdentify(store *store.Store) (string, error) {
	ctx := context.Background()
	configCreate := &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingSegmentIdentify,
		Value:       uuid.New().String(),
		Description: "The segment identify",
	}
	config, err := store.CreateSettingIfNotExist(ctx, configCreate)
	if err != nil {
		return "", err
	}

	return config.Value, nil
}

// Close will close the segment client.
func (s *segment) Close() {
	s.client.Close()
}

// Track will collect the metrics.
func (s *segment) Track(event EventType, properties analytics.Properties) {
	if err := s.client.Enqueue(analytics.Track{
		UserId:     s.identify,
		Event:      string(event),
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		s.l.Debug("segment track failed", zap.Error(err))
	}
}

// Identify will identify the workspace with license.
func (s *segment) Identify(workspace *Workspace) {
	if err := s.client.Enqueue(analytics.Identify{
		UserId:    s.identify,
		Traits:    analytics.NewTraits().Set("license", workspace.License),
		Timestamp: time.Now().UTC(),
	}); err != nil {
		s.l.Debug("segment identify failed", zap.Error(err))
	}
}

// Report will exec all the segment reporter.
func (s *segment) Report(ctx context.Context) {
	for _, reporter := range s.reporterList {
		reporterType := reflect.TypeOf(reporter).String()
		s.l.Info("Run segment reporter", zap.String("reporter", reporterType))
		if err := reporter.Report(ctx, s.store, s); err != nil {
			s.l.Info(
				"Failed to run segment reporter",
				zap.String("reporter", reporterType),
				zap.Error(err),
			)
		}
	}
}
