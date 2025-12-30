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

// defaultRolloutMaxRunningTaskRuns is the maximum number of running tasks per rollout.
// No limit by default.
const defaultRolloutMaxRunningTaskRuns = 0

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

	go s.ListenTaskSkippedOrDone(ctx)

	// Start rollout creator component
	rolloutCreator := NewRolloutCreator(s.store, s.stateCfg, s.webhookManager)
	wg.Add(3)
	go rolloutCreator.Run(ctx, wg, s.stateCfg.RolloutCreationChan)
	go s.runPendingTaskRunsScheduler(ctx, wg)
	go s.runRunningTaskRunsScheduler(ctx, wg)

	slog.Debug("Task scheduler V2 started with independent runners")
	<-ctx.Done()
}

func (s *Scheduler) ListenTaskSkippedOrDone(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("ListenTaskSkippedOrDone PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()
	slog.Info("TaskSkippedOrDoneListener started")
	type planEnvironment struct {
		planUID     int64
		environment string
	}
	planEnvironmentDoneConfirmed := map[planEnvironment]bool{}

	for {
		select {
		case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
			if err := func() error {
				task, err := s.store.GetTaskByID(ctx, taskUID)
				if err != nil {
					return errors.Wrapf(err, "failed to get task")
				}
				if planEnvironmentDoneConfirmed[planEnvironment{planUID: task.PlanID, environment: task.Environment}] {
					return nil
				}

				environmentTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &task.PlanID, Environment: &task.Environment})
				if err != nil {
					return errors.Wrapf(err, "failed to list tasks")
				}

				skippedOrDone := tasksSkippedOrDone(environmentTasks)
				if !skippedOrDone {
					return nil
				}

				planEnvironmentDoneConfirmed[planEnvironment{planUID: task.PlanID, environment: task.Environment}] = true

				// Get live environment order
				// Some environments may have zero tasks. We need to filter them out.
				environments, err := s.store.GetEnvironment(ctx)
				if err != nil {
					return errors.Wrapf(err, "failed to get environments")
				}
				var environmentOrder []string
				for _, env := range environments.GetEnvironments() {
					environmentOrder = append(environmentOrder, env.Id)
				}

				allTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PlanID: &task.PlanID})
				if err != nil {
					return errors.Wrapf(err, "failed to list tasks")
				}
				allTaskEnvironments := map[string]bool{}
				for _, task := range allTasks {
					allTaskEnvironments[task.Environment] = true
				}

				// Filter out environments that have zero tasks
				filteredEnvironmentOrder := []string{}
				for _, env := range environmentOrder {
					if allTaskEnvironments[env] {
						filteredEnvironmentOrder = append(filteredEnvironmentOrder, env)
					}
				}

				currentEnvironment := task.Environment
				var nextEnvironment string
				for i, env := range filteredEnvironmentOrder {
					if env == currentEnvironment {
						if i < len(filteredEnvironmentOrder)-1 {
							nextEnvironment = filteredEnvironmentOrder[i+1]
						}
						break
					}
				}

				plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
				if err != nil {
					return errors.Wrapf(err, "failed to get plan")
				}
				if plan == nil {
					return errors.Errorf("plan %v not found", task.PlanID)
				}
				project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
				if err != nil {
					return errors.Wrapf(err, "failed to get project")
				}
				if project == nil {
					return errors.Errorf("project %v not found", plan.ProjectID)
				}
				issueN, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &task.PlanID})
				if err != nil {
					return errors.Wrapf(err, "failed to get issue")
				}

				// every task in the stage terminated
				// create "stage ends" activity.
				s.webhookManager.CreateEvent(ctx, &webhook.Event{
					Actor:   store.SystemBotUser,
					Type:    storepb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE,
					Comment: "",
					Issue:   webhook.NewIssue(issueN),
					Rollout: webhook.NewRollout(plan),
					Project: webhook.NewProject(project),
					StageStatusUpdate: &webhook.EventStageStatusUpdate{
						StageTitle: currentEnvironment,
						StageID:    currentEnvironment,
					},
				})

				// create "notify pipeline rollout" activity.
				if err := func() error {
					if nextEnvironment == "" {
						return nil
					}
					policy, err := s.store.GetRolloutPolicy(ctx, nextEnvironment)
					if err != nil {
						return errors.Wrapf(err, "failed to get rollout policy")
					}
					s.webhookManager.CreateEvent(ctx, &webhook.Event{
						Actor:   store.SystemBotUser,
						Type:    storepb.Activity_NOTIFY_PIPELINE_ROLLOUT,
						Comment: "",
						Issue:   webhook.NewIssue(issueN),
						Project: webhook.NewProject(project),
						IssueRolloutReady: &webhook.EventIssueRolloutReady{
							RolloutPolicy: policy,
							StageName:     nextEnvironment,
						},
					})
					return nil
				}(); err != nil {
					slog.Error("failed to create rollout release notification activity", log.BBError(err))
				}

				// Check if entire plan is complete
				s.checkPlanCompletion(ctx, task.PlanID)

				return nil
			}(); err != nil {
				slog.Error("failed to handle task skipped or done", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func tasksSkippedOrDone(tasks []*store.TaskMessage) bool {
	for _, task := range tasks {
		skipped := task.Payload.GetSkipped()
		done := task.LatestTaskRunStatus == storepb.TaskRun_DONE
		if !skipped && !done {
			return false
		}
	}
	return true
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

	// Send PIPELINE_COMPLETED webhook
	startedAt := time.Now()
	completedAt := time.Now()
	if earliestStart != nil {
		startedAt = *earliestStart
	}
	if latestEnd != nil {
		completedAt = *latestEnd
	}

	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   store.SystemBotUser,
		Type:    storepb.Activity_PIPELINE_COMPLETED,
		Comment: "",
		Issue:   webhook.NewIssue(issueN),
		Rollout: webhook.NewRollout(plan),
		Project: webhook.NewProject(project),
		PipelineCompleted: &webhook.EventPipelineCompleted{
			TotalTasks:  len(tasks),
			StartedAt:   startedAt,
			CompletedAt: completedAt,
		},
	})
}
