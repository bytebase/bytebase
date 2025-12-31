package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/webhook"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
		case <-s.stateCfg.TaskRunTickleChan:
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

	s.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_RUNNING,
		},
	})

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
		s.stateCfg.RunningTaskRunsCancelFunc.Delete(taskRunUID)
	}()

	driverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.stateCfg.RunningTaskRunsCancelFunc.Store(taskRunUID, cancel)

	done, result, err := RunExecutorOnce(ctx, driverCtx, executor, task, taskRunUID)

	switch {
	case !done && err != nil:
		slog.Debug("Encountered transient error running task, will retry",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			log.BBError(err),
		)
		return

	case done && err != nil && errors.Is(err, context.Canceled):
		slog.Warn("task run is canceled",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			log.BBError(err),
		)
		resultBytes, marshalErr := protojson.Marshal(&storepb.TaskRunResult{
			Detail:    "The task run is canceled",
			Changelog: "",
			Version:   "",
		})
		if marshalErr != nil {
			slog.Error("Failed to marshal task run result",
				slog.Int("task_id", task.ID),
				slog.String("type", task.Type.String()),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.Ok
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:      taskRunUID,
			Updater: common.SystemBotEmail,
			Status:  storepb.TaskRun_CANCELED,
			Code:    &code,
			Result:  &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as CANCELED",
				slog.Int("id", task.ID),
				log.BBError(err),
			)
			return
		}
		return

	case done && err != nil:
		slog.Warn("task run failed",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			log.BBError(err),
		)
		taskRunResult := &storepb.TaskRunResult{
			Detail:    err.Error(),
			Changelog: "",
			Version:   "",
		}
		var errWithPosition *db.ErrorWithPosition
		if errors.As(err, &errWithPosition) {
			taskRunResult.StartPosition = errWithPosition.Start
			taskRunResult.EndPosition = errWithPosition.End
		}
		resultBytes, marshalErr := protojson.Marshal(taskRunResult)
		if marshalErr != nil {
			slog.Error("Failed to marshal task run result",
				slog.Int("task_id", task.ID),
				slog.String("type", task.Type.String()),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.ErrorCode(err)
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:      taskRunUID,
			Updater: common.SystemBotEmail,
			Status:  storepb.TaskRun_FAILED,
			Code:    &code,
			Result:  &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as FAILED",
				slog.Int("id", task.ID),
				log.BBError(err),
			)
			return
		}
		s.createActivityForTaskRunStatusUpdate(ctx, task, storepb.TaskRun_FAILED, taskRunResult.Detail)
		s.recordPipelineFailure(ctx, task, taskRunResult.Detail)
		return

	case done && err == nil:
		resultBytes, marshalErr := protojson.Marshal(result)
		if marshalErr != nil {
			slog.Error("Failed to marshal task run result",
				slog.Int("task_id", task.ID),
				slog.String("type", task.Type.String()),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.Ok
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:      taskRunUID,
			Updater: common.SystemBotEmail,
			Status:  storepb.TaskRun_DONE,
			Code:    &code,
			Result:  &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as DONE",
				slog.Int("id", task.ID),
				log.BBError(err),
			)
			return
		}
		s.createActivityForTaskRunStatusUpdate(ctx, task, storepb.TaskRun_DONE, "")
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
		return
	default:
		// This case should not happen in normal flow, but adding for completeness
		slog.Error("Unexpected task execution state",
			slog.Int("id", task.ID),
			slog.String("type", task.Type.String()),
			slog.Bool("done", done),
			slog.Bool("has_error", err != nil),
		)
		return
	}
}

func (*Scheduler) createActivityForTaskRunStatusUpdate(_ context.Context, _ *store.TaskMessage, _ storepb.TaskRun_Status, _ string) {
	// No webhook events for task run status updates
}

func (s *Scheduler) recordPipelineFailure(ctx context.Context, task *store.TaskMessage, errDetail string) {
	if err := func() error {
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

		instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
		if err != nil {
			return errors.Wrapf(err, "failed to get instance")
		}

		failedTask := webhook.FailedTask{
			TaskID:       int64(task.ID),
			TaskName:     task.Type.String(),
			DatabaseName: task.GetDatabaseName(),
			InstanceName: instance.Metadata.Title,
			ErrorMessage: errDetail,
			FailedAt:     time.Now(),
		}

		s.pipelineEvents.RecordTaskFailure(
			plan.UID,
			failedTask,
			func(failedTasks []webhook.FailedTask) {
				// Use background context to avoid cancellation issues
				webhookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				s.webhookManager.CreateEvent(webhookCtx, &webhook.Event{
					Type:    storepb.Activity_PIPELINE_FAILED,
					Project: webhook.NewProject(project),
					RolloutFailed: &webhook.EventRolloutFailed{
						Rollout:     webhook.NewRollout(plan),
						FailedTasks: failedTasks,
					},
				})
			},
		)

		return nil
	}(); err != nil {
		slog.Error("failed to record pipeline failure", log.BBError(err))
	}
}

// isSequentialTask returns whether the task should be executed sequentially.
func isSequentialTask(task *store.TaskMessage) bool {
	//exhaustive:enforce
	switch task.Type {
	case storepb.Task_DATABASE_MIGRATE:
		// All DATABASE_MIGRATE operations (DDL/GHOST) should be sequential
		return true
	case storepb.Task_DATABASE_SDL:
		// SDL operations should be sequential
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
