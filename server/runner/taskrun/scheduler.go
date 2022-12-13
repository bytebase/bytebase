// Package taskrun implements a runner for executing tasks.
package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/server/runner/apprun"
	"github.com/bytebase/bytebase/server/runner/schemasync"
	"github.com/bytebase/bytebase/server/utils"
	"github.com/bytebase/bytebase/store"
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
)

// NewScheduler creates a new task scheduler.
func NewScheduler(store *store.Store, applicationRunner *apprun.Runner, schemaSyncer *schemasync.Syncer, activityManager *activity.Manager, licenseService enterpriseAPI.LicenseService, stateCfg *state.State, profile config.Profile) *Scheduler {
	return &Scheduler{
		store:             store,
		applicationRunner: applicationRunner,
		schemaSyncer:      schemaSyncer,
		activityManager:   activityManager,
		licenseService:    licenseService,
		profile:           profile,
		stateCfg:          stateCfg,
		executorMap:       make(map[api.TaskType]Executor),
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

				taskList, err := s.schedulePendingTasksToRunning(ctx)
				if err != nil {
					log.Error("Failed to schedule PENDING tasks to RUNNING",
						zap.Error(err),
					)
					return
				}

				for _, task := range taskList {
					// Skip the task that is already being executed.
					if _, ok := s.stateCfg.RunningTasks.Load(task.ID); ok {
						continue
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

					s.stateCfg.RunningTasks.Store(task.ID, true)
					go func(ctx context.Context, task *api.Task, executor Executor) {
						defer func() {
							s.stateCfg.RunningTasks.Delete(task.ID)
							s.stateCfg.RunningTasksCancel.Delete(task.ID)
							s.stateCfg.TaskProgress.Delete(task.ID)
							s.stateCfg.Lock()
							s.stateCfg.InstanceOutstandingConnections[task.InstanceID]--
							s.stateCfg.Unlock()
						}()

						executorCtx, cancel := context.WithCancel(ctx)
						s.stateCfg.RunningTasksCancel.Store(task.ID, cancel)

						done, result, err := RunExecutorOnce(executorCtx, executor, task)

						select {
						case <-executorCtx.Done():
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
								IDList:    []int{task.ID},
								UpdaterID: api.SystemBotID,
								Status:    api.TaskFailed,
								Code:      &code,
								Result:    &result,
							}
							if _, err := s.PatchTaskStatus(ctx, task, taskStatusPatch); err != nil {
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
								IDList:    []int{task.ID},
								UpdaterID: api.SystemBotID,
								Status:    api.TaskDone,
								Code:      &code,
								Result:    &result,
							}
							if _, err := s.PatchTaskStatus(ctx, task, taskStatusPatch); err != nil {
								log.Error("Failed to mark task as DONE",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.Error(err),
								)
								return
							}

							issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
							if err != nil {
								log.Error("failed to getIssueByPipelineID", zap.Int("pipelineID", task.PipelineID), zap.Error(err))
								return
							}
							// The task has finished, and we may move to a new stage.
							// if the current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
							if issue != nil {
								if stage := utils.GetActiveStage(issue.Pipeline); stage != nil && stage.ID != task.StageID {
									environmentID := stage.EnvironmentID
									ok, err := s.CanPrincipalBeAssignee(ctx, issue.AssigneeID, environmentID, issue.ProjectID, issue.Type)
									if err != nil {
										log.Error("failed to check if the current assignee still fits in the new assignee group", zap.Error(err))
										return
									}
									if !ok {
										// reassign the issue to a new assignee if the current one doesn't fit.
										assigneeID, err := s.GetDefaultAssigneeID(ctx, environmentID, issue.ProjectID, issue.Type)
										if err != nil {
											log.Error("failed to get a default assignee", zap.Error(err))
											return
										}
										patch := &api.IssuePatch{
											ID:         issue.ID,
											UpdaterID:  api.SystemBotID,
											AssigneeID: &assigneeID,
										}
										if _, err := s.store.PatchIssue(ctx, patch); err != nil {
											log.Error("failed to update the issue assignee", zap.Any("issuePatch", patch))
											return
										}
									}
								}
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

// PatchTaskStatement patches the statement and earliest allowed time for a patch.
func (s *Scheduler) PatchTaskStatement(ctx context.Context, task *api.Task, taskPatch *api.TaskPatch, issue *api.Issue) (*api.Task, error) {
	if taskPatch.Statement != nil {
		if err := canUpdateTaskStatement(task); err != nil {
			return nil, err
		}
		if issue.Project.WorkflowType == api.UIWorkflow {
			schemaVersion := common.DefaultMigrationVersion()
			taskPatch.SchemaVersion = &schemaVersion
		}
	}

	taskPatched, err := s.store.PatchTask(ctx, taskPatch)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\"", task.Name)).SetInternal(err)
	}
	// Update statement or earliest allowed time, dismiss stale approvals and transfer the status to PendingApproval for Pending tasks.
	// TODO(d): revisit this as task pending is only a short-period of time.
	if taskPatched.Status == api.TaskPending {
		t, err := s.PatchTaskStatus(ctx, taskPatched, &api.TaskStatusPatch{
			IDList:    []int{taskPatch.ID},
			UpdaterID: taskPatch.UpdaterID,
			Status:    api.TaskPendingApproval,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change task status to PendingApproval after updating task: %v", taskPatched.Name)).SetInternal(err)
		}
		taskPatched = t
	}
	if issue.AssigneeNeedAttention && issue.Project.WorkflowType == api.UIWorkflow {
		needAttention := false
		patch := &api.IssuePatch{
			ID:                    issue.ID,
			UpdaterID:             api.SystemBotID,
			AssigneeNeedAttention: &needAttention,
		}
		if _, err := s.store.PatchIssue(ctx, patch); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to try to patch issue assignee_need_attention after updating task statement").SetInternal(err)
		}
	}

	// Trigger task checks.
	if taskPatch.Statement != nil {
		// it's ok to fail.
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonSQLModified); err != nil {
			log.Error("failed to cancel external approval on SQL modified", zap.Int("issue_id", issue.ID), zap.Error(err))
		}
		if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
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

		if api.IsSyntaxCheckSupported(task.Database.Instance.Engine) {
			payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
				Statement: *taskPatch.Statement,
				DbType:    task.Database.Instance.Engine,
				Charset:   taskPatched.Database.CharacterSet,
				Collation: taskPatched.Database.Collation,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name))
			}
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
				CreatorID: api.SystemBotID,
				TaskID:    task.ID,
				Type:      api.TaskCheckDatabaseStatementSyntax,
				Payload:   string(payload),
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger syntax check after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}

		if api.IsSQLReviewSupported(task.Database.Instance.Engine) {
			if err := s.triggerDatabaseStatementAdviseTask(ctx, *taskPatch.Statement, taskPatched); err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to trigger database statement advise task")).SetInternal(err)
			}
		}

		if api.IsStatementTypeCheckSupported(task.Instance.Engine) {
			payload, err := json.Marshal(api.TaskCheckDatabaseStatementTypePayload{
				Statement: *taskPatch.Statement,
				DbType:    task.Instance.Engine,
				Charset:   task.Database.CharacterSet,
				Collation: task.Database.Collation,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal check statement type payload: %v", task.Name))
			}
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
				CreatorID: api.SystemBotID,
				TaskID:    task.ID,
				Type:      api.TaskCheckDatabaseStatementType,
				Payload:   string(payload),
			}); err != nil {
				// It's OK if we failed to trigger a check, just emit an error log
				log.Error("Failed to trigger statement type check after changing the task statement",
					zap.Int("task_id", task.ID),
					zap.String("task_name", task.Name),
					zap.Error(err),
				)
			}
		}
	}

	// Update statement activity.
	if taskPatch.Statement != nil {
		oldStatement, err := utils.GetTaskStatement(task)
		if err != nil {
			return nil, err
		}
		newStatement := *taskPatch.Statement

		// create a task statement update activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatementUpdatePayload{
			TaskID:       taskPatched.ID,
			OldStatement: oldStatement,
			NewStatement: newStatement,
			TaskName:     task.Name,
			IssueName:    issue.Name,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity after updating task statement: %v", taskPatched.Name).SetInternal(err)
		}
		if _, err := s.activityManager.CreateActivity(ctx, &api.ActivityCreate{
			CreatorID:   taskPatched.CreatorID,
			ContainerID: taskPatched.PipelineID,
			Type:        api.ActivityPipelineTaskStatementUpdate,
			Payload:     string(payload),
			Level:       api.ActivityInfo,
		}, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task statement: %v", taskPatched.Name)).SetInternal(err)
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
			IssueName:            issue.Name,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal earliest allowed time activity payload: %v", task.Name))
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   taskPatched.CreatorID,
			ContainerID: taskPatched.PipelineID,
			Type:        api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
			Payload:     string(payload),
			Level:       api.ActivityInfo,
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating task earliest allowed time: %v", taskPatched.Name)).SetInternal(err)
		}
	}
	return taskPatched, nil
}

