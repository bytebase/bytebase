// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/pkg/errors"
)

// State is the state for all in-memory states within the server.
type State struct {
	// ApprovalFinding is the set of issues for finding the approval template.
	ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// TaskRunConnectionID is the map from task run ID to the connection id of the connection to the database.
	TaskRunConnectionID sync.Map // map[taskRunID]string

	// RunningTaskRuns is the set of running taskruns.
	RunningTaskRuns sync.Map // map[taskRunID]bool
	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc
	// RunningDatabaseMigration is the taskUID of the running migration on the database.
	RunningDatabaseMigration sync.Map // map[databaseKey]taskUID

	// RunningPlanChecks is the set of running plan checks.
	RunningPlanChecks sync.Map
	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc
	// InstanceOutstandingConnections is the maximum number of connections per instance.
	InstanceOutstandingConnections *resourceLimiter
	// RolloutOutstandingTasks is the maximum number of tasks per rollout.
	RolloutOutstandingTasks *resourceLimiter

	// TaskSkippedOrDoneChan is the channel for notifying the task is skipped or done.
	TaskSkippedOrDoneChan chan int

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan int64
}

func New() (*State, error) {
	return &State{
		InstanceOutstandingConnections: &resourceLimiter{connections: map[string]int{}},
		RolloutOutstandingTasks:        &resourceLimiter{connections: map[string]int{}},
		TaskSkippedOrDoneChan:          make(chan int, 1000),
		PlanCheckTickleChan:            make(chan int, 1000),
		TaskRunTickleChan:              make(chan int, 1000),
		RolloutCreationChan:            make(chan int64, 100),
	}, nil
}

type resourceLimiter struct {
	sync.Mutex
	connections map[string]int
}

// limit <= 0 means no limit.
func (c *resourceLimiter) Increment(key string, limit int) bool {
	c.Lock()
	defer c.Unlock()
	if limit <= 0 {
		// No limit.
		// Increment anyway to balance the decrement.
		c.connections[key]++
		return false
	}
	if c.connections[key] >= limit {
		return true
	}
	c.connections[key]++
	return false
}

func (c *resourceLimiter) Decrement(key string) {
	c.Lock()
	defer c.Unlock()
	c.connections[key]--
}
