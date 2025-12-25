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
	for _, taskRun := range taskRuns {
		if err := s.schedulePendingTaskRun(ctx, taskRun); err != nil {
			slog.Error("failed to schedule pending task run", log.BBError(err))
		}
	}

	return nil
}

func (s *Scheduler) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// here, we move pending taskruns to running taskruns which means they are ready to be executed.
	// pending taskruns remain pending if
	// 1. taskRun.RunAt not met.
	// 2. for versioned tasks, there are other versioned tasks on the same database with
	// a smaller version not finished yet. we need to wait for those first.
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if taskRun.RunAt != nil && time.Now().Before(*taskRun.RunAt) {
		return nil
	}

	doSchedule, err := func() (bool, error) {
		if task.DatabaseName == nil {
			return true, nil
		}

		schemaVersion := task.Payload.GetSchemaVersion()
		if schemaVersion == "" {
			return true, nil
		}

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
		s.stateCfg.TaskRunSchedulerInfo.Delete(taskRun.ID)
		return true, nil
	}()
	if err != nil {
		return errors.Wrapf(err, "failed to check blocking versioned tasks")
	}
	if !doSchedule {
		return nil
	}

	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:      taskRun.ID,
		Updater: common.SystemBotEmail,
		Status:  storepb.TaskRun_RUNNING,
	}); err != nil {
		return errors.Wrapf(err, "failed to update task run status to running")
	}
	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_WAITING,
		},
	})

	// Tickle the running scheduler to immediately pick up this newly RUNNING task
	// This prevents delays when user clicks "Run stage" in the UI
	select {
	case s.stateCfg.TaskRunTickleChan <- 0:
	default:
		// Channel is full, running scheduler will pick it up on next tick
	}

	return nil
}
