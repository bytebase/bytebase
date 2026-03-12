// Package bus contains the message bus for synchronization within the server.
package bus

import (
	"sync"
)

// Bus is the message bus for all in-memory communication within the server.
type Bus struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan string // issue ID

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[taskRunID]context.CancelFunc

	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[planCheckRunUID]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan string

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan string
}

func New() (*Bus, error) {
	return &Bus{
		ApprovalCheckChan:       make(chan string, 1000),
		PlanCheckTickleChan:     make(chan int, 1000),
		TaskRunTickleChan:       make(chan int, 1000),
		RolloutCreationChan:     make(chan string, 100),
		PlanCompletionCheckChan: make(chan string, 1000),
	}, nil
}
