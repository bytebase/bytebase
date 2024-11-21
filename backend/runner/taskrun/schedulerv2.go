package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	taskSchedulerInterval = 5 * time.Second
)

// SchedulerV2 is the V2 scheduler for task run.
type SchedulerV2 struct {
	store          *store.Store
	stateCfg       *state.State
	webhookManager *webhook.Manager
	executorMap    map[api.TaskType]Executor
	profile        *config.Profile
	licenseService enterprise.LicenseService
}

// NewSchedulerV2 will create a new scheduler.
func NewSchedulerV2(
	store *store.Store,
	stateCfg *state.State,
	webhookManager *webhook.Manager,
	profile *config.Profile,
	licenseService enterprise.LicenseService,
) *SchedulerV2 {
	return &SchedulerV2{
		store:          store,
		stateCfg:       stateCfg,
		webhookManager: webhookManager,
		profile:        profile,
		executorMap:    map[api.TaskType]Executor{},
		licenseService: licenseService,
	}
}

// Register will register a task executor factory.
func (s *SchedulerV2) Register(taskType api.TaskType, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorMap[taskType] = executorGetter
}

// Run will start the scheduler.
func (s *SchedulerV2) Run(ctx context.Context, wg *sync.WaitGroup) {
	go s.ListenTaskSkippedOrDone(ctx)

	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Task scheduler V2 started and will run every %v", taskSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-s.stateCfg.TaskRunTickleChan:
			s.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerV2) runOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Task scheduler V2 PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	if err := s.scheduleAutoRolloutTasks(ctx); err != nil {
		slog.Error("failed to schedule auto rollout tasks", log.BBError(err))
	}

	if err := s.schedulePendingTaskRuns(ctx); err != nil {
		slog.Error("failed to schedule pending task runs", log.BBError(err))
	}

	if err := s.scheduleRunningTaskRuns(ctx); err != nil {
		slog.Error("failed to schedule running task runs", log.BBError(err))
	}
}

func (s *SchedulerV2) scheduleAutoRolloutTasks(ctx context.Context) error {
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{ShowDeleted: true})
	if err != nil {
		return errors.Wrapf(err, "failed to list environments")
	}

	for _, environment := range environments {
		policy, err := s.store.GetRolloutPolicy(ctx, environment.UID)
		if err != nil {
			return errors.Wrapf(err, "failed to get rollout policy for environment %d", environment.UID)
		}

		taskIDs, err := s.store.ListTasksToAutoRollout(ctx, []int{environment.UID})
		if err != nil {
			return errors.Wrapf(err, "failed to list tasks with zero task run")
		}
		for _, taskID := range taskIDs {
			if err := s.scheduleAutoRolloutTask(ctx, policy, taskID); err != nil {
				slog.Error("failed to schedule auto rollout task", log.BBError(err))
			}
		}
	}

	return nil
}

func (s *SchedulerV2) canTaskAutoRollout(ctx context.Context, rolloutPolicy *storepb.RolloutPolicy, task *store.TaskMessage) (bool, error) {
	if rolloutPolicy.Automatic {
		return true, nil
	}

	if s.licenseService.IsFeatureEnabled(api.FeatureRolloutPolicy) != nil {
		// nolint:nilerr
		return true, nil
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return false, err
	}
	if instance == nil || instance.Deleted {
		return false, nil
	}

	if s.licenseService.IsFeatureEnabledForInstance(api.FeatureRolloutPolicy, instance) != nil {
		// nolint:nilerr
		return true, nil
	}

	return false, nil
}

