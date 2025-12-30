package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	taskRuns, err := s.store.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		Status: &[]storepb.TaskRun_Status{storepb.TaskRun_RUNNING},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending tasks")
	}

	// Find the minimum task ID for each database.
	// We only run the first (i.e. which has the minimum task ID) task for each database.
	// 1. For ddl tasks, we run them one by one to get a sane schema dump and thus diff.
	// 2. For versioned tasks, this is our last resort to determine the order for tasks with the same version. We don't want to run them in parallel.
	// 2.1. Rollout 1 tasks will be run before rollout 2 tasks. Where, rollout 1 tasks are created before rollout 2 tasks.
	minTaskIDForDatabase := map[string]int{}
	for _, taskRun := range taskRuns {
		task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
		if err != nil {
			slog.Error("failed to get task", slog.Int("task id", taskRun.TaskUID), log.BBError(err))
			continue
		}
		if task.DatabaseName == nil {
			continue
		}

		databaseKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)
		if isSequentialTask(task) {
			if _, ok := minTaskIDForDatabase[databaseKey]; !ok {
				minTaskIDForDatabase[databaseKey] = task.ID
			} else if minTaskIDForDatabase[databaseKey] > task.ID {
				minTaskIDForDatabase[databaseKey] = task.ID
			}
		}
	}

	for _, taskRun := range taskRuns {
		if err := s.scheduleRunningTaskRun(ctx, taskRun, minTaskIDForDatabase); err != nil {
			slog.Error("failed to schedule running task run", log.BBError(err))
		}
	}

	return nil
}

func (s *Scheduler) scheduleRunningTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, minTaskIDForDatabase map[string]int) error {
	// Skip the task run if it is already executing.
	if _, ok := s.stateCfg.RunningTaskRuns.Load(taskRun.ID); ok {
		return nil
	}
	task, err := s.store.GetTaskByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task.DatabaseName != nil && isSequentialTask(task) {
		// Skip the task run if there is an ongoing migration on the database.
		if taskUIDAny, ok := s.stateCfg.RunningDatabaseMigration.Load(getDatabaseKey(task.InstanceID, *task.DatabaseName)); ok {
			if taskUID, ok := taskUIDAny.(int); ok {
				s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
					ReportTime: timestamppb.Now(),
					WaitingCause: &storepb.SchedulerInfo_WaitingCause{
						Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
							TaskUid: int32(taskUID),
						},
					},
				})
			}
			return nil
		}
		if taskUID := minTaskIDForDatabase[getDatabaseKey(task.InstanceID, *task.DatabaseName)]; taskUID != task.ID {
			s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
				ReportTime: timestamppb.Now(),
				WaitingCause: &storepb.SchedulerInfo_WaitingCause{
					Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
						TaskUid: int32(taskUID),
					},
				},
			})
			return nil
		}
	}

	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return errors.Wrapf(err, "failed to get instance")
	}
	if instance == nil {
		return errors.Errorf("instance %v not found", task.InstanceID)
	}
	if instance.Deleted {
		return errors.Errorf("instance %v is deleted", task.InstanceID)
	}
	executor, ok := s.executorMap[task.Type]
	if !ok {
		return errors.Errorf("executor not found for task type: %v", task.Type)
	}

	// Check max running task runs per rollout.
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

	planID := strconv.FormatInt(plan.UID, 10)
	maxRunningTaskRunsPerRollout := int(project.Setting.GetParallelTasksPerRollout())
	if maxRunningTaskRunsPerRollout <= 0 {
		maxRunningTaskRunsPerRollout = defaultRolloutMaxRunningTaskRuns
	}
	if s.stateCfg.RolloutOutstandingTasks.Increment(planID+"/"+task.InstanceID, maxRunningTaskRunsPerRollout) {
		s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
			ReportTime: timestamppb.Now(),
			WaitingCause: &storepb.SchedulerInfo_WaitingCause{
				Cause: &storepb.SchedulerInfo_WaitingCause_ParallelTasksLimit{
					ParallelTasksLimit: true,
				},
			},
		})
		return nil
	}

	// decrement the connection count if we return below.
	revertRolloutConnectionsIncrement := true
	defer func() {
		if revertRolloutConnectionsIncrement {
			s.stateCfg.RolloutOutstandingTasks.Decrement(planID + "/" + task.InstanceID)
		}
	}()

	// Set taskrun StartAt when it's about to run.
	// So that the waiting time is not taken into account of the actual execution time.
	if err := s.store.UpdateTaskRunStartAt(ctx, taskRun.ID); err != nil {
		return errors.Wrapf(err, "failed to update task run start at")
	}

	// We MUST NOT return early below this line.
	// If we do want to return early, we must revert related states.
	s.stateCfg.TaskRunSchedulerInfo.Delete(taskRun.ID)
	s.stateCfg.RunningTaskRuns.Store(taskRun.ID, true)
	if task.DatabaseName != nil {
		s.stateCfg.RunningDatabaseMigration.Store(getDatabaseKey(task.InstanceID, *task.DatabaseName), task.ID)
	}

	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_RUNNING,
		},
	})

	// We are sure that we will run the task.
	// The executor will decrement them.
	revertRolloutConnectionsIncrement = false
	go s.runTaskRunOnce(ctx, taskRun, task, executor)
	return nil
}

