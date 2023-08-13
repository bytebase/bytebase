// Package taskrun implements a runner for executing tasks.
package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/runner/apprun"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

const (
	taskSchedulerInterval = time.Duration(1) * time.Second
)

var (
	taskCancellationImplemented = map[api.TaskType]bool{
		api.TaskDatabaseSchemaUpdateGhostSync: true,
	}
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		api.TaskPendingApproval: {api.TaskPending, api.TaskDone},
		api.TaskPending:         {api.TaskCanceled, api.TaskRunning, api.TaskPendingApproval, api.TaskDone},
		api.TaskRunning:         {api.TaskDone, api.TaskFailed, api.TaskCanceled},
		api.TaskDone:            {},
		api.TaskFailed:          {api.TaskPendingApproval, api.TaskDone},
		api.TaskCanceled:        {api.TaskPendingApproval, api.TaskDone},
	}
	allowedSkippedTaskStatus = map[api.TaskStatus]bool{
		api.TaskPendingApproval: true,
		api.TaskFailed:          true,
	}
	terminatedTaskStatus = map[api.TaskStatus]bool{
		api.TaskDone:     true,
		api.TaskCanceled: true,
		api.TaskFailed:   true,
	}
)

// NewScheduler creates a new task scheduler.
func NewScheduler(
	store *store.Store,
	applicationRunner *apprun.Runner,
	schemaSyncer *schemasync.Syncer,
	activityManager *activity.Manager,
	licenseService enterpriseAPI.LicenseService,
	stateCfg *state.State,
	profile config.Profile,
	metricReporter *metricreport.Reporter) *Scheduler {
	return &Scheduler{
		store:             store,
		applicationRunner: applicationRunner,
		schemaSyncer:      schemaSyncer,
		activityManager:   activityManager,
		licenseService:    licenseService,
		profile:           profile,
		stateCfg:          stateCfg,
		executorMap:       make(map[api.TaskType]Executor),
		metricReporter:    metricReporter,
	}
}

// Scheduler is the task scheduler.
type Scheduler struct {
	store             *store.Store
	applicationRunner *apprun.Runner
	schemaSyncer      *schemasync.Syncer
	activityManager   *activity.Manager
	licenseService    enterpriseAPI.LicenseService
	stateCfg          *state.State
	profile           config.Profile
	executorMap       map[api.TaskType]Executor
	metricReporter    *metricreport.Reporter
}

// Register will register a task executor factory.
func (s *Scheduler) Register(taskType api.TaskType, executorGetter Executor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorMap[taskType] = executorGetter
}

