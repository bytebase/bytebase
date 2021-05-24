package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

type TaskExecutor interface {
	// Run will be called periodically by the scheduler until terminated is true.
	// Note, it's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	Run(ctx context.Context, taskRun api.TaskRun) (terminated bool, err error)
}
