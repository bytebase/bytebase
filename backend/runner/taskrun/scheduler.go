package taskrun

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	taskSchedulerInterval = 5 * time.Second
)

// Scheduler is the scheduler for task run.
type Scheduler struct {
	store          *store.Store
	bus            *bus.Bus
	webhookManager *webhook.Manager
	executorMap    map[storepb.Task_Type]Executor
	profile        *config.Profile
}

// NewScheduler will create a new scheduler.
func NewScheduler(
	store *store.Store,
	bus *bus.Bus,
	webhookManager *webhook.Manager,
	profile *config.Profile,
) *Scheduler {
	return &Scheduler{
		store:          store,
		bus:            bus,
		webhookManager: webhookManager,
		profile:        profile,
		executorMap:    map[storepb.Task_Type]Executor{},
	}
}

// Register will register a task executor factory.
func (s *Scheduler) Register(taskType storepb.Task_Type, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType.String())
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType.String())
	}
	s.executorMap[taskType] = executorGetter
}

// Run will start the scheduler.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	go s.runTaskCompletionListener(ctx)

	// Start rollout creator component
	rolloutCreator := NewRolloutCreator(s.store, s.bus, s.webhookManager)
	wg.Add(3)
	go rolloutCreator.Run(ctx, wg, s.bus.RolloutCreationChan)
	go s.runPendingTaskRunsScheduler(ctx, wg)
	go s.runRunningTaskRunsScheduler(ctx, wg)

	slog.Debug("Task scheduler V2 started with independent runners")
	<-ctx.Done()
}

func (s *Scheduler) runTaskCompletionListener(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("runTaskCompletionListener PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()
	slog.Info("Plan completion checker started")

	for {
		select {
		case <-ctx.Done():
			return
		case planID := <-s.bus.PlanCompletionCheckChan:
			s.checkPlanCompletion(ctx, planID)
		}
	}
}

// checkPlanCompletion checks if all tasks in a plan are complete and successful.
// If so, sends PIPELINE_COMPLETED webhook.
// Called when tasks are marked DONE/SKIPPED, or when tasks are skipped/canceled via API.
func (s *Scheduler) checkPlanCompletion(ctx context.Context, planID int64) {
	// Get all tasks for this plan
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		slog.Error("failed to list tasks for plan completion check", log.BBError(err))
		return
	}

	// Check if all tasks are complete (DONE or SKIPPED)
	for _, task := range tasks {
		status := task.LatestTaskRunStatus

		// Only DONE and SKIPPED are considered complete
		// FAILED and CANCELED are not complete states
		isComplete := status == storepb.TaskRun_DONE ||
			status == storepb.TaskRun_SKIPPED ||
			task.Payload.GetSkipped()

		if !isComplete {
			// Not all tasks complete - no webhook
			return
		}
	}

	// All tasks complete and successful - try to claim completion notification
	claimed, err := s.store.ClaimPipelineCompletionNotification(ctx, planID)
	if err != nil {
		slog.Error("failed to claim pipeline completion notification", log.BBError(err))
		return
	}
	if !claimed {
		return // Already sent
	}

	// Get plan and project for webhook
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil || plan == nil {
		slog.Error("failed to get plan for completion webhook", log.BBError(err))
		return
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil || project == nil {
		slog.Error("failed to get project for completion webhook", log.BBError(err))
		return
	}

	// Send PIPELINE_COMPLETED webhook
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Type:    storepb.Activity_PIPELINE_COMPLETED,
		Project: webhook.NewProject(project),
		RolloutCompleted: &webhook.EventRolloutCompleted{
			Rollout: webhook.NewRollout(plan),
		},
	})
}