func (s *Scheduler) runTaskRunOnce(ctx context.Context, taskRun *store.TaskRunMessage, task *store.TaskMessage, executor Executor) {
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
		// We don't need to do s.stateCfg.RunningTaskRuns.Delete(taskRun.ID) to avoid race condition.
		s.stateCfg.RunningTaskRunsCancelFunc.Delete(taskRun.ID)
		if task.DatabaseName != nil {
			s.stateCfg.RunningDatabaseMigration.Delete(getDatabaseKey(task.InstanceID, *task.DatabaseName))
		}
		s.stateCfg.RolloutOutstandingTasks.Decrement(strconv.FormatInt(task.PlanID, 10) + "/" + task.InstanceID)
	}()

	driverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.stateCfg.RunningTaskRunsCancelFunc.Store(taskRun.ID, cancel)

	done, result, err := RunExecutorOnce(ctx, driverCtx, executor, task, taskRun.ID)

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
			ID:      taskRun.ID,
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
			ID:      taskRun.ID,
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
			ID:      taskRun.ID,
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

func (s *Scheduler) createActivityForTaskRunStatusUpdate(ctx context.Context, task *store.TaskMessage, newStatus storepb.TaskRun_Status, errDetail string) {
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
		issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{
			PlanUID: &task.PlanID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get issue")
		}
		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   store.SystemBotUser,
			Type:    storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Rollout: webhook.NewRollout(plan),
			Project: webhook.NewProject(project),
			TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
				Title:  task.GetDatabaseName(),
				Status: newStatus.String(),
				Detail: errDetail,
			},
		})
		return nil
	}(); err != nil {
		slog.Error("failed to create activity for task run status update", log.BBError(err))
	}
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

		issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &task.PlanID})
		if err != nil {
			return errors.Wrap(err, "failed to get issue")
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
				firstFailureTime := time.Now().Add(-5 * time.Minute)
				if len(failedTasks) > 0 {
					firstFailureTime = failedTasks[0].FailedAt
				}

				// Use background context to avoid cancellation issues
				webhookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				s.webhookManager.CreateEvent(webhookCtx, &webhook.Event{
					Actor:   store.SystemBotUser,
					Type:    storepb.Activity_PIPELINE_FAILED,
					Project: webhook.NewProject(project),
					Issue:   webhook.NewIssue(issue),
					Rollout: webhook.NewRollout(plan),
					PipelineFailed: &webhook.EventPipelineFailed{
						FailedTasks:      failedTasks,
						FirstFailureTime: firstFailureTime,
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