// Run will run the task scheduler.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Task scheduler started and will run every %v", taskSchedulerInterval))
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = errors.Errorf("%v", r)
						}
						log.Error("Task scheduler PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
					}
				}()

				ctx := context.Background()

				if err := s.scheduleAutoApprovedTasks(ctx); err != nil {
					log.Error("Failed to schedule auto approved tasks", zap.Error(err))
					return
				}

				if err := s.schedulePendingTasks(ctx); err != nil {
					log.Error("Failed to schedule tasks in the active stage",
						zap.Error(err),
					)
					return
				}

				// Inspect all running tasks
				// This fetches quite a bit info and may cause performance issue if we have many ongoing tasks
				// We may optimize this in the future since only some relationship info is needed by the executor
				taskStatusList := []api.TaskStatus{api.TaskRunning}
				tasks, err := s.store.ListTasks(ctx, &api.TaskFind{
					StatusList: &taskStatusList,
				})
				if err != nil {
					log.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}

				// For each database, we will only execute the earliest running task (minimal task ID) and hold up the rest of the running tasks.
				// Sort the taskList by ID first.
				// databaseRunningTasks is the mapping from database ID to the earliest task of this database.
				sort.Slice(tasks, func(i, j int) bool {
					return tasks[i].ID < tasks[j].ID
				})
				databaseRunningTasks := make(map[int]int)
				for _, task := range tasks {
					if task.DatabaseID == nil {
						continue
					}
					if _, ok := databaseRunningTasks[*task.DatabaseID]; ok {
						continue
					}
					databaseRunningTasks[*task.DatabaseID] = task.ID
				}

				for _, task := range tasks {
					// Skip task belongs to archived instances
					instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
					if err != nil {
						continue
					}
					if instance.Deleted {
						continue
					}
					// Skip the task that is already being executed.
					if _, ok := s.stateCfg.RunningTasks.Load(task.ID); ok {
						continue
					}
					// Skip the task that is not the earliest task of the database.
					// earliestTaskID is the one that should be executed.
					if task.DatabaseID != nil {
						if earliestTaskID, ok := databaseRunningTasks[*task.DatabaseID]; ok && earliestTaskID != task.ID {
							continue
						}
					}

					executor, ok := s.executorMap[task.Type]
					if !ok {
						log.Error("Skip running task with unknown type",
							zap.Int("id", task.ID),
							zap.String("name", task.Name),
							zap.String("type", string(task.Type)),
						)
						continue
					}

					s.stateCfg.Lock()
					if s.stateCfg.InstanceOutstandingConnections[task.InstanceID] >= state.InstanceMaximumConnectionNumber {
						s.stateCfg.Unlock()
						continue
					}
					s.stateCfg.InstanceOutstandingConnections[task.InstanceID]++
					s.stateCfg.Unlock()

					s.stateCfg.RunningTasks.Store(task.ID, true)
					go func(ctx context.Context, task *store.TaskMessage, executor Executor) {
						defer func() {
							s.stateCfg.RunningTasks.Delete(task.ID)
							s.stateCfg.RunningTasksCancel.Delete(task.ID)
							s.stateCfg.TaskProgress.Delete(task.ID)
							s.stateCfg.Lock()
							s.stateCfg.InstanceOutstandingConnections[task.InstanceID]--
							s.stateCfg.Unlock()
						}()

						driverCtx, cancel := context.WithCancel(ctx)
						s.stateCfg.RunningTasksCancel.Store(task.ID, cancel)

						done, result, err := RunExecutorOnce(ctx, driverCtx, executor, task)

						select {
						case <-driverCtx.Done():
							// task cancelled
							log.Debug("Task canceled",
								zap.Int("id", task.ID),
								zap.String("name", task.Name),
								zap.String("type", string(task.Type)),
							)
							return
						default:
						}

						if !done && err != nil {
							log.Debug("Encountered transient error running task, will retry",
								zap.Int("id", task.ID),
								zap.String("name", task.Name),
								zap.String("type", string(task.Type)),
								zap.Error(err),
							)
							return
						}
						if done && err != nil {
							log.Warn("Failed to run task",
								zap.Int("id", task.ID),
								zap.String("name", task.Name),
								zap.String("type", string(task.Type)),
								zap.Error(err),
							)
							bytes, marshalErr := json.Marshal(api.TaskRunResultPayload{
								Detail: err.Error(),
							})
							if marshalErr != nil {
								log.Error("Failed to marshal task run result",
									zap.Int("task_id", task.ID),
									zap.String("type", string(task.Type)),
									zap.Error(marshalErr),
								)
								return
							}
							code := common.ErrorCode(err)
							result := string(bytes)
							taskStatusPatch := &api.TaskStatusPatch{
								ID:        task.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskFailed,
								Code:      &code,
								Result:    &result,
							}
							if err := s.PatchTaskStatus(ctx, task, taskStatusPatch); err != nil {
								log.Error("Failed to mark task as FAILED",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.Error(err),
								)
							}
							return
						}
						if done && err == nil {
							bytes, err := json.Marshal(*result)
							if err != nil {
								log.Error("Failed to marshal task run result",
									zap.Int("task_id", task.ID),
									zap.String("type", string(task.Type)),
									zap.Error(err),
								)
								return
							}
							code := common.Ok
							result := string(bytes)
							taskStatusPatch := &api.TaskStatusPatch{
								ID:        task.ID,
								UpdaterID: api.SystemBotID,
								Status:    api.TaskDone,
								Code:      &code,
								Result:    &result,
							}
							if err := s.PatchTaskStatus(ctx, task, taskStatusPatch); err != nil {
								log.Error("Failed to mark task as DONE",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.Error(err),
								)
								return
							}
							return
						}
					}(ctx, task, executor)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// PatchTask patches the statement, earliest allowed time and rollbackEnabled for a task.
func (s *Scheduler) PatchTask(ctx context.Context, task *store.TaskMessage, taskPatch *api.TaskPatch, issue *store.IssueMessage) error {
	if taskPatch.SheetID != nil {
		if err := canUpdateTaskStatement(task); err != nil {
			return err
		}
		// If task created in UI mode in VCS Project, we should give it a new migration version.
		// https://linear.app/bytebase/issue/BYT-3311/the-executed-sql-is-not-expected
		isUICreatedInVCSProject := false
		if issue.Project.Workflow == api.VCSWorkflow {
			switch task.Type {
			case api.TaskDatabaseSchemaUpdate:
				var payload api.TaskDatabaseSchemaUpdatePayload
				if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
					return errors.Wrapf(err, "failed to unmarshal task payload")
				}
				if payload.VCSPushEvent == nil {
					isUICreatedInVCSProject = true
				}
			case api.TaskDatabaseSchemaUpdateSDL:
				var payload api.TaskDatabaseSchemaUpdateSDLPayload
				if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
					return errors.Wrapf(err, "failed to unmarshal task payload")
				}
				if payload.VCSPushEvent == nil {
					isUICreatedInVCSProject = true
				}
			case api.TaskDatabaseSchemaUpdateGhostSync:
				var payload api.TaskDatabaseSchemaUpdateGhostSyncPayload
				if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
					return errors.Wrapf(err, "failed to unmarshal task payload")
				}
				if payload.VCSPushEvent == nil {
					isUICreatedInVCSProject = true
				}
			case api.TaskDatabaseDataUpdate:
				var payload api.TaskDatabaseDataUpdatePayload
				if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
					return errors.Wrapf(err, "failed to unmarshal task payload")
				}
				if payload.VCSPushEvent == nil {
					isUICreatedInVCSProject = true
				}
			}
		}
		if issue.Project.Workflow == api.UIWorkflow || isUICreatedInVCSProject {
			schemaVersion := common.DefaultMigrationVersion()
			taskPatch.SchemaVersion = &schemaVersion
		}
	}

	// Reset because we are trying to build
	// the RollbackSheetID again and there could be previous runs.
	if taskPatch.RollbackEnabled != nil && *taskPatch.RollbackEnabled {
		empty := ""
		pending := api.RollbackSQLStatusPending
		taskPatch.RollbackSQLStatus = &pending
		taskPatch.RollbackSheetID = nil
		taskPatch.RollbackError = &empty
	}
	// if *taskPatch.RollbackEnabled == false, we don't reset
	// 1. they are meaningless anyway if rollback is disabled
	// 2. they will be reset when enabling
	// 3. we cancel the generation after writing to db. The rollback sql generation may finish and write to db, too.

	taskPatched, err := s.store.UpdateTaskV2(ctx, taskPatch)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}
	if issue.NeedAttention && issue.Project.Workflow == api.UIWorkflow {
		needAttention := false
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to try to patch issue assignee_need_attention after updating task statement").SetInternal(err)
		}
	}

	// enqueue or cancel after it's written to the database.
	if taskPatch.RollbackEnabled != nil {
		// Enqueue the rollback sql generation if the task done.
		if *taskPatch.RollbackEnabled && taskPatched.Status == api.TaskDone {
			s.stateCfg.RollbackGenerate.Store(taskPatched.ID, taskPatched)
		} else {
			// Cancel running rollback sql generation.
			if v, ok := s.stateCfg.RollbackCancel.Load(taskPatched.ID); ok {
				if cancel, ok := v.(context.CancelFunc); ok {
					cancel()
				}
			}
			// We don't erase the keys for RollbackCancel and RollbackGenerate here because they will eventually be erased by the rollback runner.
		}
	}

	// Trigger task checks.
	if taskPatch.SheetID != nil {
		dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
		if err != nil {
			return err
		}
		if dbSchema == nil {
			return errors.Errorf("database schema ID not found %v", task.DatabaseID)
		}
		// it's ok to fail.
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonSQLModified); err != nil {
			log.Error("failed to cancel external approval on SQL modified", zap.Int("issue_id", issue.UID), zap.Error(err))
		}
		s.stateCfg.IssueExternalApprovalRelayCancelChan <- issue.UID
		if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
			if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
				CreatorID: taskPatched.CreatorID,
				TaskID:    task.ID,
				Type:      api.TaskCheckGhostSync,
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger gh-ost dry run after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return err
		}

		if api.IsSQLReviewSupported(instance.Engine) {
			if err := s.triggerDatabaseStatementAdviseTask(ctx, taskPatched); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to trigger database statement advise task")).SetInternal(err)
			}
		}

		if api.IsStatementTypeCheckSupported(instance.Engine) {
			if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
				CreatorID: taskPatched.CreatorID,
				TaskID:    task.ID,
				Type:      api.TaskCheckDatabaseStatementType,
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger statement type check after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		if api.IsTaskCheckReportSupported(instance.Engine) && api.IsTaskCheckReportNeededForTaskType(task.Type) {
			if err := s.store.CreateTaskCheckRun(ctx,
				&store.TaskCheckRunMessage{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementAffectedRowsReport,
				},
				&store.TaskCheckRunMessage{
					CreatorID: taskPatched.CreatorID,
					TaskID:    task.ID,
					Type:      api.TaskCheckDatabaseStatementTypeReport,
				},
			); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger task report check after changing the task statement", zap.Int("task_id", task.ID), zap.String("task_name", task.Name), zap.Error(err))
			}
		}
	}

	if taskPatch.SheetID != nil {
		oldSheetID, err := utils.GetTaskSheetID(task.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to get old sheet ID")
		}
		newSheetID := *taskPatch.SheetID

		// create a task sheet update activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
			TaskID:     taskPatched.ID,
			OldSheetID: oldSheetID,
			NewSheetID: newSheetID,
			TaskName:   task.Name,
			IssueName:  issue.Title,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task sheet: %v", taskPatched.Name).SetInternal(err)
		}
		if _, err := s.activityManager.CreateActivity(ctx, &store.ActivityMessage{
			CreatorUID:   taskPatch.UpdaterID,
			ContainerUID: taskPatched.PipelineID,
			Type:         api.ActivityPipelineTaskStatementUpdate,
			Payload:      string(payload),
			Level:        api.ActivityInfo,
		}, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
		}
	}

	// Earliest allowed time update activity.
	if taskPatch.EarliestAllowedTs != nil {
		// create an activity
		payload, err := json.Marshal(api.ActivityPipelineTaskEarliestAllowedTimeUpdatePayload{
			TaskID:               taskPatched.ID,
			OldEarliestAllowedTs: task.EarliestAllowedTs,
			NewEarliestAllowedTs: taskPatched.EarliestAllowedTs,
			TaskName:             task.Name,
			IssueName:            issue.Title,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal earliest allowed time activity payload: %v", task.Name))
		}
		activityCreate := &store.ActivityMessage{
			CreatorUID:   taskPatch.UpdaterID,
			ContainerUID: taskPatched.PipelineID,
			Type:         api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:      string(payload),
			Level:        api.ActivityInfo,
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}
	}
	return nil
}

