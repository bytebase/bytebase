package taskrun

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
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
	stateCfg       *state.State
	webhookManager *webhook.Manager
	executorMap    map[storepb.Task_Type]Executor
	profile        *config.Profile
	pipelineEvents *PipelineEventsTracker
}

// NewScheduler will create a new scheduler.
func NewScheduler(
	store *store.Store,
	stateCfg *state.State,
	webhookManager *webhook.Manager,
	profile *config.Profile,
) *Scheduler {
	return &Scheduler{
		store:          store,
		stateCfg:       stateCfg,
		webhookManager: webhookManager,
		profile:        profile,
		executorMap:    map[storepb.Task_Type]Executor{},
		pipelineEvents: NewPipelineEventsTracker(),
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
	rolloutCreator := NewRolloutCreator(s.store, s.stateCfg, s.webhookManager)
	wg.Add(3)
	go rolloutCreator.Run(ctx, wg, s.stateCfg.RolloutCreationChan)
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
	slog.Info("Task completion listener started")

	for {
		select {
		case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
			if err := func() error {
				task, err := s.store.GetTaskByID(ctx, taskUID)
				if err != nil {
					return errors.Wrapf(err, "failed to get task")
				}

				// Check if entire plan is complete and handle webhooks
				s.checkPlanCompletion(ctx, task.PlanID)

				return nil
			}(); err != nil {
				slog.Error("failed to handle task completion", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) checkPlanCompletion(ctx context.Context, planID int64) {
	// Get all tasks for this plan
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		slog.Error("failed to list tasks for plan completion check", log.BBError(err))
		return
	}

	// Check if all tasks are in terminal state
	allComplete := true
	hasFailures := false
	var earliestStart, latestEnd *time.Time

	for _, task := range tasks {
		status := task.LatestTaskRunStatus

		// Check if task is in terminal state
		isTerminal := status == storepb.TaskRun_DONE ||
			status == storepb.TaskRun_FAILED ||
			status == storepb.TaskRun_CANCELED ||
			status == storepb.TaskRun_SKIPPED ||
			task.Payload.GetSkipped()

		if !isTerminal {
			allComplete = false
			break
		}

		// Track failures (excluding skipped tasks)
		if status == storepb.TaskRun_FAILED && !task.Payload.GetSkipped() {
			hasFailures = true
		}

		// Track start/end times for metrics
		if task.RunAt != nil {
			if earliestStart == nil || task.RunAt.Before(*earliestStart) {
				earliestStart = task.RunAt
			}
		}
		if task.UpdatedAt != nil {
			if latestEnd == nil || task.UpdatedAt.After(*latestEnd) {
				latestEnd = task.UpdatedAt
			}
		}
	}

	// Not all tasks complete yet
	if !allComplete {
		return
	}

	// Always clear the failure window when plan completes
	s.pipelineEvents.Clear(planID)

	// Only send completion webhook if there were no failures
	if hasFailures {
		return
	}

	// Get plan, project and issue for webhook
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &planID})
	if err != nil {
		slog.Error("failed to get plan for completion webhook", log.BBError(err))
		return
	}
	if plan == nil {
		slog.Error("plan not found for completion webhook", slog.Int64("plan_id", planID))
		return
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		slog.Error("failed to get project for completion webhook", log.BBError(err))
		return
	}
	if project == nil {
		slog.Error("project not found for completion webhook", slog.String("project_id", plan.ProjectID))
		return
	}

	issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planID})
	if err != nil {
		slog.Error("failed to get issue for completion webhook", log.BBError(err))
		return
	}
	if issueN == nil {
		slog.Error("issue not found for completion webhook", slog.Int64("plan_id", planID))
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

// getFailedTaskRuns returns all failed task runs for a plan to include in webhook payload.
func (s *Scheduler) getFailedTaskRuns(ctx context.Context, planID int64) []webhook.FailedTask {
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &planID})
	if err != nil {
		slog.Error("failed to list tasks for failed webhook", log.BBError(err))
		return nil
	}

	var failures []webhook.FailedTask
	for _, task := range tasks {
		if task.LatestTaskRunStatus != storepb.TaskRun_FAILED {
			continue
		}

		// Get the latest failed task run
		taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{TaskUID: &task.ID})
		if err != nil {
			slog.Error("failed to list task runs", log.BBError(err))
			continue
		}

		// Find the latest failed run
		var latestFailed *store.TaskRunMessage
		for _, tr := range taskRuns {
			if tr.Status == storepb.TaskRun_FAILED {
				if latestFailed == nil || tr.UpdatedAt.After(latestFailed.UpdatedAt) {
					latestFailed = tr
				}
			}
		}

		if latestFailed == nil {
			continue
		}

		errorMsg := ""
		if latestFailed.ResultProto != nil && latestFailed.ResultProto.Detail != "" {
			errorMsg = latestFailed.ResultProto.Detail
		}

		// Construct task name from type and database
		taskName := task.Type.String()
		if task.DatabaseName != nil && *task.DatabaseName != "" {
			taskName = taskName + " - " + *task.DatabaseName
		}

		dbName := ""
		if task.DatabaseName != nil {
			dbName = *task.DatabaseName
		}

		failures = append(failures, webhook.FailedTask{
			TaskID:       int64(task.ID),
			TaskName:     taskName,
			DatabaseName: dbName,
			InstanceName: task.InstanceID,
			ErrorMessage: errorMsg,
			FailedAt:     latestFailed.UpdatedAt,
		})
	}

	return failures
}
