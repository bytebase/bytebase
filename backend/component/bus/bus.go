// Package bus contains the message bus for synchronization within the server.
package bus

import (
	"sync"
)

// PlanRef identifies a plan by project and UID.
type PlanRef struct {
	ProjectID string
	PlanID    int64
}

// TaskRunRef identifies a task run by project and ID.
type TaskRunRef struct {
	ProjectID string
	ID        int
}

// PlanCheckRunRef identifies a plan check run by project and UID.
type PlanCheckRunRef struct {
	ProjectID string
	UID       int
}

// Bus is the message bus for all in-memory communication within the server.
type Bus struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan int64 // issue UID

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[TaskRunRef]context.CancelFunc

	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[PlanCheckRunRef]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunTickleChan is the tickler for task run scheduler.
	TaskRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan PlanRef

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan PlanRef
}

func New() (*Bus, error) {
	return &Bus{
		ApprovalCheckChan:       make(chan int64, 1000),
		PlanCheckTickleChan:     make(chan int, 1000),
		TaskRunTickleChan:       make(chan int, 1000),
		RolloutCreationChan:     make(chan PlanRef, 100),
		PlanCompletionCheckChan: make(chan PlanRef, 1000),
	}, nil
}
