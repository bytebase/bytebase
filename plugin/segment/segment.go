package segment

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/bytebase/bytebase/plugin/segment/api"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"

	"go.uber.org/zap"
)

// Segment is the metrics collector https://segment.com/.
type Segment struct {
	l        *zap.Logger
	identify string
	client   analytics.Client
}

// Service is the service for Segment.
type Service interface {
	Close()
	Track()
}

// NewService creates a new instance of Segment
func NewService(l *zap.Logger, dataDir string, key string) (*Segment, error) {
	identify, err := getIdentify(dataDir)
	if err != nil {
		return nil, err
	}

	client := analytics.New(key)

	return &Segment{
		l:        l,
		identify: identify,
		client:   client,
	}, nil
}

func getIdentify(dataDir string) (string, error) {
	filename := fmt.Sprintf("%s/segment-identify", dataDir)
	identify, err := ioutil.ReadFile(filename)

	if err == nil {
		return string(identify), nil
	}

	id := uuid.New().String()
	err = ioutil.WriteFile(filename, []byte(id), 0644)
	return id, err
}

// Close will close the segment client.
func (s *Segment) Close() {
	s.client.Close()
}

// Track will collect the metrics.
func (s *Segment) Track(event api.Event) {
	if err := s.client.Enqueue(analytics.Track{
		UserId:     s.identify,
		Event:      string(event.GetType()),
		Properties: event.GetProperties(),
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		s.l.Debug("segment track failed", zap.Error(err))
	}
}

// Identify will identify the user with specific fields.
func (s *Segment) Identify(identify api.Identify) {
	if err := s.client.Enqueue(analytics.Identify{
		UserId:    s.identify,
		Traits:    identify.GetTraits(),
		Timestamp: time.Now().UTC(),
	}); err != nil {
		s.l.Debug("segment identify failed", zap.Error(err))
	}
}