func (s *SchedulerV2) scheduleAutoRolloutTask(ctx context.Context, rolloutPolicy *storepb.RolloutPolicy, taskUID int) error {
	task, err := s.store.GetTaskV2ByID(ctx, taskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task == nil {
		return nil
	}

	canAutoRollout, err := s.canTaskAutoRollout(ctx, rolloutPolicy, task)
	if err != nil {
		return err
	}
	if !canAutoRollout {
		return nil
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return err
	}
	if instance.Deleted {
		return nil
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return err
	}
	if issue != nil {
		if issue.Status != api.IssueOpen {
			return nil
		}
		approved, err := utils.CheckIssueApproved(issue)
		if err != nil {
			return errors.Wrapf(err, "failed to check if the issue is approved")
		}
		if !approved {
			return nil
		}
	}

	// the latest checks of the plan must pass
	pass, err := func() (bool, error) {
		plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{PipelineID: &task.PipelineID})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get plan")
		}
		if plan == nil {
			return true, nil
		}
		latestRuns, err := s.store.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
			PlanUID:    &plan.UID,
			LatestOnly: true,
		})
		if err != nil {
			return false, errors.Wrapf(err, "failed to list latest plan check runs")
		}
		for _, run := range latestRuns {
			if run.Status != store.PlanCheckRunStatusDone {
				return false, nil
			}
			for _, result := range run.Result.Results {
				if result.Status != storepb.PlanCheckRunResult_Result_SUCCESS {
					return false, nil
				}
			}
		}
		return true, nil
	}()
	if err != nil {
		return errors.Wrapf(err, "failed to check if plan check passes")
	}
	if !pass {
		return nil
	}

	sheetUID, err := api.GetSheetUIDFromTaskPayload(task.Payload)
	if err != nil {
		return errors.Wrapf(err, "failed to get sheet uid")
	}

	create := &store.TaskRunMessage{
		CreatorID: api.SystemBotID,
		TaskUID:   task.ID,
		SheetUID:  sheetUID,
		Name:      fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
	}

	if err := s.store.CreatePendingTaskRuns(ctx, create); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	if issue != nil {
		tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.StageID, taskUID)}
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_PENDING, api.SystemBotID, ""); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
		}

		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    webhook.EventTypeTaskRunStatusUpdate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
				Title:  issue.Title,
				Status: api.TaskRunPending.String(),
			},
		})
	}

	return nil
}

func (s *SchedulerV2) schedulePendingTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		Status: &[]api.TaskRunStatus{api.TaskRunPending},
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

func (s *SchedulerV2) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// here, we move pending taskruns to running taskruns which means they are ready to be executed.
	// pending taskruns remain pending if
	// 1. earliestAllowedTs not met.
	// 2. blocked by other tasks via TaskDAG
	// 3. for versioned tasks, there are other versioned tasks on the same database with
	// a smaller version not finished yet. we need to wait for those first.
	task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task.EarliestAllowedTs != 0 && time.Now().Before(time.Unix(task.EarliestAllowedTs, 0)) {
		return nil
	}
	for _, blockingTaskUID := range task.DependsOn {
		blockingTask, err := s.store.GetTaskV2ByID(ctx, blockingTaskUID)
		if err != nil {
			return errors.Wrapf(err, "failed to get blocking task %v", blockingTaskUID)
		}

		skipped := struct {
			Skipped bool `json:"skipped"`
		}{}
		if err := json.Unmarshal([]byte(blockingTask.Payload), &skipped); err != nil {
			return errors.Wrapf(err, "failed to unmarshal payload")
		}
		if skipped.Skipped {
			continue
		}

		if blockingTask.LatestTaskRunStatus != api.TaskRunDone {
			return nil
		}
	}

	if common.IsDev() && s.profile.DevelopmentVersioned {
		doSchedule, err := func() (bool, error) {
			if task.DatabaseID == nil {
				return true, nil
			}

			var version struct {
				Version string `json:"schemaVersion"`
			}
			if err := json.Unmarshal([]byte(task.Payload), &version); err != nil {
				return false, errors.Wrapf(err, "failed to unmarshal task payload")
			}
			if version.Version == "" {
				return true, nil
			}

			taskIDs, err := s.store.FindBlockingTasksByVersion(ctx, *task.DatabaseID, version.Version)
			if err != nil {
				return false, errors.Wrapf(err, "failed to find blocking versioned tasks")
			}
			if len(taskIDs) > 0 {
				s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
					ReportTime: timestamppb.Now(),
					WaitingCause: &storepb.SchedulerInfo_WaitingCause{
						Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
							TaskUid: int32(taskIDs[0]),
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
	}

	if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
		ID:        taskRun.ID,
		UpdaterID: api.SystemBotID,
		Status:    api.TaskRunRunning,
	}); err != nil {
		return errors.Wrapf(err, "failed to update task run status to running")
	}
	s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
		Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
		TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
			Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_WAITING,
		},
	})
	return nil
}

