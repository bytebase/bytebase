package segment

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"
)

// InstanceReporter is the segment reporter for instance.
type InstanceReporter struct {
}

// Report will exec the segment reporter for instance
func (t *InstanceReporter) Report(ctx context.Context, store *store.Store, segment *segment) error {
	status := api.Normal
	count, err := store.CountInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})
	if err != nil {
		return err
	}

	return segment.client.Enqueue(analytics.Track{
		UserId:     segment.identifier,
		Event:      string(InstanceEventType),
		Properties: analytics.NewProperties().Set("count", count),
		Timestamp:  time.Now().UTC(),
	})
}