func (s *Scheduler) triggerDatabaseStatementAdviseTask(ctx context.Context, task *store.TaskMessage) error {
	dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
	if err != nil {
		return err
	}
	if dbSchema == nil {
		return errors.Errorf("database schema ID not found %v", task.DatabaseID)
	}

	if err := s.store.CreateTaskCheckRun(ctx, &store.TaskCheckRunMessage{
		CreatorID: api.SystemBotID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
	}); err != nil {
		// It's OK if we failed to trigger a check, just emit an error log
		log.Error("Failed to trigger statement advise task after changing task statement",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Error(err),
		)
	}

	return nil
}

// CanPrincipalChangeTaskStatus validates if the principal has the privilege to update task status, judging from the principal role and the environment policy.
func (s *Scheduler) CanPrincipalChangeTaskStatus(ctx context.Context, principalID int, task *store.TaskMessage, toStatus api.TaskStatus) (bool, error) {
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if principalID == task.CreatorID {
			return true, nil
		}
	}
	// the workspace owner and DBA roles can always change task status.
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
	}
	if user == nil {
		return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
	}
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}

	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to find issue")
	}
	if issue == nil {
		return false, common.Errorf(common.NotFound, "issue not found by pipeline ID: %d", task.PipelineID)
	}
	groupValue, err := s.getGroupValueForTask(ctx, issue, task)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get assignee group value for taskID %d", task.ID)
	}
	if groupValue == nil {
		return false, nil
	}
	// as the policy says, the project owner has the privilege to change task status.
	if *groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == principalID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *Scheduler) getGroupValueForTask(ctx context.Context, issue *store.IssueMessage, task *store.TaskMessage) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	stages, err := s.store.ListStageV2(ctx, task.PipelineID)
	if err != nil {
		return nil, err
	}
	for _, stage := range stages {
		if stage.ID == task.StageID {
			environmentID = stage.EnvironmentID
			break
		}
	}
	if environmentID == api.UnknownID {
		return nil, common.Errorf(common.NotFound, "failed to find environmentID by task.StageID %d", task.StageID)
	}

	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issue.Type {
			return &assigneeGroup.Value, nil
		}
	}
	return nil, nil
}

