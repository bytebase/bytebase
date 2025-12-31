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

// checkDatabaseMutualExclusion checks if there's already an AVAILABLE or RUNNING task on the database.
// Returns true if the task can proceed (no conflict), false otherwise.
func (s *Scheduler) checkDatabaseMutualExclusion(ctx context.Context, task *store.TaskMessage, availableDBs map[string]bool) (bool, *int, error) {
	if task.DatabaseName == nil {
		return true, nil, nil
	}
	if !isSequentialTask(task) {
		return true, nil, nil
	}

	databaseKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)

	// Check in-memory tracking first (tasks promoted this round)
	if availableDBs[databaseKey] {
		return false, nil, nil
	}

	// Check database for AVAILABLE or RUNNING tasks on the same database
	activeTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return false, nil, errors.Wrapf(err, "failed to list active task runs")
	}

	for _, tr := range activeTaskRuns {
		activeTask, err := s.store.GetTaskByID(ctx, tr.TaskUID)
		if err != nil {
			return false, nil, errors.Wrapf(err, "failed to get task")
		}
		if activeTask.DatabaseName == nil {
			continue
		}
		if !isSequentialTask(activeTask) {
			continue
		}
		if getDatabaseKey(activeTask.InstanceID, *activeTask.DatabaseName) == databaseKey {
			return false, &activeTask.ID, nil
		}
	}

	return true, nil, nil
}

// checkParallelLimit checks if promoting this task would exceed the parallel task limit.
// Returns true if the task can proceed, false otherwise.
func (s *Scheduler) checkParallelLimit(ctx context.Context, task *store.TaskMessage, rolloutCounts map[int64]int) (bool, error) {
	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get plan")
	}
	if plan == nil {
		return false, errors.Errorf("plan %v not found", task.PlanID)
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return false, errors.Errorf("project %v not found", plan.ProjectID)
	}

	maxParallel := int(project.Setting.GetParallelTasksPerRollout())
	if maxParallel <= 0 {
		// No limit
		return true, nil
	}

	// Count current AVAILABLE + RUNNING for this rollout
	activeTaskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		PlanUID: &task.PlanID,
		Status:  &[]storepb.TaskRun_Status{storepb.TaskRun_AVAILABLE, storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to list active task runs")
	}

	currentCount := len(activeTaskRuns) + rolloutCounts[task.PlanID]
	return currentCount < maxParallel, nil
}

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
		case <-s.stateCfg.TaskRunTickleChan:
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
		return errors.Wrapf(err, "failed to list pending tasks")
	}

	// Track what we've promoted this round to avoid over-committing
	availableDBs := map[string]bool{} // database key -> has AVAILABLE this round
	rolloutCounts := map[int64]int{}  // plan_id -> count promoted this round

	for _, taskRun := range taskRuns {
		promoted, err := s.schedulePendingTaskRun(ctx, taskRun, availableDBs, rolloutCounts)
		if err != nil {
			slog.Error("failed to schedule pending task run", log.BBError(err))
			continue
		}
		if promoted {
			task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
			if err != nil {
				slog.Error("failed to get task after promotion", log.BBError(err))
				continue
			}
			if task.DatabaseName != nil && isSequentialTask(task) {
				availableDBs[getDatabaseKey(task.InstanceID, *task.DatabaseName)] = true
			}
			rolloutCounts[task.PlanID]++
		}
	}

	return nil
}

func (s *Scheduler) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, availableDBs map[string]bool, rolloutCounts map[int64]int) (bool, error) {
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get task")
	}

	// Check 1: RunAt time
	if taskRun.RunAt != nil && time.Now().Before(*taskRun.RunAt) {
		return false, nil
	}

	// Check 2: Version ordering (blocking tasks with smaller versions)
	if task.DatabaseName != nil {
		schemaVersion := task.Payload.GetSchemaVersion()
		if schemaVersion != "" {
			maybeTaskID, err := s.store.FindBlockingTaskByVersion(ctx, task.PlanID, task.InstanceID, *task.DatabaseName, schemaVersion)
			if err != nil {
				return false, errors.Wrapf(err, "failed to find blocking versioned tasks")
			}
			if maybeTaskID != nil {
				s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
					ReportTime: timestamppb.Now(),
					WaitingCause: &storepb.SchedulerInfo_WaitingCause{
						Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
							TaskUid: int32(*maybeTaskID),
						},
					},
				})
				return false, nil
			}
		}
	}

	// Check 3: Database mutual exclusion (for sequential tasks)
	canProceed, blockingTaskID, err := s.checkDatabaseMutualExclusion(ctx, task, availableDBs)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check database mutual exclusion")
	}
	if !canProceed {
		if blockingTaskID != nil {
			s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
				ReportTime: timestamppb.Now(),
				WaitingCause: &storepb.SchedulerInfo_WaitingCause{
					Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
						TaskUid: int32(*blockingTaskID),
					},
				},
			})
		}
		return false, nil
	}

	// Check 4: Parallel task limit per rollout
	withinLimit, err := s.checkParallelLimit(ctx, task, rolloutCounts)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check parallel limit")
	}
	if !withinLimit {
		s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
			ReportTime: timestamppb.Now(),
			WaitingCause: &storepb.SchedulerInfo_WaitingCause{
				Cause: &storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit{
					ParallelTasksLimit: true,
				},
			},
		})
		return false, nil
	}

	// All checks passed - promote to AVAILABLE
	s.stateCfg.TaskRunSchedulerInfo.Delete(taskRun.ID)
	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:      taskRun.ID,
		Updater: common.SystemBotEmail,
		Status:  storepb.TaskRun_AVAILABLE,
	}); err != nil {
		return false, errors.Wrapf(err, "failed to update task run status to available")
	}

	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_WAITING,
		},
	})

	// Tickle the running scheduler
	select {
	case s.stateCfg.TaskRunTickleChan <- 0:
	default:
	}

	return true, nil
}
