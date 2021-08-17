package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
)

type TaskExecutor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	// Note, it's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error)
}

// defaultMigrationVersion returns the default migration version string
// Use the concatenation of current time and the task id to guarantee uniqueness in a monotonic increasing way.
func defaultMigrationVersionFromTaskId(taskId int) string {
	return strings.Join([]string{time.Now().Format("20060102150405"), strconv.Itoa(taskId)}, ".")
}