func (s *SchedulerV2) scheduleRunningTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
		Status: &[]api.TaskRunStatus{api.TaskRunRunning},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list pending tasks")
	}

	// TODO(p0ny): remove these because we will follow the version order
	// Find the minimum task ID for each database.
	// We only run the first (i.e. which has the minimum task ID) task for each database.
	minTaskIDForDatabase := map[int]int{}
	for _, taskRun := range taskRuns {
		task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
		if err != nil {
			slog.Error("failed to get task", slog.Int("task id", taskRun.TaskUID), log.BBError(err))
			continue
		}
		if task.DatabaseID == nil {
			continue
		}

		if task.Type.Sequential() {
			if _, ok := minTaskIDForDatabase[*task.DatabaseID]; !ok {
				minTaskIDForDatabase[*task.DatabaseID] = task.ID
			} else if minTaskIDForDatabase[*task.DatabaseID] > task.ID {
				minTaskIDForDatabase[*task.DatabaseID] = task.ID
			}
		}
	}

	for _, taskRun := range taskRuns {
		// Skip the task run if it is already executing.
		if _, ok := s.stateCfg.RunningTaskRuns.Load(taskRun.ID); ok {
			continue
		}
		task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
		if err != nil {
			slog.Error("failed to get task", slog.Int("task id", taskRun.TaskUID), log.BBError(err))
			continue
		}
		if task.DatabaseID != nil && task.Type.Sequential() {
			// Skip the task run if there is an ongoing migration on the database.
			if taskUIDAny, ok := s.stateCfg.RunningDatabaseMigration.Load(*task.DatabaseID); ok {
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
				continue
			}
			if taskUID := minTaskIDForDatabase[*task.DatabaseID]; taskUID != task.ID {
				s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
					ReportTime: timestamppb.Now(),
					WaitingCause: &storepb.SchedulerInfo_WaitingCause{
						Cause: &storepb.SchedulerInfo_WaitingCause_TaskUid{
							TaskUid: int32(taskUID),
						},
					},
				})
				continue
			}
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			continue
		}
		if instance.Deleted {
			continue
		}
		executor, ok := s.executorMap[task.Type]
		if !ok {
			slog.Error("Skip running task with unknown type",
				slog.Int("id", task.ID),
				slog.String("name", task.Name),
				slog.String("type", string(task.Type)),
			)
			continue
		}
		maximumConnections := int(instance.Options.GetMaximumConnections())
		if s.stateCfg.InstanceOutstandingConnections.Increment(task.InstanceID, maximumConnections) {
			s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
				ReportTime: timestamppb.Now(),
				WaitingCause: &storepb.SchedulerInfo_WaitingCause{
					Cause: &storepb.SchedulerInfo_WaitingCause_ConnectionLimit{
						ConnectionLimit: true,
					},
				},
			})
			continue
		}

		s.stateCfg.TaskRunSchedulerInfo.Delete(taskRun.ID)

		s.stateCfg.RunningTaskRuns.Store(taskRun.ID, true)
		if task.DatabaseID != nil {
			s.stateCfg.RunningDatabaseMigration.Store(*task.DatabaseID, task.ID)
		}

		s.store.CreateTaskRunLogS(ctx, taskRun.ID, time.Now(), s.profile.DeployID, &storepb.TaskRunLog{
			Type: storepb.TaskRunLog_TASK_RUN_STATUS_UPDATE,
			TaskRunStatusUpdate: &storepb.TaskRunLog_TaskRunStatusUpdate{
				Status: storepb.TaskRunLog_TaskRunStatusUpdate_RUNNING_RUNNING,
			},
		})
		go s.runTaskRunOnce(ctx, taskRun, task, executor)
	}

	return nil
}

