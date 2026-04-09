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
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	taskSchedulerInterval = 5 * time.Second
	// haFailGracePeriod is the duration to wait before failing task runs after
	// the HA license check starts failing. This avoids permanently failing task
	// runs during rolling restarts where stale heartbeats cause transient
	// over-counts.
	haFailGracePeriod = 10 * time.Minute
)

// Scheduler is the scheduler for task run.
type Scheduler struct {
	store          *store.Store
	bus            *bus.Bus
	webhookManager *webhook.Manager
	licenseService *enterprise.LicenseService
	executorMap    map[storepb.Task_Type]Executor
	profile        *config.Profile
	// haFailSince is when CheckReplicaLimit first started failing.
	// Zero means the check is currently passing.
	haFailSince time.Time
}

// NewScheduler will create a new scheduler.
func NewScheduler(
	store *store.Store,
	bus *bus.Bus,
	webhookManager *webhook.Manager,
	licenseService *enterprise.LicenseService,
	profile *config.Profile,
) *Scheduler {
	return &Scheduler{
		store:          store,
		bus:            bus,
		webhookManager: webhookManager,
		licenseService: licenseService,
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

// failTaskRunsForHA fails all PENDING and AVAILABLE task runs because HA is not licensed.
func (s *Scheduler) failTaskRunsForHA(ctx context.Context, haErr error) {
	taskRuns, err := s.store.ListTaskRunsByStatus(ctx, []storepb.TaskRun_Status{storepb.TaskRun_PENDING, storepb.TaskRun_AVAILABLE})
	if err != nil {
		slog.Error("failed to list task runs for HA limit check", log.BBError(err))
		return
	}
	if len(taskRuns) == 0 {
		return
	}

	// Track affected plans to send failure webhooks.
	// Key by (projectID, planID) to match ClaimPipelineFailureNotification's dedup key.
	type planKey struct {
		projectID string
		planID    int64
	}
	affectedPlans := map[planKey]string{} // value = first environment seen

	for _, taskRun := range taskRuns {
		if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			ProjectID: taskRun.ProjectID,
			Status:    storepb.TaskRun_FAILED,
			ResultProto: &storepb.TaskRunResult{
				Detail: haErr.Error(),
			},
		}); err != nil {
			slog.Error("failed to fail task run for HA limit",
				slog.Int64("taskRunID", taskRun.ID),
				log.BBError(err),
			)
			continue
		}
		key := planKey{projectID: taskRun.ProjectID, planID: taskRun.PlanUID}
		if _, ok := affectedPlans[key]; !ok {
			affectedPlans[key] = taskRun.Environment
		}
	}

	slog.Warn("Failed task runs due to HA license restriction", slog.Int64("count", int64(len(taskRuns))), log.BBError(haErr))

	// Send PIPELINE_FAILED webhook for each affected plan.
	for key, environment := range affectedPlans {
		claimed, err := s.store.ClaimPipelineFailureNotification(ctx, key.projectID, key.planID)
		if err != nil {
			slog.Error("failed to claim pipeline failure notification", log.BBError(err))
			continue
		}
		if !claimed {
			continue
		}
		plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: key.projectID, UID: &key.planID})
		if err != nil || plan == nil {
			slog.Error("failed to get plan for HA failure webhook", log.BBError(err))
			continue
		}
		project, err := s.store.GetProjectByResourceID(ctx, plan.ProjectID)
		if err != nil || project == nil {
			slog.Error("failed to get project for HA failure webhook", log.BBError(err))
			continue
		}
		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Type:    storepb.Activity_PIPELINE_FAILED,
			Project: webhook.NewProject(project),
			RolloutFailed: &webhook.EventRolloutFailed{
				Rollout:     webhook.NewRollout(plan),
				Environment: environment,
			},
		})
	}
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
		case ref := <-s.bus.PlanCompletionCheckChan:
			s.checkPlanCompletion(ctx, ref)
		}
	}
}

// checkPlanCompletion checks if all tasks in a plan are complete and successful.
// If so, sends PIPELINE_COMPLETED webhook and auto-resolves issues for deferred rollout plans.
// Deferred rollout plans (exportDataConfig, createDatabaseConfig) auto-resolve when tasks complete.
// Called when tasks are marked DONE/SKIPPED, or when tasks are skipped/canceled via API.
func (s *Scheduler) checkPlanCompletion(ctx context.Context, ref bus.PlanRef) {
	planID := ref.PlanID
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: ref.ProjectID, UID: &planID})
	if err != nil || plan == nil {
		slog.Error("failed to get plan for completion check", log.BBError(err))
		return
	}

	// Get all tasks for this plan
	tasks, err := s.store.ListTasks(ctx, &store.TaskFind{ProjectID: ref.ProjectID, PlanID: &planID})
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
	claimed, err := s.store.ClaimPipelineCompletionNotification(ctx, ref.ProjectID, planID)
	if err != nil {
		slog.Error("failed to claim pipeline completion notification", log.BBError(err))
		return
	}
	if !claimed {
		return // Already sent
	}

	project, err := s.store.GetProjectByResourceID(ctx, plan.ProjectID)
	if err != nil || project == nil {
		slog.Error("failed to get project for completion webhook", log.BBError(err))
		return
	}

	// Use environment from the first task (all tasks should be in the same environment for a rollout)
	environment := ""
	if len(tasks) > 0 {
		environment = tasks[0].Environment
	}

	// Send PIPELINE_COMPLETED webhook
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Type:    storepb.Activity_PIPELINE_COMPLETED,
		Project: webhook.NewProject(project),
		RolloutCompleted: &webhook.EventRolloutCompleted{
			Rollout:     webhook.NewRollout(plan),
			Environment: environment,
		},
	})

	// Auto-resolve issue for deferred rollout plans.
	// Deferred rollout plans are those with only exportDataConfig or createDatabaseConfig specs.
	// These are simple single-phase operations that don't require manual resolution.
	if isDeferredRolloutPlan(plan) {
		s.autoResolveIssue(ctx, ref.ProjectID, planID)
	}
}

// isDeferredRolloutPlan returns true if the plan contains only deferred rollout specs
// (exportDataConfig or createDatabaseConfig).
func isDeferredRolloutPlan(plan *store.PlanMessage) bool {
	specs := plan.Config.GetSpecs()
	if len(specs) == 0 {
		return false
	}
	for _, spec := range specs {
		if spec.GetExportDataConfig() == nil && spec.GetCreateDatabaseConfig() == nil {
			return false
		}
	}
	return true
}

// autoResolveIssue automatically resolves the issue associated with a plan by setting its status to DONE.
func (s *Scheduler) autoResolveIssue(ctx context.Context, projectID string, planID int64) {
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectID}, PlanUID: &planID})
	if err != nil {
		slog.Error("failed to get issue for auto-resolve", log.BBError(err))
		return
	}
	if issue == nil {
		return
	}
	if issue.Status != storepb.Issue_OPEN {
		return
	}

	if _, err := s.store.UpdateIssue(ctx, issue.ProjectID, issue.UID, &store.UpdateIssueMessage{Status: new(storepb.Issue_DONE)}); err != nil {
		slog.Error("failed to auto-resolve issue", slog.String("project", projectID), slog.Int64("issueUID", issue.UID), log.BBError(err))
		return
	}
	slog.Info("auto-resolved deferred rollout issue", slog.String("project", projectID), slog.Int64("issueUID", issue.UID), slog.Int64("planID", planID))
}
