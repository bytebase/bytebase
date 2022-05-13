package task

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/segment"
	segmentAPI "github.com/bytebase/bytebase/plugin/segment/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// InstanceTask is the segment task for instance.
type InstanceTask struct {
	l *zap.Logger
}

// RunOnce will exec the segment task for instance
func (t *InstanceTask) RunOnce(ctx context.Context, store *store.Store, segment *segment.Segment) error {
	status := api.Normal
	count, err := store.CountInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})
	if err != nil {
		return err
	}
	segment.Track(&segmentAPI.InstanceEvent{
		Count: count,
	})
	return nil
}
