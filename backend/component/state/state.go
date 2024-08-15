// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/pkg/errors"
)

// defaultInstanceMaximumConnections is the maximum number of connections outstanding per instance by default.
const defaultInstanceMaximumConnections = 10

// State is the state for all in-memory states within the server.
type State struct {
	// InstanceSlowQuerySyncChan is the channel for synchronizing slow query logs for instances.
	InstanceSlowQuerySyncChan chan *InstanceSlowQuerySyncMessage

	// ApprovalFinding is the set of issues for finding the approval template.
	ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

	// TaskProgress is the map from task ID to task progress.
	TaskProgress sync.Map // map[taskID]api.Progress
	// GhostTaskState is the map from task ID to gh-ost state.
	GhostTaskState sync.Map // map[taskID]sharedGhostState

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// TaskRunConnectionID is the map from task run ID to the connection id of the connection to the database.
	TaskRunConnectionID sync.Map // map[taskRunID]string

	// RunningTaskRuns is the set of running taskruns.
	RunningTaskRuns sync.Map // map[taskRunID]bool
	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc
	// RunningDatabaseMigration is the taskUID of the running migration on the database.
	RunningDatabaseMigration sync.Map // map[databaseID]taskUID

	// RunningPlanChecks is the set of running plan checks.
	RunningPlanChecks sync.Map
	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc
	// InstanceOutstandingConnections is the maximum number of connections per instance.
	InstanceOutstandingConnections *connectionLimiter

	// IssueExternalApprovalRelayCancelChan cancels the external approval from relay for issue issueUID.
	IssueExternalApprovalRelayCancelChan chan int

	// TaskSkippedOrDoneChan is the channel for notifying the task is skipped or done.
	TaskSkippedOrDoneChan chan int

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	ExpireCache *lru.Cache[string, bool]
}

func New() (*State, error) {
	expireCache, err := lru.New[string, bool](128)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create auth expire cache")
	}
	return &State{
		InstanceSlowQuerySyncChan:            make(chan *InstanceSlowQuerySyncMessage, 100),
		InstanceOutstandingConnections:       &connectionLimiter{connections: map[int]int{}},
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

type connectionLimiter struct {
	sync.Mutex
	connections map[int]int
}

func (c *connectionLimiter) Increment(instanceID, maxConnections int) bool {
	c.Lock()
	defer c.Unlock()
	if maxConnections == 0 {
		maxConnections = defaultInstanceMaximumConnections
	}

	if c.connections[instanceID] >= maxConnections {
		return true
	}
	c.connections[instanceID]++
	return false
}

func (c *connectionLimiter) Decrement(instanceID int) {
	c.Lock()
	defer c.Unlock()
	c.connections[instanceID]--
}
