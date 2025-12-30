package taskrun

import (
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/component/webhook"
)

// PipelineFailureWindow tracks failed tasks for aggregation.
type PipelineFailureWindow struct {
	mu               sync.Mutex
	firstFailureTime time.Time
	failedTasks      []webhook.FailedTask
	notificationSent bool
	timer            *time.Timer
}

// PipelineEventsTracker manages failure aggregation windows per plan.
type PipelineEventsTracker struct {
	mu      sync.RWMutex
	windows map[int64]*PipelineFailureWindow // planID -> window
}

// NewPipelineEventsTracker creates a new pipeline events tracker.
func NewPipelineEventsTracker() *PipelineEventsTracker {
	return &PipelineEventsTracker{
		windows: make(map[int64]*PipelineFailureWindow),
	}
}

// RecordTaskFailure adds a failed task to the aggregation window.
func (t *PipelineEventsTracker) RecordTaskFailure(planID int64, task webhook.FailedTask, onAggregated func([]webhook.FailedTask)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	window, exists := t.windows[planID]
	if !exists || window.notificationSent {
		// Start new window
		window = &PipelineFailureWindow{
			firstFailureTime: time.Now(),
			failedTasks:      []webhook.FailedTask{task},
			notificationSent: false,
		}
		t.windows[planID] = window

		// Set 5-minute timer
		window.timer = time.AfterFunc(5*time.Minute, func() {
			t.mu.Lock()
			defer t.mu.Unlock()

			if w, ok := t.windows[planID]; ok && !w.notificationSent {
				w.notificationSent = true
				onAggregated(w.failedTasks)
			}
		})
	} else {
		// Add to existing window
		window.mu.Lock()
		window.failedTasks = append(window.failedTasks, task)
		window.mu.Unlock()
	}
}

// Clear removes the window for a plan (call after pipeline completes).
func (t *PipelineEventsTracker) Clear(planID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if window, exists := t.windows[planID]; exists {
		if window.timer != nil {
			window.timer.Stop()
		}
		delete(t.windows, planID)
	}
}
