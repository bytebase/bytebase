// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

// InstanceMaximumConnectionNumber is the maximum number of connections outstanding per instance.
const InstanceMaximumConnectionNumber = 20

// State is the state for all in-memory states within the server.
type State struct {
	// InstanceDatabaseSyncChan is the channel for synchronizing schemas for instances.
	InstanceDatabaseSyncChan chan *api.Instance

	// RollbackGenerateMap is the set of tasks for generating rollback statements.
	RollbackGenerateMap sync.Map // map[task.ID]*store.TaskMessage
	// RollbacksCancel cancels the running rollback SQL generation for task taskID.
	RollbacksCancel sync.Map // map[taskID]context.CancelFunc

	// TaskProgress is the map from task ID to task progress.
	TaskProgress sync.Map // map[taskID]api.Progress
	// GhostTaskState is the map from task ID to gh-ost state.
	GhostTaskState sync.Map // map[taskID]sharedGhostState

	// RunningBackupDatabases is the set of databases running backups.
	RunningBackupDatabases sync.Map // map[databaseID]bool
	// RunningTaskChecks is the set of running task checks.
	RunningTaskChecks sync.Map // map[taskCheckID]bool
	// RunningTasks is the set of running tasks.
	RunningTasks sync.Map // map[taskID]bool
	// RunningTasksCancel is the cancel's of running tasks.
	RunningTasksCancel sync.Map // map[taskID]context.CancelFunc
	// InstanceOutstandingConnections is the maximum number of connections per instance.
	InstanceOutstandingConnections map[int]int

	sync.Mutex
}
