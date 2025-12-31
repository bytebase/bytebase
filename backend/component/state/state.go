// Package state contains the synchronization state shared within the server.
package state

import (
	"sync"
)

// State is the state for all in-memory states within the server.
type State struct {
	// ApprovalFinding is the set of issues for finding the approval template.
	ApprovalFinding sync.Map // map[issue.ID]*store.IssueMessage

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// TaskRunConnectionID is the map from task run ID to the connection id of the connection to the database.
	TaskRunConnectionID sync.Map // map[taskRunID]string

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// RunningPlanChecks is the set of running plan checks.
	RunningPlanChecks sync.Map
	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc

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
		TaskSkippedOrDoneChan: make(chan int, 1000),
		PlanCheckTickleChan:   make(chan int, 1000),
		TaskRunTickleChan:     make(chan int, 1000),
		RolloutCreationChan:   make(chan int64, 100),
	}, nil
}