func (s *SchedulerV2) runTaskRunOnce(ctx context.Context, taskRun *store.TaskRunMessage, task *store.TaskMessage, executor Executor) {
	defer func() {
		// We don't need to do s.stateCfg.RunningTaskRuns.Delete(taskRun.ID) to avoid race condition.
		s.stateCfg.RunningTaskRunsCancelFunc.Delete(taskRun.ID)
		if task.DatabaseID != nil {
			s.stateCfg.RunningDatabaseMigration.Delete(*task.DatabaseID)
		}
		s.stateCfg.InstanceOutstandingConnections.Decrement(task.InstanceID)
	}()

	driverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.stateCfg.RunningTaskRunsCancelFunc.Store(taskRun.ID, cancel)

	done, result, err := RunExecutorOnce(ctx, driverCtx, executor, task, taskRun.ID)

	if !done && err != nil {
		slog.Debug("Encountered transient error running task, will retry",
			slog.Int("id", task.ID),
			slog.String("name", task.Name),
			slog.String("type", string(task.Type)),
			log.BBError(err),
		)
		return
	}

	if done && err != nil && errors.Is(err, context.Canceled) {
		slog.Warn("task run is canceled",
			slog.Int("id", task.ID),
			slog.String("name", task.Name),
			slog.String("type", string(task.Type)),
			log.BBError(err),
		)
		resultBytes, marshalErr := protojson.Marshal(&storepb.TaskRunResult{
			Detail:        "The task run is canceled",
			ChangeHistory: "",
			Version:       "",
		})
		if marshalErr != nil {
			slog.Error("Failed to marshal task run result",
				slog.Int("task_id", task.ID),
				slog.String("type", string(task.Type)),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.Ok
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunCanceled,
			Code:      &code,
			Result:    &result,
		}

		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as CANCELED",
				slog.Int("id", task.ID),
				slog.String("name", task.Name),
				log.BBError(err),
			)
			return
		}
		return
	}

	if done && err != nil {
		slog.Warn("task run failed",
			slog.Int("id", task.ID),
			slog.String("name", task.Name),
			slog.String("type", string(task.Type)),
			log.BBError(err),
		)

		taskRunResult := &storepb.TaskRunResult{
			Detail:        err.Error(),
			ChangeHistory: "",
			Version:       "",
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
				slog.String("type", string(task.Type)),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.ErrorCode(err)
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunFailed,
			Code:      &code,
			Result:    &result,
		}

		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as FAILED",
				slog.Int("id", task.ID),
				slog.String("name", task.Name),
				log.BBError(err),
			)
			return
		}

		if err := func() error {
			issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{
				PipelineID: &task.PipelineID,
			})
			if err != nil {
				return errors.Wrap(err, "failed to get issue")
			}
			if issue == nil {
				return nil
			}
			tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.StageID, task.ID)}
			return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_FAILED, api.SystemBotID, "")
		}(); err != nil {
			slog.Warn("failed to create issue comment", log.BBError(err))
		}

		s.createActivityForTaskRunStatusUpdate(ctx, task, api.TaskRunFailed, taskRunResult.Detail)
		return
	}

	if done && err == nil {
		resultBytes, marshalErr := protojson.Marshal(result)
		if marshalErr != nil {
			slog.Error("Failed to marshal task run result",
				slog.Int("task_id", task.ID),
				slog.String("type", string(task.Type)),
				log.BBError(marshalErr),
			)
			return
		}
		code := common.Ok
		result := string(resultBytes)
		taskRunStatusPatch := &store.TaskRunStatusPatch{
			ID:        taskRun.ID,
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunDone,
			Code:      &code,
			Result:    &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as DONE",
				slog.Int("id", task.ID),
				slog.String("name", task.Name),
				log.BBError(err),
			)
			return
		}

		if err := func() error {
			issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{
				PipelineID: &task.PipelineID,
			})
			if err != nil {
				return errors.Wrap(err, "failed to get issue")
			}
			if issue == nil {
				return nil
			}
			tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.StageID, task.ID)}
			return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_DONE, api.SystemBotID, "")
		}(); err != nil {
			slog.Warn("failed to create issue comment", log.BBError(err))
		}

		s.createActivityForTaskRunStatusUpdate(ctx, task, api.TaskRunDone, "")
		s.stateCfg.TaskSkippedOrDoneChan <- task.ID
		return
	}
}