// ClearRunningTasks changes all RUNNING tasks and taskRuns to CANCELED.
// When there are running tasks and Bytebase server is shutdown, these task executors are stopped, but the tasks' status are still RUNNING.
// When Bytebase is restarted, the task scheduler will re-schedule those RUNNING tasks, which should be CANCELED instead.
// So we change their status to CANCELED before starting the scheduler.
// And corresponding taskRuns are also changed to CANCELED.
func (s *Scheduler) ClearRunningTasks(ctx context.Context) error {
	taskFind := &api.TaskFind{StatusList: &[]api.TaskStatus{api.TaskRunning}}
	runningTasks, err := s.store.ListTasks(ctx, taskFind)
	if err != nil {
		return errors.Wrap(err, "failed to get running tasks")
	}
	if len(runningTasks) > 0 {
		var taskIDs []int
		for _, task := range runningTasks {
			taskIDs = append(taskIDs, task.ID)
		}
		if err := s.store.BatchPatchTaskStatus(ctx, taskIDs, api.TaskCanceled, api.SystemBotID); err != nil {
			return errors.Wrapf(err, "failed to change task %v's status to %s", taskIDs, api.TaskCanceled)
		}
	}

	runningTaskRuns, err := s.store.ListTaskRun(ctx, &store.TaskRunFind{StatusList: &[]api.TaskRunStatus{api.TaskRunRunning}})
	if err != nil {
		return errors.Wrap(err, "failed to get running task runs")
	}
	if len(runningTaskRuns) > 0 {
		var taskRunIDs []int
		for _, taskRun := range runningTaskRuns {
			taskRunIDs = append(taskRunIDs, taskRun.ID)
		}
		if err := s.store.BatchPatchTaskRunStatus(ctx, taskRunIDs, api.TaskRunCanceled, api.SystemBotID); err != nil {
			return errors.Wrapf(err, "failed to change task run %v's status to %s", taskRunIDs, api.TaskRunCanceled)
		}
	}

	for _, task := range runningTasks {
		// If it's a backup task, we also change the corresponding backup's status to FAILED, because the task is canceled just now.
		if task.Type != api.TaskDatabaseBackup {
			continue
		}
		// TODO(d): batch patching backup status.
		var payload api.TaskDatabaseBackupPayload
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return errors.Wrapf(err, "failed to parse the payload of backup task %d", task.ID)
		}
		statusFailed := string(api.BackupStatusFailed)
		backup, err := s.store.UpdateBackupV2(ctx, &store.UpdateBackupMessage{
			UID:       payload.BackupID,
			UpdaterID: api.SystemBotID,
			Status:    &statusFailed,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to patch backup %d's status from %s to %s", payload.BackupID, api.BackupStatusPendingCreate, api.BackupStatusFailed)
		}
		log.Debug(fmt.Sprintf("Changed backup %d's status from %s to %s", payload.BackupID, api.BackupStatusPendingCreate, api.BackupStatusFailed))
		if err := removeLocalBackupFile(s.profile.DataDir, backup); err != nil {
			log.Warn(err.Error())
		}
	}
	return nil
}

var (
	allowedStatementUpdateTaskTypes = map[api.TaskType]bool{
		api.TaskDatabaseCreate:                true,
		api.TaskDatabaseSchemaUpdate:          true,
		api.TaskDatabaseSchemaUpdateSDL:       true,
		api.TaskDatabaseDataUpdate:            true,
		api.TaskDatabaseSchemaUpdateGhostSync: true,
	}
	allowedPatchStatementStatus = map[api.TaskStatus]bool{
		api.TaskPendingApproval: true,
		api.TaskFailed:          true,
	}
)

func canUpdateTaskStatement(task *store.TaskMessage) *echo.HTTPError {
	if ok := allowedStatementUpdateTaskTypes[task.Type]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update statement for task type %q", task.Type))
	}
	// Allow frontend to change the SQL statement of
	// 1. a PendingApproval task which hasn't started yet;
	// 2. a Failed task which can be retried.
	if !allowedPatchStatementStatus[task.Status] {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("cannot update task in %q status", task.Status))
	}
	return nil
}

