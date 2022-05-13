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

// NewInstanceTask returns the executor for instance task.
func NewInstanceTask(l *zap.Logger) Executor {
	return &InstanceTask{
		l: l,
	}
}

// RunOnce will exec the segment task for instance
func (t *InstanceTask) Run(ctx context.Context, store *store.Store, segment *segment.Segment) error {
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