func (s *SchedulerV2) ListenTaskSkippedOrDone(ctx context.Context) {
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
	stageDoneConfirmed := map[int]bool{}

	for {
		select {
		case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
			if err := func() error {
				task, err := s.store.GetTaskV2ByID(ctx, taskUID)
				if err != nil {
					return errors.Wrapf(err, "failed to get task")
				}
				if stageDoneConfirmed[task.StageID] {
					return nil
				}

				stageTasks, err := s.store.ListTasks(ctx, &api.TaskFind{StageID: &task.StageID})
				if err != nil {
					return errors.Wrapf(err, "failed to list tasks")
				}

				skippedOrDone, err := tasksSkippedOrDone(stageTasks)
				if err != nil {
					return errors.Wrapf(err, "failed to check if tasks are skipped or done")
				}
				if !skippedOrDone {
					return nil
				}

				stageDoneConfirmed[task.StageID] = true

				stages, err := s.store.ListStageV2(ctx, task.PipelineID)
				if err != nil {
					return errors.Wrapf(err, "failed to list stages")
				}

				var taskStage *store.StageMessage
				var nextStage *store.StageMessage
				var pipelineDone bool
				for i, stage := range stages {
					if stage.ID == task.StageID {
						taskStage = stages[i]
						if i < len(stages)-1 {
							nextStage = stages[i+1]
						}
						if i == len(stages)-1 {
							pipelineDone = true
						}
						break
					}
				}
				if taskStage == nil {
					return errors.Errorf("failed to find stage")
				}

				issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
				if err != nil {
					return errors.Wrapf(err, "failed to get issue")
				}
				if issue == nil {
					return nil
				}

				if err := func() error {
					p := &storepb.IssueCommentPayload{
						Event: &storepb.IssueCommentPayload_StageEnd_{
							StageEnd: &storepb.IssueCommentPayload_StageEnd{
								Stage: common.FormatStage(issue.Project.ResourceID, taskStage.PipelineID, taskStage.ID),
							},
						},
					}
					_, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
						IssueUID: issue.UID,
						Payload:  p,
					}, api.SystemBotID)
					return err
				}(); err != nil {
					slog.Warn("failed to create issue comment", log.BBError(err))
				}

				// every task in the stage terminated
				// create "stage ends" activity.
				if err := func() error {
					s.webhookManager.CreateEvent(ctx, &webhook.Event{
						Actor:   s.store.GetSystemBotUser(ctx),
						Type:    webhook.EventTypeStageStatusUpdate,
						Comment: "",
						Issue:   webhook.NewIssue(issue),
						Project: webhook.NewProject(issue.Project),
						StageStatusUpdate: &webhook.EventStageStatusUpdate{
							StageTitle: taskStage.Name,
							StageUID:   taskStage.ID,
						},
					})
					return nil
				}(); err != nil {
					slog.Error("failed to create ActivityPipelineStageStatusUpdate activity", log.BBError(err))
				}
				// create "notify pipeline rollout" activity.
				if err := func() error {
					if nextStage == nil {
						return nil
					}
					policy, err := s.store.GetRolloutPolicy(ctx, nextStage.EnvironmentID)
					if err != nil {
						return errors.Wrapf(err, "failed to get rollout policy")
					}
					s.webhookManager.CreateEvent(ctx, &webhook.Event{
						Actor:   s.store.GetSystemBotUser(ctx),
						Type:    webhook.EventTypeIssueRolloutReady,
						Comment: "",
						Issue:   webhook.NewIssue(issue),
						Project: webhook.NewProject(issue.Project),
						IssueRolloutReady: &webhook.EventIssueRolloutReady{
							RolloutPolicy: policy,
							StageName:     nextStage.Name,
						},
					})
					return nil
				}(); err != nil {
					slog.Error("failed to create rollout release notification activity", log.BBError(err))
				}

				// After all tasks in the pipeline are done, we will resolve the issue if the issue is auto-resolvable.
				if issue.Project.Setting.AutoResolveIssue && pipelineDone {
					if err := func() error {
						// For those database data export issues, we don't resolve them automatically.
						if issue.Type == api.IssueDatabaseDataExport {
							return nil
						}

						newStatus := api.IssueDone
						updatedIssue, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{Status: &newStatus}, api.SystemBotID)
						if err != nil {
							return errors.Wrapf(err, "failed to update issue status")
						}

						fromStatus := storepb.IssueCommentPayload_IssueUpdate_IssueStatus(storepb.IssueCommentPayload_IssueUpdate_IssueStatus_value[issue.Status.String()])
						toStatus := storepb.IssueCommentPayload_IssueUpdate_IssueStatus(storepb.IssueCommentPayload_IssueUpdate_IssueStatus_value[updatedIssue.Status.String()])
						if _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
							IssueUID: issue.UID,
							Payload: &storepb.IssueCommentPayload{
								Event: &storepb.IssueCommentPayload_IssueUpdate_{
									IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
										FromStatus: &fromStatus,
										ToStatus:   &toStatus,
									},
								},
							},
						}, api.SystemBotID); err != nil {
							return errors.Wrapf(err, "failed to create issue comment after changing the issue status")
						}

						s.webhookManager.CreateEvent(ctx, &webhook.Event{
							Actor:   s.store.GetSystemBotUser(ctx),
							Type:    webhook.EventTypeIssueStatusUpdate,
							Comment: "",
							Issue:   webhook.NewIssue(updatedIssue),
							Project: webhook.NewProject(updatedIssue.Project),
						})

						return nil
					}(); err != nil {
						slog.Error("failed to update issue status", log.BBError(err))
					}
				}
				return nil
			}(); err != nil {
				slog.Error("failed to handle task skipped or done", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *SchedulerV2) createActivityForTaskRunStatusUpdate(ctx context.Context, task *store.TaskMessage, newStatus api.TaskRunStatus, errDetail string) {
	if err := func() error {
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{
			PipelineID: &task.PipelineID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get issue")
		}
		if issue == nil {
			return nil
		}
		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    webhook.EventTypeTaskRunStatusUpdate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Project: webhook.NewProject(issue.Project),
			TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
				Title:  task.Name,
				Status: newStatus.String(),
				Detail: errDetail,
			},
		})
		return nil
	}(); err != nil {
		slog.Error("failed to create activity for task run status update", log.BBError(err))
	}
}

func tasksSkippedOrDone(tasks []*store.TaskMessage) (bool, error) {
	for _, task := range tasks {
		skipped, err := utils.GetTaskSkipped(task)
		if err != nil {
			return false, err
		}
		done := task.LatestTaskRunStatus == api.TaskRunDone
		if !skipped && !done {
			return false, nil
		}
	}
	return true, nil
}
