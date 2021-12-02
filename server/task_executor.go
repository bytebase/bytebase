package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
)

// TaskExecutor is the task executor.
type TaskExecutor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	//
	// NOTE
	//
	// 1. It's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	// 2. If err is non-nil, then the detail field will be ignored since info is provided in the err.
	RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error)
}

// defaultMigrationVersion returns the default migration version string
// Use the concatenation of current time and the task id to guarantee uniqueness in a monotonic increasing way.
func defaultMigrationVersionFromTaskID(taskID int) string {
	return strings.Join([]string{time.Now().Format("20060102150405"), strconv.Itoa(taskID)}, ".")
}
