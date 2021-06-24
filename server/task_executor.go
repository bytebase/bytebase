package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

type TaskExecutor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	// Note, it's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, err error)
}