// scheduleIfNeeded schedules the task if
//  2. it has no blocking tasks.
//  3. it has passed the earliest allowed time.
func (s *Scheduler) scheduleIfNeeded(ctx context.Context, task *store.TaskMessage) error {
	blocked, err := s.isTaskBlocked(ctx, task)
	if err != nil {
		return errors.Wrap(err, "failed to check if task is blocked")
	}
	if blocked {
		return nil
	}
	if task.EarliestAllowedTs != 0 && time.Now().Before(time.Unix(task.EarliestAllowedTs, 0)) {
		return nil
	}

	return s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Status:    api.TaskRunning,
	})
}

func (s *Scheduler) isTaskBlocked(ctx context.Context, task *store.TaskMessage) (bool, error) {
	for _, block := range task.BlockedBy {
		blockingTask, err := s.store.GetTaskV2ByID(ctx, block)
		if err != nil {
			return true, errors.Wrapf(err, "failed to fetch the blocking task, id: %d", block)
		}
		if blockingTask.Status != api.TaskDone {
			return true, nil
		}
	}
	return false, nil
}

// scheduleAutoApprovedTasks schedules tasks that are approved automatically.
func (s *Scheduler) scheduleAutoApprovedTasks(ctx context.Context) error {
	taskStatusList := []api.TaskStatus{api.TaskPendingApproval}
	taskList, err := s.store.ListTasks(ctx, &api.TaskFind{
		StatusList:      &taskStatusList,
		NoBlockingStage: true,
		NonRollbackTask: true,
	})
	if err != nil {
		return err
	}

	for _, task := range taskList {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
		if err != nil {
			return err
		}
		environmentID := instance.EnvironmentID
		// TODO(d): support creating database with environment override.
		if task.DatabaseID != nil {
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
			if err != nil {
				return err
			}
			environmentID = database.EffectiveEnvironmentID
		}
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &environmentID})
		if err != nil {
			return err
		}

		policy, err := s.store.GetPipelineApprovalPolicy(ctx, environment.UID)
		if err != nil {
			return errors.Wrapf(err, "failed to get approval policy for environment ID %d", environment.UID)
		}
		if policy.Value != api.PipelineApprovalValueManualNever {
			continue
		}

		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
		if err != nil {
			return err
		}
		if issue != nil {
			if issue.Status != api.IssueOpen {
				continue
			}
			approved, err := utils.CheckIssueApproved(issue)
			if err != nil {
				log.Warn("taskrun scheduler: failed to check if the issue is approved when scheduling auto-deployed tasks", zap.Int("taskID", task.ID), zap.Int("issueID", issue.UID), zap.Error(err))
				continue
			}
			if !approved {
				continue
			}
		}

		taskCheckRuns, err := s.store.ListTaskCheckRuns(ctx, &store.TaskCheckRunFind{TaskID: &task.ID})
		if err != nil {
			return err
		}
		ok, err := utils.PassAllCheck(task, api.TaskCheckStatusSuccess, taskCheckRuns, instance.Engine)
		if err != nil {
			return errors.Wrap(err, "failed to check if can auto-approve")
		}
		if ok {
			if err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
				ID:        task.ID,
				UpdaterID: api.SystemBotID,
				Status:    api.TaskPending,
			}); err != nil {
				return errors.Wrap(err, "failed to change task status")
			}
		}
	}
	return nil
}

// schedulePendingTasks tries to schedule pending tasks.
func (s *Scheduler) schedulePendingTasks(ctx context.Context) error {
	taskStatus := []api.TaskStatus{api.TaskPending}
	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{StatusList: &taskStatus})
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := s.scheduleIfNeeded(ctx, task); err != nil {
			return errors.Wrap(err, "failed to schedule task")
		}
	}
	return nil
}

// PatchTaskStatus patches a single task.
func (s *Scheduler) PatchTaskStatus(ctx context.Context, task *store.TaskMessage, taskStatusPatch *api.TaskStatusPatch) (err error) {
	defer func() {
		if err != nil {
			log.Error("Failed to change task status.",
				zap.Int("id", task.ID),
				zap.String("name", task.Name),
				zap.String("old_status", string(task.Status)),
				zap.String("new_status", string(taskStatusPatch.Status)),
				zap.Error(err))
		}
	}()

	if !isTaskStatusTransitionAllowed(task.Status, taskStatusPatch.Status) {
		return &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status]),
		}
	}

	if taskStatusPatch.Skipped != nil && *taskStatusPatch.Skipped && !allowedSkippedTaskStatus[task.Status] {
		return &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("cannot skip task whose status is %v", task.Status),
		}
	}

	if taskStatusPatch.Status == api.TaskCanceled {
		if !taskCancellationImplemented[task.Type] {
			return common.Errorf(common.NotImplemented, "Canceling task type %s is not supported", task.Type)
		}
		cancelAny, ok := s.stateCfg.RunningTasksCancel.Load(task.ID)
		if !ok {
			return errors.New("failed to cancel task")
		}
		cancel := cancelAny.(context.CancelFunc)
		cancel()
		result, err := json.Marshal(api.TaskRunResultPayload{
			Detail: "Task cancellation requested.",
		})
		if err != nil {
			return errors.Wrapf(err, "failed to marshal TaskRunResultPayload")
		}
		resultStr := string(result)
		taskStatusPatch.Result = &resultStr
	}

	taskPatched, err := s.store.UpdateTaskStatusV2(ctx, taskStatusPatch)
	if err != nil {
		return errors.Wrapf(err, "failed to change task %v(%v) status", task.ID, task.Name)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return errors.Wrapf(err, "failed to fetch containing issue after changing the task status: %v", task.Name)
	}

	// Create an activity
	if err := s.createTaskStatusUpdateActivity(ctx, task, taskStatusPatch, issue); err != nil {
		return err
	}

	// Cancel every task depending on the canceled task.
	if taskPatched.Status == api.TaskCanceled {
		if err := s.cancelDependingTasks(ctx, taskPatched); err != nil {
			return errors.Wrapf(err, "failed to cancel depending tasks for task %d", taskPatched.ID)
		}
	}

	if issue != nil {
		if err := s.onTaskStatusPatched(ctx, issue, taskPatched); err != nil {
			return err
		}
	}

	return nil
}

