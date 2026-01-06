package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func getDatabaseKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
}

// runRunningTaskRunsScheduler runs in a separate goroutine to schedule running task runs.
func (s *Scheduler) runRunningTaskRunsScheduler(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Running task runs scheduler started and will run every %v", taskSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			if err := s.scheduleRunningTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule running task runs", log.BBError(err))
			}
		case <-s.bus.TaskRunTickleChan:
			if err := s.scheduleRunningTaskRuns(ctx); err != nil {
				slog.Error("failed to schedule running task runs", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) scheduleRunningTaskRuns(ctx context.Context) error {
	// Atomically claim all AVAILABLE task runs
	claimed, err := s.store.ClaimAvailableTaskRuns(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to claim available task runs")
	}

	for _, c := range claimed {
		if err := s.executeTaskRun(ctx, c.TaskRunUID, c.TaskUID); err != nil {
			slog.Error("failed to execute task run", slog.Int("id", c.TaskRunUID), log.BBError(err))
		}
	}

	return nil
}

// executeTaskRun executes a task run that is already in RUNNING status.
func (s *Scheduler) executeTaskRun(ctx context.Context, taskRunUID, taskUID int) error {
	task, err := s.store.GetTaskByID(ctx, taskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task == nil {
		return errors.Errorf("task %v not found", taskUID)
	}

	executor, ok := s.executorMap[task.Type]
	if !ok {
		return errors.Errorf("executor not found for task type: %v", task.Type)
	}

	// Update started_at
	if err := s.store.UpdateTaskRunStartAt(ctx, taskRunUID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}

	go s.runTaskRunOnce(ctx, taskRunUID, task, executor)
	return nil
}

func (s *Scheduler) runTaskRunOnce(ctx context.Context, taskRunUID int, task *store.TaskMessage, executor Executor) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Task scheduler V2 runTaskRunOnce PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()
	defer func() {
		s.bus.RunningTaskRunsCancelFunc.Delete(taskRunUID)
	}()

	driverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.bus.RunningTaskRunsCancelFunc.Store(taskRunUID, cancel)

	result, err := RunExecutorOnce(ctx, driverCtx, executor, task, taskRunUID)

	if err != nil && errors.Is(err, context.Canceled) {
		slog.Warn("task run is canceled",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			log.BBError(err),
		)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:          taskRunUID,
			Updater:     common.SystemBotEmail,
			Status:      storepb.TaskRun_CANCELED,
			ResultProto: &storepb.TaskRunResult{},
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as CANCELED",
				slog.Int("id", task.ID),
				log.BBError(err),
			)
			return
		}
		return
	}

	if err != nil {
		slog.Warn("task run failed",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			log.BBError(err),
		)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:      taskRunUID,
			Updater: common.SystemBotEmail,
			Status:  storepb.TaskRun_FAILED,
			ResultProto: &storepb.TaskRunResult{
				Detail: err.Error(),
			},
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as FAILED",
				slog.Int("id", task.ID),
				log.BBError(err),
			)
			return
		}

		// Immediately try to send PIPELINE_FAILED webhook (HA-safe atomic claim)
		claimed, err := s.store.ClaimPipelineFailureNotification(ctx, task.PlanID)
		if err != nil {
			slog.Error("failed to claim pipeline failure notification", log.BBError(err))
		} else if claimed {
			// Get plan and project for webhook
			plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
			if err != nil || plan == nil {
				slog.Error("failed to get plan for failure webhook", log.BBError(err))
			} else {
				project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
				if err != nil || project == nil {
					slog.Error("failed to get project for failure webhook", log.BBError(err))
				} else {
					// Send PIPELINE_FAILED webhook
					s.webhookManager.CreateEvent(ctx, &webhook.Event{
						Type:    storepb.Activity_PIPELINE_FAILED,
						Project: webhook.NewProject(project),
						RolloutFailed: &webhook.EventRolloutFailed{
							Rollout: webhook.NewRollout(plan),
						},
					})
				}
			}
		}
		return
	}

	// Success case
	taskRunStatusPatch := &store.TaskRunStatusPatch{
		ID:          taskRunUID,
		Updater:     common.SystemBotEmail,
		Status:      storepb.TaskRun_DONE,
		ResultProto: result,
	}
	if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
		slog.Error("Failed to mark task as DONE",
			slog.Int("id", task.ID),
			log.BBError(err),
		)
		return
	}

	// Signal to check if plan is complete and successful (may send PIPELINE_COMPLETED)
	s.bus.PlanCompletionCheckChan <- task.PlanID
}

// isSequentialTask returns whether the task should be executed sequentially.
func isSequentialTask(task *store.TaskMessage) bool {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_MIGRATE:
		// All DATABASE_MIGRATE operations (DDL/GHOST) should be sequential
		return true
	case storepb.Task_DATABASE_CREATE,
		storepb.Task_DATABASE_EXPORT:
		return false
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return false
	default:
		return false
	}
}
