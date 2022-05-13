package task

import (
	"context"

	"github.com/bytebase/bytebase/plugin/segment"
	"github.com/bytebase/bytebase/store"
)

// Executor is the API message for segment task
type Executor interface {
	Run(ctx context.Context, store *store.Store, segment *segment.Segment) error
}