func isTaskStatusTransitionAllowed(fromStatus, toStatus api.TaskStatus) bool {
	for _, allowedStatus := range applicableTaskStatusTransition[fromStatus] {
		if allowedStatus == toStatus {
			return true
		}
	}
	return false
}

func (s *Scheduler) createTaskStatusUpdateActivity(ctx context.Context, task *store.TaskMessage, taskStatusPatch *api.TaskStatusPatch, issue *store.IssueMessage) error {
	var issueName string
	if issue != nil {
		issueName = issue.Title
	}
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskID:    task.ID,
		OldStatus: task.Status,
		NewStatus: taskStatusPatch.Status,
		IssueName: issueName,
		TaskName:  task.Name,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal activity after changing the task status: %v", task.Name)
	}

	level := api.ActivityInfo
	if taskStatusPatch.Status == api.TaskFailed {
		level = api.ActivityError
	}
	activityCreate := &store.ActivityMessage{
		CreatorUID:   taskStatusPatch.UpdaterID,
		ContainerUID: task.PipelineID,
		Type:         api.ActivityPipelineTaskStatusUpdate,
		Level:        level,
		Payload:      string(payload),
	}
	if taskStatusPatch.Comment != nil {
		activityCreate.Comment = *taskStatusPatch.Comment
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) cancelDependingTasks(ctx context.Context, task *store.TaskMessage) error {
	queue := []int{task.ID}
	seen := map[int]bool{task.ID: true}
	var idList []int
	dags, err := s.store.ListTaskDags(ctx, &store.TaskDAGFind{PipelineID: &task.PipelineID, StageID: &task.StageID})
	if err != nil {
		return err
	}
	for len(queue) != 0 {
		fromTaskID := queue[0]
		queue = queue[1:]
		for _, dag := range dags {
			if dag.FromTaskID != fromTaskID {
				continue
			}
			if seen[dag.ToTaskID] {
				return errors.Errorf("found a cycle in task dag, visit task %v twice", dag.ToTaskID)
			}
			seen[dag.ToTaskID] = true
			idList = append(idList, dag.ToTaskID)
			queue = append(queue, dag.ToTaskID)
		}
	}
	if err := s.store.BatchPatchTaskStatus(ctx, idList, api.TaskCanceled, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to change task %v's status to %s", idList, api.TaskCanceled)
	}
	return nil
}

// GetDefaultAssignee gets the default assignee for an issue.
func (s *Scheduler) GetDefaultAssignee(ctx context.Context, environmentID int, projectID int, issueType api.IssueType) (*store.UserMessage, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to GetPipelineApprovalPolicy for environmentID %d", environmentID)
	}
	if policy.Value == api.PipelineApprovalValueManualNever {
		// use SystemBot for auto approval tasks.
		systemBot, err := s.store.GetUserByID(ctx, api.SystemBotID)
		if err != nil {
			return nil, err
		}
		return systemBot, nil
	}

	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil || *groupValue == api.AssigneeGroupValueWorkspaceOwnerOrDBA {
		user, err := s.getAnyWorkspaceOwner(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a workspace owner or DBA")
		}
		return user, nil
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		projectOwner, err := s.getAnyProjectOwner(ctx, projectID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a project owner")
		}
		return projectOwner, nil
	}
	// never reached
	return nil, errors.New("invalid assigneeGroupValue")
}

// getAnyWorkspaceOwner finds a default assignee from the workspace owners.
func (s *Scheduler) getAnyWorkspaceOwner(ctx context.Context) (*store.UserMessage, error) {
	// There must be at least one non-systembot owner in the workspace.
	owner := api.Owner
	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{Role: &owner})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get role %v", api.Owner)
	}
	for _, user := range users {
		if user.ID == api.SystemBotID {
			continue
		}
		return user, nil
	}
	return nil, errors.New("failed to get a workspace owner or DBA")
}

// getAnyProjectOwner gets a default assignee from the project owners.
func (s *Scheduler) getAnyProjectOwner(ctx context.Context, projectID int) (*store.UserMessage, error) {
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
	if err != nil {
		return nil, err
	}
	// Find the project member that is not workspace owner or DBA first.
	for _, binding := range policy.Bindings {
		if binding.Role != api.Owner || len(binding.Members) == 0 {
			continue
		}
		for _, user := range binding.Members {
			if user.Role != api.Owner && user.Role != api.DBA {
				return user, nil
			}
		}
	}
	for _, binding := range policy.Bindings {
		if binding.Role == api.Owner && len(binding.Members) > 0 {
			return binding.Members[0], nil
		}
	}
	return nil, errors.New("failed to get a project owner")
}

