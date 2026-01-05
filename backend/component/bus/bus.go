// Package bus contains the message bus for synchronization within the server.
package bus

import (
	"sync"
)

// Bus is the message bus for all in-memory communication within the server.
type Bus struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan int64 // issue UID

	TaskRunSchedulerInfo sync.Map // map[taskRunID]*storepb.SchedulerInfo

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan int64

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan int64
}

func New() (*Bus, error) {
	return &Bus{
		ApprovalCheckChan:       make(chan int64, 1000),
		PlanCheckTickleChan:     make(chan int, 1000),
		TaskRunTickleChan:       make(chan int, 1000),
		RolloutCreationChan:     make(chan int64, 100),
		PlanCompletionCheckChan: make(chan int64, 1000),
	}, nil
}
