package segment

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"
)

// InstanceReporter is the segment reporter for instance.
type InstanceReporter struct {
}

// Report will exec the segment reporter for instance
func (t *InstanceReporter) Report(ctx context.Context, store *store.Store, segment *segment) error {
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

	properties := analytics.NewProperties().Set("count", len(instanceList))
	for engine, count := range instanceEngineMap {
		properties.Set(string(engine), count)
	}

	return segment.client.Enqueue(analytics.Page{
		Name:       string(InstanceEventName),
		UserId:     segment.identifier,
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}