// CanPrincipalBeAssignee checks if a principal could be the assignee of an issue, judging by the principal role and the environment policy.
func (s *Scheduler) CanPrincipalBeAssignee(ctx context.Context, principalID int, environmentID int, projectID int, issueType api.IssueType) (bool, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return false, err
	}
	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil || *groupValue == api.AssigneeGroupValueWorkspaceOwnerOrDBA {
		// no value is set, fallback to default.
		// the assignee group is the workspace owner or DBA.
		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
		}
		if user == nil {
			return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
		}
		if s.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
			user.Role = api.Owner
		}
		if user.Role == api.Owner || user.Role == api.DBA {
			return true, nil
		}
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		// the assignee group is the project owner.
		if s.licenseService.IsFeatureEnabled(api.FeatureRBAC) != nil {
			// nolint:nilerr
			return true, nil
		}
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", projectID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == principalID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// ChangeIssueStatus changes the status of an issue.
func (s *Scheduler) ChangeIssueStatus(ctx context.Context, issue *store.IssueMessage, newStatus api.IssueStatus, updaterID int, comment string) error {
	if issue.PipelineUID != nil {
		tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: issue.PipelineUID})
		if err != nil {
			return err
		}
		switch newStatus {
		case api.IssueOpen:
		case api.IssueDone:
			// Returns error if any of the tasks is not DONE.
			for _, task := range tasks {
				if task.Status != api.TaskDone {
					return &common.Error{Code: common.Conflict, Err: errors.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Title, task.Name)}
				}
			}
		case api.IssueCanceled:
			// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
			// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
			// keep those tasks in the same state before the issue was canceled.
			for _, task := range tasks {
				if task.Status == api.TaskRunning {
					if err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
						ID:        task.ID,
						UpdaterID: updaterID,
						Status:    api.TaskCanceled,
					}); err != nil {
						return errors.Wrapf(err, "failed to cancel issue: %v, failed to cancel task: %v", issue.Title, task.Name)
					}
				}
			}
		}
	}

	updateIssueMessage := &store.UpdateIssueMessage{Status: &newStatus}
	if newStatus != api.IssueOpen && issue.Project.Workflow == api.UIWorkflow {
		// for UI workflow, set assigneeNeedAttention to false if we are closing the issue.
		needAttention := false
		updateIssueMessage.NeedAttention = &needAttention
	}
	updatedIssue, err := s.store.UpdateIssueV2(ctx, issue.UID, updateIssueMessage, updaterID)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}

	// Cancel external approval, it's ok if we failed.
	if newStatus != api.IssueOpen {
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonIssueNotOpen); err != nil {
			log.Error("failed to cancel external approval on issue cancellation or completion", zap.Error(err))
		}
		s.stateCfg.IssueExternalApprovalRelayCancelChan <- issue.UID
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Title,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal activity after changing the issue status: %v", updatedIssue.Title)
	}

	activityCreate := &store.ActivityMessage{
		CreatorUID:   updaterID,
		ContainerUID: issue.UID,
		Type:         api.ActivityIssueStatusUpdate,
		Level:        api.ActivityInfo,
		Comment:      comment,
		Payload:      string(payload),
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: updatedIssue,
	}); err != nil {
		return errors.Wrapf(err, "failed to create activity after changing the issue status: %v", updatedIssue.Title)
	}

	return nil
}