func (s *Scheduler) triggerDatabaseStatementAdviseTask(ctx context.Context, statement string, task *api.Task) error {
	policyID, err := s.store.GetSQLReviewPolicyIDByEnvID(ctx, task.Instance.EnvironmentID)
	if err != nil {
		// It's OK if we failed to find the SQL review policy, just emit an error log
		log.Error("Failed to found SQL review policy id for task",
			zap.Int("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Int("environment_id", task.Instance.EnvironmentID),
			zap.Error(err),
		)
		return nil
	}

	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    task.Database.Instance.Engine,
		Charset:   task.Database.CharacterSet,
		Collation: task.Database.Collation,
		PolicyID:  policyID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}

	if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &api.TaskCheckRunCreate{
		CreatorID: api.SystemBotID,
		TaskID:    task.ID,
		Type:      api.TaskCheckDatabaseStatementAdvise,
		Payload:   string(payload),
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
func (s *Scheduler) CanPrincipalChangeTaskStatus(ctx context.Context, principalID int, task *api.Task, toStatus api.TaskStatus) (bool, error) {
	// The creator can cancel task.
	if toStatus == api.TaskCanceled {
		if principalID == task.CreatorID {
			return true, nil
		}
	}
	// the workspace owner and DBA roles can always change task status.
	principal, err := s.store.GetPrincipalByID(ctx, principalID)
	if err != nil {
		return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
	}
	if principal == nil {
		return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
	}
	if principal.Role == api.Owner || principal.Role == api.DBA {
		return true, nil
	}

	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
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
		member, err := s.store.GetProjectMember(ctx, &api.ProjectMemberFind{
			ProjectID:   &issue.ProjectID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project member by projectID %d, principalID %d", issue.ProjectID, principalID)
		}
		if member != nil && member.Role == string(api.Owner) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Scheduler) getGroupValueForTask(ctx context.Context, issue *api.Issue, task *api.Task) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	for _, stage := range issue.Pipeline.StageList {
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

// ClearRunningTasks changes all RUNNING tasks to CANCELED.
// When there are running tasks and Bytebase server is shutdown, these task executors are stopped, but the tasks' status are still RUNNING.
// When Bytebase is restarted, the task scheduler will re-schedule those RUNNING tasks, which should be CANCELED instead.
// So we change their status to CANCELED before starting the scheduler.
func (s *Scheduler) ClearRunningTasks(ctx context.Context) error {
	taskFind := &api.TaskFind{StatusList: &[]api.TaskStatus{api.TaskRunning}}
	runningTasks, err := s.store.FindTask(ctx, taskFind, false /* returnOnErr */)
	if err != nil {
		return errors.Wrap(err, "failed to get running tasks")
	}
	if len(runningTasks) == 0 {
		return nil
	}

	var taskIDs []int
	for _, task := range runningTasks {
		taskIDs = append(taskIDs, task.ID)
	}
	if err := s.store.BatchPatchTaskStatus(ctx, taskIDs, api.TaskCanceled, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to change task %v's status to %s", taskIDs, api.TaskCanceled)
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
		backup, err := s.store.PatchBackup(ctx, &api.BackupPatch{
			ID:        payload.BackupID,
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

func canUpdateTaskStatement(task *api.Task) *echo.HTTPError {
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

func (s *Scheduler) passAllCheck(ctx context.Context, task *api.Task, allowedStatus api.TaskCheckStatus) (bool, error) {
	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunDone, api.TaskCheckRunFailed}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &task.ID,
		StatusList: &statusList,
	}
	taskCheckRunList, err := s.store.FindTaskCheckRun(ctx, taskCheckRunFind)
	if err != nil {
		return false, err
	}

	// schema update, data update and gh-ost sync task have required task check.
	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseSchemaUpdateSDL || task.Type == api.TaskDatabaseDataUpdate || task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := passCheck(taskCheckRunList, api.TaskCheckDatabaseConnect, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		pass, err = passCheck(taskCheckRunList, api.TaskCheckInstanceMigrationSchema, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return false, err
		}
		if instance == nil {
			return false, errors.Errorf("instance ID not found %v", task.InstanceID)
		}

		if api.IsSyntaxCheckSupported(instance.Engine) {
			ok, err := passCheck(taskCheckRunList, api.TaskCheckDatabaseStatementSyntax, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		if api.IsSQLReviewSupported(instance.Engine) {
			ok, err := passCheck(taskCheckRunList, api.TaskCheckDatabaseStatementAdvise, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}

		if instance.Engine == db.Postgres {
			ok, err := passCheck(taskCheckRunList, api.TaskCheckDatabaseStatementType, allowedStatus)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		ok, err := passCheck(taskCheckRunList, api.TaskCheckGhostSync, allowedStatus)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// Returns true only if the task check run result is at least the minimum required level.
// For PendingApproval->Pending transitions, the minimum level is SUCCESS.
// For Pending->Running transitions, the minimum level is WARN.
// TODO(dragonly): refactor arguments.
func passCheck(taskCheckRunList []*api.TaskCheckRun, checkType api.TaskCheckType, allowedStatus api.TaskCheckStatus) (bool, error) {
	var lastRun *api.TaskCheckRun
	for _, run := range taskCheckRunList {
		if checkType != run.Type {
			continue
		}
		if lastRun == nil || lastRun.ID < run.ID {
			lastRun = run
		}
	}

	if lastRun == nil || lastRun.Status == api.TaskCheckRunFailed {
		return false, nil
	}
	checkResult := &api.TaskCheckRunResultPayload{}
	if err := json.Unmarshal([]byte(lastRun.Result), checkResult); err != nil {
		return false, err
	}
	for _, result := range checkResult.ResultList {
		if result.Status.LessThan(allowedStatus) {
			return false, nil
		}
	}

	return true, nil
}

// canSchedule returns whether a task can be scheduled.
func (s *Scheduler) canSchedule(ctx context.Context, task *api.Task) (bool, error) {
	blocked, err := s.isTaskBlocked(ctx, task)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if task is blocked")
	}
	if blocked {
		return false, nil
	}

	if task.EarliestAllowedTs != 0 && time.Now().Before(time.Unix(task.EarliestAllowedTs, 0)) {
		return false, nil
	}

	return s.passAllCheck(ctx, task, api.TaskCheckStatusWarn)
}

func (s *Scheduler) isTaskBlocked(ctx context.Context, task *api.Task) (bool, error) {
	for _, blockingTaskIDString := range task.BlockedBy {
		blockingTaskID, err := strconv.Atoi(blockingTaskIDString)
		if err != nil {
			return true, errors.Wrapf(err, "failed to convert id string to int, id string: %v", blockingTaskIDString)
		}
		blockingTask, err := s.store.GetTaskByID(ctx, blockingTaskID)
		if err != nil {
			return true, errors.Wrapf(err, "failed to fetch the blocking task, id: %v", blockingTaskID)
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
	taskFind := &api.TaskFind{
		StatusList: &taskStatusList,
	}
	taskList, err := s.store.FindTask(ctx, taskFind, false)
	if err != nil {
		return err
	}

	for _, task := range taskList {
		policy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
		if err != nil {
			return errors.Wrapf(err, "failed to get approval policy for environment ID %d", task.Instance.EnvironmentID)
		}
		if policy.Value != api.PipelineApprovalValueManualNever {
			continue
		}

		ok, err := s.passAllCheck(ctx, task, api.TaskCheckStatusSuccess)
		if err != nil {
			return errors.Wrap(err, "failed to check if can auto-approve")
		}
		if ok {
			if _, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
				IDList:    []int{task.ID},
				UpdaterID: api.SystemBotID,
				Status:    api.TaskPending,
			}); err != nil {
				return errors.Wrap(err, "failed to change task status")
			}
		}
	}
	return nil
}

// schedulePendingTasks tries to schedule the PENDING tasks to RUNNING.
func (s *Scheduler) schedulePendingTasksToRunning(ctx context.Context) ([]*api.Task, error) {
	taskStatusList := []api.TaskStatus{api.TaskPending}
	taskFind := &api.TaskFind{
		StatusList: &taskStatusList,
	}
	taskList, err := s.store.FindTask(ctx, taskFind, false)
	if err != nil {
		return nil, err
	}
	var runningTasks []*api.Task
	for _, task := range taskList {
		// Skip task belongs to archived instances
		if i := task.Instance; i == nil || i.RowStatus == api.Archived {
			continue
		}
		schedule, err := s.canSchedule(ctx, task)
		if err != nil {
			log.Error("failed to check task schedulable", zap.Int("taskID", task.ID), zap.Error(err))
			continue
		}
		if !schedule {
			continue
		}

		// Increment the maximum outstanding tasks per instance.
		s.stateCfg.Lock()
		if s.stateCfg.InstanceOutstandingConnections[task.InstanceID] >= state.InstanceMaximumConnectionNumber {
			s.stateCfg.Unlock()
			continue
		}
		s.stateCfg.InstanceOutstandingConnections[task.InstanceID]++
		s.stateCfg.Unlock()

		updatedTask, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
			IDList:    []int{task.ID},
			UpdaterID: api.SystemBotID,
			Status:    api.TaskRunning,
		})
		if err != nil {
			log.Error("failed to patch task to RUNNING", zap.Int("taskID", task.ID), zap.Error(err))
			continue
		}

		runningTasks = append(runningTasks, updatedTask)
	}
	return runningTasks, nil
}

// PatchTaskStatus patches a single task.
func (s *Scheduler) PatchTaskStatus(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
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
		return nil, &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status]),
		}
	}

	if len(taskStatusPatch.IDList) != 1 {
		return nil, errors.Errorf("expect to patch 1 task, get %d", len(taskStatusPatch.IDList))
	}

	if taskStatusPatch.Status == api.TaskCanceled {
		if !taskCancellationImplemented[task.Type] {
			return nil, common.Errorf(common.NotImplemented, "Canceling task type %s is not supported", task.Type)
		}
		cancelAny, ok := s.stateCfg.RunningTasksCancel.Load(task.ID)
		cancel := cancelAny.(context.CancelFunc)
		if !ok {
			return nil, errors.New("failed to cancel task")
		}
		cancel()
		result, err := json.Marshal(api.TaskRunResultPayload{
			Detail: "Task cancellation requested.",
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal TaskRunResultPayload")
		}
		resultStr := string(result)
		taskStatusPatch.Result = &resultStr
	}

	taskPatchedList, err := s.store.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to change task %v(%v) status", task.ID, task.Name)
	}
	taskPatched := taskPatchedList[0]

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue after changing the task status: %v", task.Name)
	}
	// Not all pipelines belong to an issue, so it's OK if issue is not found.
	if issue == nil {
		log.Debug("Pipeline has no linking issue",
			zap.Int("pipelineID", task.PipelineID),
			zap.String("task", task.Name))
	}

	// Create an activity
	if err := s.createTaskStatusUpdateActivity(ctx, task, taskStatusPatch, issue); err != nil {
		return nil, err
	}

	// Cancel every task depending on the canceled task.
	if taskPatched.Status == api.TaskCanceled {
		if err := s.cancelDependingTasks(ctx, taskPatched); err != nil {
			return nil, errors.Wrapf(err, "failed to cancel depending tasks for task %d", taskPatched.ID)
		}
	}

	// If every task in the stage completes, it means that we are moving into a new stage. We need
	// 1. cancel external approval.
	// 2. for UI workflow, set issue.AssigneeNeedAttention to false.
	if taskPatched.Status == api.TaskDone && issue != nil {
		foundStage := false
		stageTaskAllDone := true
		for _, stage := range issue.Pipeline.StageList {
			if stage.ID == taskPatched.StageID {
				foundStage = true
				for _, task := range stage.TaskList {
					if task.Status != api.TaskDone {
						stageTaskAllDone = false
						break
					}
				}
				break
			}
		}
		// every task in the stage completes
		if foundStage && stageTaskAllDone {
			// Cancel external approval, it's ok if we failed.
			if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonNoTaskPendingApproval); err != nil {
				log.Error("failed to cancel external approval on stage tasks completion", zap.Int("issue_id", issue.ID), zap.Error(err))
			}

			if issue.Project.WorkflowType == api.UIWorkflow {
				needAttention := false
				patch := &api.IssuePatch{
					ID:                    issue.ID,
					UpdaterID:             api.SystemBotID,
					AssigneeNeedAttention: &needAttention,
				}
				if _, err := s.store.PatchIssue(ctx, patch); err != nil {
					return nil, errors.Wrapf(err, "failed to patch issue assigneeNeedAttention after completing the whole stage, issuePatch: %+v", patch)
				}
			}
		}
	}

	// If every task in the pipeline completes, and the assignee is system bot:
	// Case 1: If the task is associated with an issue, then we mark the issue (including the pipeline) as DONE.
	// Case 2: If the task is NOT associated with an issue, then we mark the pipeline as DONE.
	if taskPatched.Status == api.TaskDone && (issue == nil || issue.AssigneeID == api.SystemBotID) {
		pipeline, err := s.store.GetPipelineByID(ctx, taskPatched.PipelineID)
		if err != nil {
			return nil, errors.Errorf("failed to fetch pipeline/issue as DONE after completing task %v", taskPatched.Name)
		}
		if pipeline == nil {
			return nil, errors.Errorf("pipeline not found for ID %v", taskPatched.PipelineID)
		}
		if areAllTasksDone(pipeline) {
			if issue == nil {
				// System-generated tasks such as backup tasks don't have corresponding issues.
				status := api.PipelineDone
				pipelinePatch := &api.PipelinePatch{
					ID:        pipeline.ID,
					UpdaterID: taskStatusPatch.UpdaterID,
					Status:    &status,
				}
				if _, err := s.store.PatchPipeline(ctx, pipelinePatch); err != nil {
					return nil, errors.Wrapf(err, "failed to mark pipeline %v as DONE after completing task %v", pipeline.Name, taskPatched.Name)
				}
			} else {
				issue.Pipeline = pipeline
				if _, err := s.ChangeIssueStatus(ctx, issue, api.IssueDone, taskStatusPatch.UpdaterID, ""); err != nil {
					return nil, errors.Wrapf(err, "failed to mark issue %v as DONE after completing task %v", issue.Name, taskPatched.Name)
				}
			}
		}
	}

	return taskPatched, nil
}

func isTaskStatusTransitionAllowed(fromStatus, toStatus api.TaskStatus) bool {
	for _, allowedStatus := range applicableTaskStatusTransition[fromStatus] {
		if allowedStatus == toStatus {
			return true
		}
	}
	return false
}

func (s *Scheduler) createTaskStatusUpdateActivity(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch, issue *api.Issue) error {
	var issueName string
	if issue != nil {
		issueName = issue.Name
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
	activityCreate := &api.ActivityCreate{
		CreatorID:   taskStatusPatch.UpdaterID,
		ContainerID: task.PipelineID,
		Type:        api.ActivityPipelineTaskStatusUpdate,
		Level:       level,
		Payload:     string(payload),
	}
	if taskStatusPatch.Comment != nil {
		activityCreate.Comment = *taskStatusPatch.Comment
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) cancelDependingTasks(ctx context.Context, task *api.Task) error {
	queue := []int{task.ID}
	seen := map[int]bool{task.ID: true}
	var idList []int
	for len(queue) != 0 {
		fromTaskID := queue[0]
		queue = queue[1:]
		dagList, err := s.store.FindTaskDAGList(ctx, &api.TaskDAGFind{FromTaskID: &fromTaskID})
		if err != nil {
			return err
		}
		for _, dag := range dagList {
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

func areAllTasksDone(pipeline *api.Pipeline) bool {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status != api.TaskDone {
				return false
			}
		}
	}
	return true
}

// GetDefaultAssigneeID gets the default assignee for an issue.
func (s *Scheduler) GetDefaultAssigneeID(ctx context.Context, environmentID int, projectID int, issueType api.IssueType) (int, error) {
	policy, err := s.store.GetPipelineApprovalPolicy(ctx, environmentID)
	if err != nil {
		return api.UnknownID, errors.Wrapf(err, "failed to GetPipelineApprovalPolicy for environmentID %d", environmentID)
	}
	if policy.Value == api.PipelineApprovalValueManualNever {
		// use SystemBot for auto approval tasks.
		return api.SystemBotID, nil
	}

	var groupValue *api.AssigneeGroupValue
	for i, group := range policy.AssigneeGroupList {
		if group.IssueType == issueType {
			groupValue = &policy.AssigneeGroupList[i].Value
			break
		}
	}
	if groupValue == nil || *groupValue == api.AssigneeGroupValueWorkspaceOwnerOrDBA {
		member, err := s.getAnyWorkspaceOwnerOrDBA(ctx)
		if err != nil {
			return api.UnknownID, errors.Wrap(err, "failed to get a workspace owner or DBA")
		}
		return member.PrincipalID, nil
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		projectMember, err := s.getAnyProjectOwner(ctx, projectID)
		if err != nil {
			return api.UnknownID, errors.Wrap(err, "failed to get a project owner")
		}
		return projectMember.PrincipalID, nil
	}
	// never reached
	return api.UnknownID, errors.New("invalid assigneeGroupValue")
}

// getAnyFromWorkspaceOwnerOrDBA finds a default assignee from the workspace owners or DBAs.
func (s *Scheduler) getAnyWorkspaceOwnerOrDBA(ctx context.Context) (*api.Member, error) {
	for _, role := range []api.Role{api.Owner, api.DBA} {
		memberList, err := s.store.FindMember(ctx, &api.MemberFind{
			Role: &role,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get role %v", role)
		}
		if len(memberList) > 0 {
			return memberList[0], nil
		}
	}
	return nil, errors.New("failed to get a workspace owner or DBA")
}

// getAnyProjectOwner gets a default assignee from the project owners.
func (s *Scheduler) getAnyProjectOwner(ctx context.Context, projectID int) (*api.ProjectMember, error) {
	role := api.Owner
	find := &api.ProjectMemberFind{
		ProjectID: &projectID,
		Role:      &role,
	}
	projectMemberList, err := s.store.FindProjectMember(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to FindProjectMember with ProjectMemberFind %+v", find)
	}
	if len(projectMemberList) > 0 {
		return projectMemberList[0], nil
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
		principal, err := s.store.GetPrincipalByID(ctx, principalID)
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get principal by ID %d", principalID)
		}
		if principal == nil {
			return false, common.Errorf(common.NotFound, "principal not found by ID %d", principalID)
		}
		if !s.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
			principal.Role = api.Owner
		}
		if principal.Role == api.Owner || principal.Role == api.DBA {
			return true, nil
		}
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		// the assignee group is the project owner.
		member, err := s.store.GetProjectMember(ctx, &api.ProjectMemberFind{
			ProjectID:   &projectID,
			PrincipalID: &principalID,
		})
		if err != nil {
			return false, common.Wrapf(err, common.Internal, "failed to get project member by projectID %d, principalID %d", projectID, principalID)
		}
		if member == nil {
			return false, common.Errorf(common.NotFound, "project member not found by projectID %d, principalID %d", projectID, principalID)
		}
		if !s.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
			member.Role = string(api.Owner)
		}
		if member.Role == string(api.Owner) {
			return true, nil
		}
	}
	return false, nil
}

// ChangeIssueStatus changes the status of an issue.
func (s *Scheduler) ChangeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updaterID int, comment string) (*api.Issue, error) {
	var pipelineStatus api.PipelineStatus
	switch newStatus {
	case api.IssueOpen:
		pipelineStatus = api.PipelineOpen
	case api.IssueDone:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Name, task.Name)}
				}
			}
		}
		pipelineStatus = api.PipelineDone
	case api.IssueCanceled:
		// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
		// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
		// keep those tasks in the same state before the issue was canceled.
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status == api.TaskRunning {
					if _, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
						IDList:    []int{task.ID},
						UpdaterID: updaterID,
						Status:    api.TaskCanceled,
					}); err != nil {
						return nil, errors.Wrapf(err, "failed to cancel issue: %v, failed to cancel task: %v", issue.Name, task.Name)
					}
				}
			}
		}
		pipelineStatus = api.PipelineCanceled
	}

	pipelinePatch := &api.PipelinePatch{
		ID:        issue.PipelineID,
		UpdaterID: updaterID,
		Status:    &pipelineStatus,
	}
	if _, err := s.store.PatchPipeline(ctx, pipelinePatch); err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status, failed to update pipeline status with patch %+v", issue.Name, pipelinePatch)
	}

	issuePatch := &api.IssuePatch{
		ID:        issue.ID,
		UpdaterID: updaterID,
		Status:    &newStatus,
	}
	if newStatus != api.IssueOpen && issue.Project.WorkflowType == api.UIWorkflow {
		// for UI workflow, set assigneeNeedAttention to false if we are closing the issue.
		needAttention := false
		issuePatch.AssigneeNeedAttention = &needAttention
	}
	updatedIssue, err := s.store.PatchIssue(ctx, issuePatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update issue %q's status with patch %v", issue.Name, issuePatch)
	}

	// Cancel external approval, it's ok if we failed.
	if newStatus != api.IssueOpen {
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonIssueNotOpen); err != nil {
			log.Error("failed to cancel external approval on issue cancellation or completion", zap.Error(err))
		}
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Name,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal activity after changing the issue status: %v", issue.Name)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: issue.ID,
		Type:        api.ActivityIssueStatusUpdate,
		Level:       api.ActivityInfo,
		Comment:     comment,
		Payload:     string(payload),
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: updatedIssue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create activity after changing the issue status: %v", issue.Name)
	}

	return updatedIssue, nil
}
