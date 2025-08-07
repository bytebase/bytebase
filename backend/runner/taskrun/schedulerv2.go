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
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	taskSchedulerInterval = 5 * time.Second
)

// defaultRolloutMaxRunningTaskRuns is the maximum number of running tasks per rollout.
// No limit by default.
const defaultRolloutMaxRunningTaskRuns = 0

// SchedulerV2 is the V2 scheduler for task run.
type SchedulerV2 struct {
	store          *store.Store
	stateCfg       *state.State
	webhookManager *webhook.Manager
	executorMap    map[storepb.Task_Type]Executor
	profile        *config.Profile
	licenseService *enterprise.LicenseService
}

// NewSchedulerV2 will create a new scheduler.
func NewSchedulerV2(
	store *store.Store,
	stateCfg *state.State,
	webhookManager *webhook.Manager,
	profile *config.Profile,
	licenseService *enterprise.LicenseService,
) *SchedulerV2 {
	return &SchedulerV2{
		store:          store,
		stateCfg:       stateCfg,
		webhookManager: webhookManager,
		profile:        profile,
		executorMap:    map[storepb.Task_Type]Executor{},
		licenseService: licenseService,
	}
}

// Register will register a task executor factory.
func (s *SchedulerV2) Register(taskType storepb.Task_Type, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType.String())
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType.String())
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
	environments, err := s.store.GetEnvironmentSetting(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to list environments")
	}

	var envs []string
	for _, environment := range environments.GetEnvironments() {
		policy, err := s.store.GetRolloutPolicy(ctx, environment.Id)
		if err != nil {
			return errors.Wrapf(err, "failed to get rollout policy for environment %s", environment.Id)
		}
		if policy.Automatic {
			envs = append(envs, environment.Id)
		}
	}
	taskIDs, err := s.store.ListTasksToAutoRollout(ctx, envs)
	if err != nil {
		return errors.Wrapf(err, "failed to list tasks with zero task run")
	}
	for _, taskID := range taskIDs {
		if err := s.scheduleAutoRolloutTask(ctx, taskID); err != nil {
			slog.Error("failed to schedule auto rollout task", log.BBError(err))
		}
	}

	return nil
}

