package segment

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"
	"go.uber.org/zap"
)

// InstanceReporter is the segment reporter for instance.
type InstanceReporter struct {
	l *zap.Logger
}

// Report will exec the segment reporter for instance
func (r *InstanceReporter) Report(ctx context.Context, store *store.Store, segment *segment) error {
	status := api.Normal
	instanceList, err := store.FindInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})
	if err != nil {
		return err
	}

	instanceEngineMap := make(map[db.Type]int)
	for _, instance := range instanceList {
		instanceEngineMap[instance.Engine] = instanceEngineMap[instance.Engine] + 1
	}

	for engine, count := range instanceEngineMap {
		properties := analytics.NewProperties().
			Set("database", string(engine)).
			Set("count", count)

		err := segment.client.Enqueue(analytics.Track{
			Event:      string(InstanceEventName),
			UserId:     segment.identifier,
			Properties: properties,
			Timestamp:  time.Now().UTC(),
		})
		if err != nil {
			r.l.Debug("failed to enqueue report event for instance", zap.String("database", string(engine)))
		}
	}

	return nil
}
