package taskrun

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// runPendingTaskRunsScheduler runs in a separate goroutine to schedule pending task runs.
func (s *Scheduler) runPendingTaskRunsScheduler(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Pending task runs scheduler started and will run every %v", taskSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				if s.haFailSince.IsZero() {
					s.haFailSince = time.Now()
				}
				if time.Since(s.haFailSince) >= haFailGracePeriod {
					s.failTaskRunsForHA(ctx, err)
				}
				continue
			}
			s.haFailSince = time.Time{}
			if err := s.schedulePendingTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule pending task runs", log.BBError(err))
			}
		case <-s.bus.TaskRunTickleChan:
			if err := s.licenseService.CheckReplicaLimit(ctx); err != nil {
				// Grace period is tracked by the ticker branch; just skip here.
				continue
			}
			s.haFailSince = time.Time{}
			if err := s.schedulePendingTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule pending task runs", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) schedulePendingTaskRuns(ctx context.Context) (err error) {
	startedAt := time.Now()
	tx, err := s.store.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin pending scheduler transaction")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			slog.Error("Failed to rollback pending scheduler transaction", log.BBError(rollbackErr))
		}
	}()

	// Acquire cluster-wide mutex - only one replica runs at a time.
	acquired, err := store.TryAdvisoryXactLock(ctx, tx, store.AdvisoryLockKeyPendingScheduler)
	if err != nil {
		return errors.Wrapf(err, "failed to acquire pending scheduler advisory lock")
	}
	if !acquired {
		slog.Debug("Pending scheduler advisory lock held by another replica, skipping",
			slog.Duration("duration", time.Since(startedAt)),
		)
		return nil
	}

	taskRuns, err := s.store.ListTaskRunsByStatus(ctx, []storepb.TaskRun_Status{storepb.TaskRun_PENDING})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending task runs")
	}

	// Build context once per cycle.
	sc, err := newSchedulingContext(ctx, s.store)
	if err != nil {
		return errors.Wrapf(err, "failed to create scheduling context")
	}

	for _, taskRun := range taskRuns {
		if err := s.schedulePendingTaskRun(ctx, taskRun, sc); err != nil {
			slog.Error("failed to schedule pending task run",
				slog.Int64("taskRunID", taskRun.ID),
				log.BBError(err),
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit pending scheduler transaction")
	}

	return nil
}

func (s *Scheduler) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, sc *schedulingContext) error {
	task, err := s.store.GetTaskByID(ctx, taskRun.ProjectID, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}

	// Check 1: RunAt time
	if taskRun.RunAt != nil && time.Now().Before(*taskRun.RunAt) {
		return nil
	}

	// Check 3: Database mutual exclusion (for sequential tasks)
	canProceed, _ := sc.checkDatabaseMutualExclusion(task)
	if !canProceed {
		return nil
	}

	// Check 4: Parallel task limit per rollout
	maxParallel, err := s.getMaxParallelForTask(ctx, task)
	if err != nil {
		return errors.Wrapf(err, "failed to get max parallel limit")
	}
	if !sc.checkParallelLimit(task, maxParallel) {
		s.storeParallelLimitCause(ctx, taskRun.ProjectID, taskRun.ID)
		return nil
	}

	// All checks passed - promote to AVAILABLE
	if err := s.promoteTaskRun(ctx, taskRun); err != nil {
		return err
	}
	sc.markPromoted(task)

	return nil
}

func (s *Scheduler) storeParallelLimitCause(ctx context.Context, projectID string, taskRunID int64) {
	payload := &storepb.TaskRunPayload{
		SchedulerInfo: &storepb.SchedulerInfo{
			ReportTime: timestamppb.Now(),
			WaitingCause: &storepb.SchedulerInfo_WaitingCause{
				Cause: &storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit{
					ParallelTasksLimit: true,
				},
			},
		},
	}
	if err := s.store.UpdateTaskRunPayload(ctx, projectID, taskRunID, payload); err != nil {
		slog.Error("failed to store parallel limit cause", log.BBError(err))
	}
}

func (s *Scheduler) getMaxParallelForTask(ctx context.Context, task *store.TaskMessage) (int, error) {
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{ProjectID: task.ProjectID, UID: &task.PlanID})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return 0, errors.Errorf("plan %v not found", task.PlanID)
	}

	project, err := s.store.GetProjectByResourceID(ctx, plan.ProjectID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return 0, errors.Errorf("project %v not found", plan.ProjectID)
	}

	return int(project.Setting.GetParallelTasksPerRollout()), nil
}