func (s *SchedulerV2) scheduleAutoRolloutTask(ctx context.Context, taskUID int) error {
	task, err := s.store.GetTaskV2ByID(ctx, taskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task == nil {
		return nil
	}

	pipeline, err := s.store.GetPipelineV2ByID(ctx, task.PipelineID)
	if err != nil {
		return errors.Wrapf(err, "failed to get pipeline")
	}
	if pipeline == nil {
		return errors.Errorf("pipeline %v not found", task.PipelineID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
	if err != nil {
		return errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %v not found", pipeline.ProjectID)
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
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
		if issue.Status != storepb.Issue_OPEN {
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
			PlanUID: &plan.UID,
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

	create := &store.TaskRunMessage{
		CreatorID: common.SystemBotID,
		TaskUID:   task.ID,
	}
	if task.Payload.GetSheetId() != 0 {
		sheetUID := int(task.Payload.GetSheetId())
		create.SheetUID = &sheetUID
	}

	if err := s.store.CreatePendingTaskRuns(ctx, create); err != nil {
		return errors.Wrapf(err, "failed to create pending task runs")
	}

	if issue != nil {
		tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.Environment, taskUID)}
		if err := s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_PENDING, common.SystemBotID, ""); err != nil {
			slog.Warn("failed to create issue comment", "issueUID", issue.UID, log.BBError(err))
		}
	}
	s.webhookManager.CreateEvent(ctx, &webhook.Event{
		Actor:   s.store.GetSystemBotUser(ctx),
		Type:    common.EventTypeTaskRunStatusUpdate,
		Comment: "",
		Issue:   webhook.NewIssue(issue),
		Rollout: webhook.NewRollout(pipeline),
		Project: webhook.NewProject(project),
		TaskRunStatusUpdate: &webhook.EventTaskRunStatusUpdate{
			Title:  task.GetDatabaseName(),
			Status: storepb.TaskRun_PENDING.String(),
		},
	})

	return nil
}

func (s *SchedulerV2) schedulePendingTaskRuns(ctx context.Context) error {
	taskRuns, err := s.store.ListTaskRunsV2(ctx, &store.FindTaskRunMessage{
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

func (s *SchedulerV2) schedulePendingTaskRun(ctx context.Context, taskRun *store.TaskRunMessage) error {
	// here, we move pending taskruns to running taskruns which means they are ready to be executed.
	// pending taskruns remain pending if
	// 1. taskRun.RunAt not met.
	// 2. for versioned tasks, there are other versioned tasks on the same database with
	// a smaller version not finished yet. we need to wait for those first.
	task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
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

		maybeTaskID, err := s.store.FindBlockingTaskByVersion(ctx, task.PipelineID, task.InstanceID, *task.DatabaseName, schemaVersion)
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
		ID:        taskRun.ID,
		UpdaterID: common.SystemBotID,
		Status:    storepb.TaskRun_RUNNING,
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
		task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
		if err != nil {
			slog.Error("failed to get task", slog.Int("task id", taskRun.TaskUID), log.BBError(err))
			continue
		}
		if task.DatabaseName == nil {
			continue
		}

		databaseKey := getDatabaseKey(task.InstanceID, *task.DatabaseName)
		if isSequentialTask(task.Type) {
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

func (s *SchedulerV2) scheduleRunningTaskRun(ctx context.Context, taskRun *store.TaskRunMessage, minTaskIDForDatabase map[string]int) error {
	// Skip the task run if it is already executing.
	if _, ok := s.stateCfg.RunningTaskRuns.Load(taskRun.ID); ok {
		return nil
	}
	task, err := s.store.GetTaskV2ByID(ctx, taskRun.TaskUID)
	if err != nil {
		return errors.Wrapf(err, "failed to get task")
	}
	if task.DatabaseName != nil && isSequentialTask(task.Type) {
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

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return errors.Wrapf(err, "failed to get instance")
	}
	if instance.Deleted {
		return errors.Errorf("instance %v is deleted", task.InstanceID)
	}
	executor, ok := s.executorMap[task.Type]
	if !ok {
		return errors.Errorf("executor not found for task type: %v", task.Type)
	}

	// Check max connections per instance.
	maximumConnections := int(instance.Metadata.GetMaximumConnections())
	if maximumConnections <= 0 {
		maximumConnections = common.DefaultInstanceMaximumConnections
	}
	if s.stateCfg.InstanceOutstandingConnections.Increment(task.InstanceID, maximumConnections) {
		s.stateCfg.TaskRunSchedulerInfo.Store(taskRun.ID, &storepb.SchedulerInfo{
			ReportTime: timestamppb.Now(),
			WaitingCause: &storepb.SchedulerInfo_WaitingCause{
				Cause: &storepb.SchedulerInfo_WaitingCause_ConnectionLimit{
					ConnectionLimit: true,
				},
			},
		})
		return nil
	}
	// decrement the connection count if we return below.
	revertInstanceConnectionsIncrement := true
	defer func() {
		if revertInstanceConnectionsIncrement {
			s.stateCfg.InstanceOutstandingConnections.Decrement(task.InstanceID)
		}
	}()

	// Check max running task runs per rollout.
	pipeline, err := s.store.GetPipelineV2ByID(ctx, task.PipelineID)
	if err != nil {
		return errors.Wrapf(err, "failed to get pipeline")
	}
	if pipeline == nil {
		return errors.Errorf("pipeline %v not found", task.PipelineID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
	if err != nil {
		return errors.Wrapf(err, "failed to get project")
	}
	if project == nil {
		return errors.Errorf("project %v not found", pipeline.ProjectID)
	}

	rolloutID := strconv.Itoa(pipeline.ID)
	maxRunningTaskRunsPerRollout := int(project.Setting.GetParallelTasksPerRollout())
	if maxRunningTaskRunsPerRollout <= 0 {
		maxRunningTaskRunsPerRollout = defaultRolloutMaxRunningTaskRuns
	}
	if s.stateCfg.RolloutOutstandingTasks.Increment(rolloutID+"/"+task.InstanceID, maxRunningTaskRunsPerRollout) {
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
			s.stateCfg.RolloutOutstandingTasks.Decrement(rolloutID + "/" + task.InstanceID)
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
	revertInstanceConnectionsIncrement = false
	revertRolloutConnectionsIncrement = false
	go s.runTaskRunOnce(ctx, taskRun, task, executor)
	return nil
}

func (s *SchedulerV2) runTaskRunOnce(ctx context.Context, taskRun *store.TaskRunMessage, task *store.TaskMessage, executor Executor) {
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
		s.stateCfg.InstanceOutstandingConnections.Decrement(task.InstanceID)
		s.stateCfg.RolloutOutstandingTasks.Decrement(strconv.Itoa(task.PipelineID) + "/" + task.InstanceID)
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
			ID:        taskRun.ID,
			UpdaterID: common.SystemBotID,
			Status:    storepb.TaskRun_CANCELED,
			Code:      &code,
			Result:    &result,
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
			ID:        taskRun.ID,
			UpdaterID: common.SystemBotID,
			Status:    storepb.TaskRun_FAILED,
			Code:      &code,
			Result:    &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as FAILED",
				slog.Int("id", task.ID),
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
			tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.Environment, task.ID)}
			return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_FAILED, common.SystemBotID, "")
		}(); err != nil {
			slog.Warn("failed to create issue comment", log.BBError(err))
		}
		s.createActivityForTaskRunStatusUpdate(ctx, task, storepb.TaskRun_FAILED, taskRunResult.Detail)
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
			ID:        taskRun.ID,
			UpdaterID: common.SystemBotID,
			Status:    storepb.TaskRun_DONE,
			Code:      &code,
			Result:    &result,
		}
		if _, err := s.store.UpdateTaskRunStatus(ctx, taskRunStatusPatch); err != nil {
			slog.Error("Failed to mark task as DONE",
				slog.Int("id", task.ID),
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
			tasks := []string{common.FormatTask(issue.Project.ResourceID, task.PipelineID, task.Environment, task.ID)}
			return s.store.CreateIssueCommentTaskUpdateStatus(ctx, issue.UID, tasks, storepb.IssueCommentPayload_TaskUpdate_DONE, common.SystemBotID, "")
		}(); err != nil {
			slog.Warn("failed to create issue comment", log.BBError(err))
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

func getDatabaseKey(instanceID, databaseName string) string {
	return fmt.Sprintf("%s/%s", instanceID, databaseName)
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
	type pipelineEnvironment struct {
		pipelineUID int
		environment string
	}
	pipelineEnvironmentDoneConfirmed := map[pipelineEnvironment]bool{}

	for {
		select {
		case taskUID := <-s.stateCfg.TaskSkippedOrDoneChan:
			if err := func() error {
				task, err := s.store.GetTaskV2ByID(ctx, taskUID)
				if err != nil {
					return errors.Wrapf(err, "failed to get task")
				}
				if pipelineEnvironmentDoneConfirmed[pipelineEnvironment{pipelineUID: task.PipelineID, environment: task.Environment}] {
					return nil
				}

				environmentTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &task.PipelineID, Environment: &task.Environment})
				if err != nil {
					return errors.Wrapf(err, "failed to list tasks")
				}

				skippedOrDone := tasksSkippedOrDone(environmentTasks)
				if !skippedOrDone {
					return nil
				}

				pipelineEnvironmentDoneConfirmed[pipelineEnvironment{pipelineUID: task.PipelineID, environment: task.Environment}] = true

				plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{PipelineID: &task.PipelineID})
				if err != nil {
					return errors.Wrapf(err, "failed to get plan")
				}

				// Get environment order from plan deployment config or global settings
				// Some environments may have zero tasks. We need to filter them out.
				var environmentOrder []string
				if plan != nil && len(plan.Config.GetDeployment().GetEnvironments()) > 0 {
					environmentOrder = plan.Config.Deployment.GetEnvironments()
				} else {
					// Use global environment setting order
					environmentOrder, err = getAllEnvironmentIDs(ctx, s.store)
					if err != nil {
						return errors.Wrapf(err, "failed to list environments")
					}
				}

				allTasks, err := s.store.ListTasks(ctx, &store.TaskFind{PipelineID: &task.PipelineID})
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
				var pipelineDone bool
				for i, env := range filteredEnvironmentOrder {
					if env == currentEnvironment {
						if i < len(filteredEnvironmentOrder)-1 {
							nextEnvironment = filteredEnvironmentOrder[i+1]
						}
						if i == len(filteredEnvironmentOrder)-1 {
							pipelineDone = true
						}
						break
					}
				}

				pipeline, err := s.store.GetPipelineV2ByID(ctx, task.PipelineID)
				if err != nil {
					return errors.Wrapf(err, "failed to get pipeline")
				}
				if pipeline == nil {
					return errors.Errorf("pipeline %v not found", task.PipelineID)
				}
				project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
				if err != nil {
					return errors.Wrapf(err, "failed to get project")
				}
				if project == nil {
					return errors.Errorf("project %v not found", pipeline.ProjectID)
				}
				issueN, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
				if err != nil {
					return errors.Wrapf(err, "failed to get issue")
				}

				// every task in the stage terminated
				// create "stage ends" activity.
				s.webhookManager.CreateEvent(ctx, &webhook.Event{
					Actor:   s.store.GetSystemBotUser(ctx),
					Type:    common.EventTypeStageStatusUpdate,
					Comment: "",
					Issue:   webhook.NewIssue(issueN),
					Rollout: webhook.NewRollout(pipeline),
					Project: webhook.NewProject(project),
					StageStatusUpdate: &webhook.EventStageStatusUpdate{
						StageTitle: currentEnvironment,
						StageID:    currentEnvironment,
					},
				})

				if err := func() error {
					p := &storepb.IssueCommentPayload{
						Event: &storepb.IssueCommentPayload_StageEnd_{
							StageEnd: &storepb.IssueCommentPayload_StageEnd{
								Stage: common.FormatStage(project.ResourceID, task.PipelineID, task.Environment),
							},
						},
					}
					if issueN != nil {
						_, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
							IssueUID: issueN.UID,
							Payload:  p,
						}, common.SystemBotID)
						return err
					}
					return nil
				}(); err != nil {
					slog.Warn("failed to create issue comment", log.BBError(err))
				}

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
						Actor:   s.store.GetSystemBotUser(ctx),
						Type:    common.EventTypeIssueRolloutReady,
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

				// After all tasks in the pipeline are done, we will resolve the issue if the issue is auto-resolvable.
				if issueN != nil && project.Setting.AutoResolveIssue && pipelineDone {
					if err := func() error {
						// For those database data export issues, we don't resolve them automatically.
						if issueN.Type == storepb.Issue_DATABASE_EXPORT {
							return nil
						}

						newStatus := storepb.Issue_DONE
						updatedIssue, err := s.store.UpdateIssueV2(ctx, issueN.UID, &store.UpdateIssueMessage{Status: &newStatus})
						if err != nil {
							return errors.Wrapf(err, "failed to update issue status")
						}

						fromStatus := storepb.IssueCommentPayload_IssueUpdate_IssueStatus(storepb.IssueCommentPayload_IssueUpdate_IssueStatus_value[issueN.Status.String()])
						toStatus := storepb.IssueCommentPayload_IssueUpdate_IssueStatus(storepb.IssueCommentPayload_IssueUpdate_IssueStatus_value[updatedIssue.Status.String()])
						if _, err := s.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
							IssueUID: issueN.UID,
							Payload: &storepb.IssueCommentPayload{
								Event: &storepb.IssueCommentPayload_IssueUpdate_{
									IssueUpdate: &storepb.IssueCommentPayload_IssueUpdate{
										FromStatus: &fromStatus,
										ToStatus:   &toStatus,
									},
								},
							},
						}, common.SystemBotID); err != nil {
							return errors.Wrapf(err, "failed to create issue comment after changing the issue status")
						}

						s.webhookManager.CreateEvent(ctx, &webhook.Event{
							Actor:   s.store.GetSystemBotUser(ctx),
							Type:    common.EventTypeIssueStatusUpdate,
							Comment: "",
							Issue:   webhook.NewIssue(updatedIssue),
							Project: webhook.NewProject(project),
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

func (s *SchedulerV2) createActivityForTaskRunStatusUpdate(ctx context.Context, task *store.TaskMessage, newStatus storepb.TaskRun_Status, errDetail string) {
	if err := func() error {
		rollout, err := s.store.GetPipelineV2ByID(ctx, task.PipelineID)
		if err != nil {
			return errors.Wrapf(err, "failed to get pipeline")
		}
		if rollout == nil {
			return errors.Errorf("pipeline %v not found", task.PipelineID)
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &rollout.ProjectID})
		if err != nil {
			return errors.Wrapf(err, "failed to get project")
		}
		if project == nil {
			return errors.Errorf("project %v not found", rollout.ProjectID)
		}
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{
			PipelineID: &task.PipelineID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to get issue")
		}
		s.webhookManager.CreateEvent(ctx, &webhook.Event{
			Actor:   s.store.GetSystemBotUser(ctx),
			Type:    common.EventTypeTaskRunStatusUpdate,
			Comment: "",
			Issue:   webhook.NewIssue(issue),
			Rollout: webhook.NewRollout(rollout),
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

// isSequentialTask returns whether the task should be executed sequentially.
func isSequentialTask(taskType storepb.Task_Type) bool {
	//exhaustive:enforce
	switch taskType {
	case storepb.Task_DATABASE_SCHEMA_UPDATE,
		storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST,
		storepb.Task_DATABASE_SCHEMA_UPDATE_SDL:
		return true
	case storepb.Task_DATABASE_CREATE,
		storepb.Task_DATABASE_DATA_UPDATE,
		storepb.Task_DATABASE_EXPORT:
		return false
	case storepb.Task_TASK_TYPE_UNSPECIFIED:
		return false
	default:
		return false
	}
}

// getAllEnvironmentIDs returns all environment IDs from the store.
func getAllEnvironmentIDs(ctx context.Context, s *store.Store) ([]string, error) {
	environments, err := s.GetEnvironmentSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list environments")
	}
	var environmentIDs []string
	for _, e := range environments.GetEnvironments() {
		environmentIDs = append(environmentIDs, e.Id)
	}
	return environmentIDs, nil
}
