package scheduler

import "context"

type Executor interface {
	Run(ctx context.Context, taskRun TaskRun) error
}