func (s *Scheduler) onTaskStatusPatched(ctx context.Context, issue *store.IssueMessage, taskPatched *store.TaskMessage) error {
	s.metricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.TaskStatusMetricName,
		Value: 1,
		Labels: map[string]any{
			"type":  taskPatched.Type,
			"value": taskPatched.Status,
		},
	})

	stages, err := s.store.ListStageV2(ctx, taskPatched.PipelineID)
	if err != nil {
		return errors.Wrap(err, "failed to list stages")
	}
	var taskStage, nextStage *store.StageMessage
	for i, stage := range stages {
		if stage.ID == taskPatched.StageID {
			taskStage = stages[i]
			if i+1 < len(stages) {
				nextStage = stages[i+1]
			}
			break
		}
	}
	if taskStage == nil {
		return errors.New("failed to find corresponding stage of the task in the issue pipeline")
	}

	if !taskStage.Active && nextStage != nil {
		// Every task in this stage has finished, we are moving to the next stage.
		// The current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
		func() {
			environmentID := nextStage.EnvironmentID
			ok, err := s.CanPrincipalBeAssignee(ctx, issue.Assignee.ID, environmentID, issue.Project.UID, issue.Type)
			if err != nil {
				log.Error("failed to check if the current assignee still fits in the new assignee group", zap.Error(err))
				return
			}
			if !ok {
				// reassign the issue to a new assignee if the current one doesn't fit.
				assignee, err := s.GetDefaultAssignee(ctx, environmentID, issue.Project.UID, issue.Type)
				if err != nil {
					log.Error("failed to get a default assignee", zap.Error(err))
					return
				}
				if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{Assignee: assignee}, api.SystemBotID); err != nil {
					log.Error("failed to update the issue assignee", zap.Error(err))
					return
				}
			}
		}()
	}
	if !taskStage.Active && nextStage == nil {
		// Every task in the pipeline has finished.
		// Resolve the issue automatically for the user.
		if err := s.ChangeIssueStatus(ctx, issue, api.IssueDone, api.SystemBotID, ""); err != nil {
			log.Error("failed to change the issue status to done automatically after completing every task", zap.Error(err))
		}
	}

	tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &taskPatched.PipelineID, StageID: &taskPatched.StageID})
	if err != nil {
		return errors.Wrap(err, "failed to list tasks")
	}
	stageTaskHasPendingApproval := false
	stageTaskAllTerminated := true
	for _, task := range tasks {
		if task.Status == api.TaskPendingApproval {
			stageTaskHasPendingApproval = true
		}
		if !terminatedTaskStatus[task.Status] {
			stageTaskAllTerminated = false
		}
	}

	// every task in the stage completes
	// cancel external approval, it's ok if we failed.
	if !taskStage.Active {
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonNoTaskPendingApproval); err != nil {
			log.Error("failed to cancel external approval on stage tasks completion", zap.Int("issue_id", issue.UID), zap.Error(err))
		}
	}

	// every task in the stage terminated
	// create "stage ends" activity.
	if stageTaskAllTerminated {
		if err := func() error {
			createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
				StageID:               taskStage.ID,
				StageStatusUpdateType: api.StageStatusUpdateTypeEnd,
				IssueName:             issue.Title,
				StageName:             taskStage.Name,
			}
			bytes, err := json.Marshal(createActivityPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal ActivityPipelineStageStatusUpdate payload")
			}
			activityCreate := &store.ActivityMessage{
				CreatorUID:   api.SystemBotID,
				ContainerUID: *issue.PipelineUID,
				Type:         api.ActivityPipelineStageStatusUpdate,
				Level:        api.ActivityInfo,
				Payload:      string(bytes),
			}
			if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: issue,
			}); err != nil {
				return errors.Wrap(err, "failed to create activity")
			}
			return nil
		}(); err != nil {
			log.Error("failed to create ActivityPipelineStageStatusUpdate activity", zap.Error(err))
		}
	}

	// every task in the stage completes and this is not the last stage.
	// create "stage begins" activity.
	if !taskStage.Active && nextStage != nil {
		if err := func() error {
			createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
				StageID:               nextStage.ID,
				StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
				IssueName:             issue.Title,
				StageName:             nextStage.Name,
			}
			bytes, err := json.Marshal(createActivityPayload)
			if err != nil {
				return errors.Wrap(err, "failed to marshal ActivityPipelineStageStatusUpdate payload")
			}
			activityCreate := &store.ActivityMessage{
				CreatorUID:   api.SystemBotID,
				ContainerUID: *issue.PipelineUID,
				Type:         api.ActivityPipelineStageStatusUpdate,
				Level:        api.ActivityInfo,
				Payload:      string(bytes),
			}
			if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
				Issue: issue,
			}); err != nil {
				return errors.Wrap(err, "failed to create activity")
			}
			return nil
		}(); err != nil {
			log.Error("failed to create ActivityPipelineStageStatusUpdate activity", zap.Error(err))
		}
	}

	// there isn't a pendingApproval task
	// we need to set issue.AssigneeNeedAttention to false for UI workflow.
	if taskPatched.Status == api.TaskPending && issue.Project.Workflow == api.UIWorkflow && !stageTaskHasPendingApproval {
		needAttention := false
		if _, err := s.store.UpdateIssueV2(ctx, issue.UID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			log.Error("failed to patch issue assigneeNeedAttention after finding out that there isn't any pendingApproval task in the stage", zap.Error(err))
		}
	}

	return nil
}

// CanPrincipalChangeIssueStageTaskStatus returns whether the principal can change the task status.
func (s *Scheduler) CanPrincipalChangeIssueStageTaskStatus(ctx context.Context, user *store.UserMessage, issue *store.IssueMessage, stageEnvironmentID int, toStatus api.TaskStatus) (bool, error) {
	// the workspace owner and DBA roles can always change task status.
	if user.Role == api.Owner || user.Role == api.DBA {
		return true, nil
	}
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if user.ID == issue.Creator.ID {
			return true, nil
		}
	}
	groupValue, err := s.getGroupValueForIssueTypeEnvironment(ctx, issue.Type, stageEnvironmentID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get assignee group value for issueID %d", issue.UID)
	}
	// as the policy says, the project owner has the privilege to change task status.
	if groupValue == api.AssigneeGroupValueProjectOwner {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &issue.Project.UID})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project %d policy", issue.Project.UID)
		}
		for _, binding := range policy.Bindings {
			if binding.Role != api.Owner {
				continue
			}
			for _, member := range binding.Members {
				if member.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (s *Scheduler) getGroupValueForIssueTypeEnvironment(ctx context.Context, issueType api.IssueType, environmentID int) (api.AssigneeGroupValue, error) {
	defaultGroupValue := api.AssigneeGroupValueWorkspaceOwnerOrDBA
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return defaultGroupValue, errors.Wrapf(err, "failed to get pipeline approval policy by environmentID %d", environmentID)
	}
	if policy == nil {
		return defaultGroupValue, nil
	}

	for _, assigneeGroup := range policy.AssigneeGroupList {
		if assigneeGroup.IssueType == issueType {
			return assigneeGroup.Value, nil
		}
	}
	return defaultGroupValue, nil
}
