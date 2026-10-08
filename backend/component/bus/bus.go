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
	ID        int64
}

// IssueRef identifies an issue by project and UID.
type IssueRef struct {
	ProjectID string
	UID       int64
}

// PlanCheckRunRef identifies a plan check run by project and UID.
type PlanCheckRunRef struct {
	ProjectID            string
	UID                  int64
	ApprovalInputVersion int64
}

// Bus is the message bus for all in-memory communication within the server.
type Bus struct {
	// ApprovalCheckChan signals when an issue needs approval template finding.
	// Triggered by plan check completion, issue creation (if checks already done).
	ApprovalCheckChan chan IssueRef

	// RunningTaskRunsCancelFunc is the cancelFunc of running taskruns.
	RunningTaskRunsCancelFunc sync.Map // map[TaskRunRef]context.CancelFunc

	// RunningPlanCheckRunsCancelFunc is the cancelFunc of running plan checks.
	RunningPlanCheckRunsCancelFunc sync.Map // map[PlanCheckRunRef]context.CancelFunc

	// PlanCheckTickleChan is the tickler for plan check scheduler.
	PlanCheckTickleChan chan int
	// TaskRunPendingTickleChan is the tickler for the pending task run
	// scheduler: a task run was created and needs scheduling.
	TaskRunPendingTickleChan chan int
	// TaskRunRunningTickleChan is the tickler for the running task run
	// scheduler: a task run became available and needs to be picked up.
	//
	// The two task run schedulers must not share one channel. A channel value is
	// delivered to exactly one receiver, so a shared channel hands each wake-up
	// to whichever scheduler happens to be ready for it rather than to the one
	// the sender meant, and the scheduler that actually had work waits out its
	// 5s ticker instead. That cost a measured ~5s per task run transition.
	TaskRunRunningTickleChan chan int
	// ReviewRunTickleChan is the tickler for the review run scheduler.
	ReviewRunTickleChan chan int

	// RolloutCreationChan is the channel for automatic rollout creation.
	RolloutCreationChan chan PlanRef

	// PlanCompletionCheckChan signals when a plan might be complete (for PIPELINE_COMPLETED webhook).
	PlanCompletionCheckChan chan PlanRef
}

func New() (*Bus, error) {
	return &Bus{
		ApprovalCheckChan:        make(chan IssueRef, 1000),
		PlanCheckTickleChan:      make(chan int, 1000),
		TaskRunPendingTickleChan: make(chan int, 1000),
		TaskRunRunningTickleChan: make(chan int, 1000),
		ReviewRunTickleChan:      make(chan int, 1000),
		RolloutCreationChan:      make(chan PlanRef, 100),
		PlanCompletionCheckChan:  make(chan PlanRef, 1000),
	}, nil
}

// Tickle wakes a scheduler without blocking. A tickle carries no information
// beyond "there may be work"; every scheduler sweeps the store for all of it,
// so a wake-up already queued makes another redundant. Not blocking here also
// keeps a request handler from parking on a scheduler that has stopped reading.
func Tickle(ch chan int) {
	select {
	case ch <- 0:
	default:
	}
}
