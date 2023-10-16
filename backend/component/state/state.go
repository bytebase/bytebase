// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"
	"time"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

// InstanceMaximumConnectionNumber is the maximum number of connections outstanding per instance.
const InstanceMaximumConnectionNumber = 10

// State is the state for all in-memory states within the server.
type State struct {
	// InstanceDatabaseSyncChan is the channel for synchronizing schemas for instances.
	InstanceDatabaseSyncChan chan *store.InstanceMessage
	// InstanceSlowQuerySyncChan is the channel for synchronizing slow query logs for instances.
	InstanceSlowQuerySyncChan chan *InstanceSlowQuerySyncMessage

	// RollbackGenerate is the set of tasks for generating rollback statements.
	RollbackGenerate sync.Map // map[task.ID]*store.TaskMessage
	// RollbackCancel cancels the running rollback SQL generation for task taskID.
	RollbackCancel sync.Map // map[taskID]context.CancelFunc

	// ApprovalFinding is the set of issues for finding the approval template.
	ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

	// TaskProgress is the map from task ID to task progress.
	TaskProgress sync.Map // map[taskID]api.Progress
	// GhostTaskState is the map from task ID to gh-ost state.
	GhostTaskState sync.Map // map[taskID]sharedGhostState

	// TaskRunExecutionStatuses is the map from task run ID to task run execution status.
	TaskRunExecutionStatuses sync.Map // map[taskRunID]TaskRunExecutionStatus

	// RunningTaskRuns is the set of running taskruns.
	RunningTaskRuns sync.Map // map[taskRunID]bool
	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// RunningBackupDatabases is the set of databases running backups.
	RunningBackupDatabases sync.Map // map[databaseID]bool
	// RunningTaskChecks is the set of running task checks.
	RunningTaskChecks sync.Map // map[taskCheckID]bool
	// RunningTasks is the set of running tasks.
	RunningTasks sync.Map // map[taskID]bool
	// RunningPlanChecks is the set of running plan checks.
	RunningPlanChecks sync.Map
	// RunningTasksCancel is the cancel's of running tasks.
	RunningTasksCancel sync.Map // map[taskID]context.CancelFunc
	// InstanceOutstandingConnections is the maximum number of connections per instance.
	InstanceOutstandingConnections map[int]int

	// IssueExternalApprovalRelayCancelChan cancels the external approval from relay for issue issueUID.
	IssueExternalApprovalRelayCancelChan chan int

	// TaskSkippedOrDoneChan is the channel for notifying the task is skipped or done.
	TaskSkippedOrDoneChan chan int

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	ExpireCache *ristretto.Cache

	sync.Mutex
}

func New() (*State, error) {
	expireCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1_000,
		MaxCost:     1_000, // ~1KB
		BufferItems: 64,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create auth expire cache")
	}
	return &State{
		InstanceDatabaseSyncChan:             make(chan *store.InstanceMessage, 100),
		InstanceSlowQuerySyncChan:            make(chan *InstanceSlowQuerySyncMessage, 100),
		InstanceOutstandingConnections:       make(map[int]int),
		IssueExternalApprovalRelayCancelChan: make(chan int, 1),
		TaskSkippedOrDoneChan:                make(chan int, 1000),
		PlanCheckTickleChan:                  make(chan int, 1000),
		TaskRunTickleChan:                    make(chan int, 1000),
		ExpireCache:                          expireCache,
	}, nil
}

// InstanceSlowQuerySyncMessage is the message for synchronizing slow query logs for instances.
type InstanceSlowQuerySyncMessage struct {
	InstanceID string

	// ProjectID is used to filter the database list.
	// If ProjectID is empty, then all databases will be synced.
	// If ProjectID is not empty, then only databases belong to the project will be synced.
	ProjectID string
}

type TaskRunExecutionStatus struct {
	ExecutionStatus v1pb.TaskRun_ExecutionStatus
	UpdateTime      time.Time
}
