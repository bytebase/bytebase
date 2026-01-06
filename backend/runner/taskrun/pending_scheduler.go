package taskrun

import (
	"context"
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
			if err := s.schedulePendingTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule pending task runs", log.BBError(err))
			}
		case <-s.bus.TaskRunTickleChan:
			if err := s.schedulePendingTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule pending task runs", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) schedulePendingTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_PENDING},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending task runs")
	}

	// Build context once per cycle
	sc, err := newSchedulingContext(ctx, s.store)
	if err != nil {
		return errors.Wrapf(err, "failed to create scheduling context")
	}

	for _, taskRun := range taskRuns {
		if err := s.schedulePendingTaskRun(ctx, taskRun, sc); err != nil {
			slog.Error("failed to schedule pending task run", log.BBError(err))
		}
	}

	return nil
}

func (s *Scheduler) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, sc *schedulingContext) error {
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
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
	if !sc.checkParallelLimit(task.PlanID, maxParallel) {
		s.storeParallelLimitCause(taskRun.ID)
		return nil
	}

	// All checks passed - promote to AVAILABLE
	if err := s.promoteTaskRun(ctx, taskRun); err != nil {
		return err
	}
	sc.markPromoted(task)

	return nil
}

func (s *Scheduler) storeParallelLimitCause(taskRunID int) {
	s.bus.TaskRunSchedulerInfo.Store(taskRunID, &storepb.SchedulerInfo{
		ReportTime: timestamppb.Now(),
		WaitingCause: &storepb.SchedulerInfo_WaitingCause{
			Cause: &storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit{
				ParallelTasksLimit: true,
			},
		},
	})
}

func (s *Scheduler) getMaxParallelForTask(ctx context.Context, task *store.TaskMessage) (int, error) {
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return 0, errors.Errorf("plan %v not found", task.PlanID)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return 0, errors.Errorf("project %v not found", plan.ProjectID)
	}

	return int(project.Setting.GetParallelTasksPerRollout()), nil
}

func (s *Scheduler) promoteTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	s.bus.TaskRunSchedulerInfo.Delete(taskRun.ID)

	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:      taskRun.ID,
		Updater: common.SystemBotEmail,
		Status:  storepb.TaskRun_AVAILABLE,
	}); err != nil {
		return errors.Wrapf(err, "failed to update task run status to available")
	}

	select {
	case s.bus.TaskRunTickleChan <- 0:
	default:
	}

	return nil
}

// schedulingContext holds pre-fetched and indexed active task run data for a scheduling cycle.
type schedulingContext struct {
	// Pre-indexed active task runs (AVAILABLE + RUNNING)
	activeByDatabase  map[string]int // dbKey -> blocking taskID (first found)
	activeCountByPlan map[int64]int  // planID -> active count

	// Tracks promotions this round
	promotedDBs    map[string]bool
	promotedCounts map[int64]int
}

func newSchedulingContext(ctx context.Context, s *store.Store) (*schedulingContext, error) {
	activeTaskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list active task runs")
	}

	sc := &schedulingContext{
		activeByDatabase:  make(map[string]int),
		activeCountByPlan: make(map[int64]int),
		promotedDBs:       make(map[string]bool),
		promotedCounts:    make(map[int64]int),
	}

	if len(activeTaskRuns) == 0 {
		return sc, nil
	}

	// Batch fetch all tasks
	taskIDs := make([]int, len(activeTaskRuns))
	for i, tr := range activeTaskRuns {
		taskIDs[i] = tr.TaskUID
	}

	tasks, err := s.ListTasks(ctx, &store.TaskFind{IDs: &taskIDs})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list tasks")
	}

	taskByID := make(map[int]*store.TaskMessage, len(tasks))
	for _, t := range tasks {
		taskByID[t.ID] = t
	}

	// Build indexes
	for _, tr := range activeTaskRuns {
		task := taskByID[tr.TaskUID]
		if task == nil {
			continue
		}
		sc.activeCountByPlan[task.PlanID]++

		if task.DatabaseName != nil && isSequentialTask(task) {
			dbKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)
			if _, exists := sc.activeByDatabase[dbKey]; !exists {
				sc.activeByDatabase[dbKey] = task.ID
			}
		}
	}

	return sc, nil
}

func (sc *schedulingContext) checkDatabaseMutualExclusion(task *store.TaskMessage) (canProceed bool, blockingTaskID *int) {
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

func (sc *schedulingContext) checkParallelLimit(planID int64, maxParallel int) bool {
	if maxParallel <= 0 {
		return true // no limit
	}
	currentCount := sc.activeCountByPlan[planID] + sc.promotedCounts[planID]
	return currentCount < maxParallel
}

func (sc *schedulingContext) markPromoted(task *store.TaskMessage) {
	if task.DatabaseName != nil && isSequentialTask(task) {
		sc.promotedDBs[getDatabaseKey(task.InstanceID, *task.DatabaseName)] = true
	}
	sc.promotedCounts[task.PlanID]++
}