func (s *Scheduler) promoteTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// Clear scheduler info by writing empty payload
	if err := s.store.UpdateTaskRunPayload(ctx, taskRun.ProjectID, taskRun.ID, &storepb.TaskRunPayload{}); err != nil {
		slog.Error("failed to clear scheduler info", log.BBError(err))
	}

	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:        taskRun.ID,
		ProjectID: taskRun.ProjectID,
		Updater:   "",
		Status:    storepb.TaskRun_AVAILABLE,
		AllowedStatuses: []storepb.TaskRun_Status{
			storepb.TaskRun_PENDING,
		},
	}); err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil
		}
		return errors.Wrapf(err, "failed to update task run status to available")
	}

	select {
	case s.bus.TaskRunTickleChan <- 0:
	default:
	}

	return nil
}

// planKey identifies a plan by project and ID.
type planKey struct {
	projectID string
	planID    int64
}

// schedulingContext holds pre-fetched and indexed active task run data for a scheduling cycle.
type schedulingContext struct {
	// Pre-indexed active task runs (AVAILABLE + RUNNING)
	activeByDatabase  map[string]int64 // dbKey -> blocking taskID (first found)
	activeCountByPlan map[planKey]int  // (project, planID) -> active count

	// Tracks promotions this round
	promotedDBs    map[string]bool
	promotedCounts map[planKey]int
}

func newSchedulingContext(ctx context.Context, s *store.Store) (*schedulingContext, error) {
	activeTaskRuns, err := s.ListTaskRunsByStatus(ctx, []storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list active task runs")
	}

	sc := &schedulingContext{
		activeByDatabase:  make(map[string]int64),
		activeCountByPlan: make(map[planKey]int),
		promotedDBs:       make(map[string]bool),
		promotedCounts:    make(map[planKey]int),
	}

	if len(activeTaskRuns) == 0 {
		return sc, nil
	}

	// Group task run task UIDs by project for correct project-scoped lookups.
	taskIDsByProject := make(map[string][]int64)
	for _, tr := range activeTaskRuns {
		taskIDsByProject[tr.ProjectID] = append(taskIDsByProject[tr.ProjectID], tr.TaskUID)
	}

	// Batch fetch tasks per project.
	type taskKey struct {
		projectID string
		id        int64
	}
	taskByKey := make(map[taskKey]*store.TaskMessage)
	for projectID, ids := range taskIDsByProject {
		tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, IDs: &ids})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list tasks for project %s", projectID)
		}
		for _, t := range tasks {
			taskByKey[taskKey{projectID: t.ProjectID, id: t.ID}] = t
		}
	}

	// Build indexes
	for _, tr := range activeTaskRuns {
		task := taskByKey[taskKey{projectID: tr.ProjectID, id: tr.TaskUID}]
		if task == nil {
			continue
		}
		sc.activeCountByPlan[planKey{projectID: task.ProjectID, planID: task.PlanID}]++

		if task.DatabaseName != nil && isSequentialTask(task) {
			dbKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)
			if _, exists := sc.activeByDatabase[dbKey]; !exists {
				sc.activeByDatabase[dbKey] = task.ID
			}
		}
	}

	return sc, nil
}

func (sc *schedulingContext) checkDatabaseMutualExclusion(task *store.TaskMessage) (canProceed bool, blockingTaskID *int64) {
	if task.DatabaseName == nil || !isSequentialTask(task) {
		return true, nil
	}

	dbKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)

	// Check in-memory tracking (promoted this round)
	if sc.promotedDBs[dbKey] {
		return false, nil
	}

	// Check pre-fetched active tasks
	if blockingID, exists := sc.activeByDatabase[dbKey]; exists {
		return false, &blockingID
	}

	return true, nil
}

func (sc *schedulingContext) checkParallelLimit(task *store.TaskMessage, maxParallel int) bool {
	if maxParallel <= 0 {
		return true // no limit
	}
	pk := planKey{projectID: task.ProjectID, planID: task.PlanID}
	currentCount := sc.activeCountByPlan[pk] + sc.promotedCounts[pk]
	return currentCount < maxParallel
}

func (sc *schedulingContext) markPromoted(task *store.TaskMessage) {
	if task.DatabaseName != nil && isSequentialTask(task) {
		sc.promotedDBs[getDatabaseKey(task.InstanceID, *task.DatabaseName)] = true
	}
	sc.promotedCounts[planKey{projectID: task.ProjectID, planID: task.PlanID}]++
}
