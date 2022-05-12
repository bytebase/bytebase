package segment

import (
	"fmt"
	"io/ioutil"

	"github.com/bytebase/bytebase/common"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"
	"go.uber.org/zap"
)

// Segment is the metrics collector https://segment.com/.
type Segment struct {
	logger   *zap.Logger
	identify string
	mode     common.ReleaseMode
	client   analytics.Client
}

// Service is the service for Segment.
type Service interface {
	Close()
	Track()
}

// NewService creates a new instance of Segment
func NewService(logger *zap.Logger, dataDir string, mode common.ReleaseMode) (*Segment, error) {
	identify, err := getIdentify(dataDir)
	if err != nil {
		return nil, err
	}

	client := analytics.New("SEGMENT_WRITE_KEY")

	return &Segment{
		logger:   logger,
		identify: identify,
		mode:     mode,
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
func (s *Segment) Track() {

}
