// Package taskrun implements a runner for executing tasks.
package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
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
	"github.com/bytebase/bytebase/backend/runner/apprun"
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

				if err := s.scheduleActiveStageToRunning(ctx); err != nil {
					log.Error("Failed to schedule tasks in the active stage",
						zap.Error(err),
					)
					return
				}

				// Inspect all running tasks
				taskStatusList := []api.TaskStatus{api.TaskRunning}
				taskFind := &api.TaskFind{
					StatusList: &taskStatusList,
				}
				// This fetches quite a bit info and may cause performance issue if we have many ongoing tasks
				// We may optimize this in the future since only some relationship info is needed by the executor
				taskList, err := s.store.FindTask(ctx, taskFind)
				if err != nil {
					log.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}

				// For each database, we will only execute the earliest running task (minimal task ID) and hold up the rest of the running tasks.
				// Sort the taskList by ID first.
				// databaseRunningTasks is the mapping from database ID to the earliest task of this database.
				sort.Slice(taskList, func(i, j int) bool {
					return taskList[i].ID < taskList[j].ID
				})
				databaseRunningTasks := make(map[int]int)
				for _, task := range taskList {
					if task.DatabaseID == nil {
						continue
					}
					if _, ok := databaseRunningTasks[*task.DatabaseID]; ok {
						continue
					}
					databaseRunningTasks[*task.DatabaseID] = task.ID
				}

				for _, task := range taskList {
					// Skip task belongs to archived instances
					if i := task.Instance; i == nil || i.RowStatus == api.Archived {
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
								ID:        task.ID,
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
								ID:        task.ID,
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

							issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
							if err != nil {
								log.Error("failed to getIssueByPipelineID", zap.Int("pipelineID", task.PipelineID), zap.Error(err))
								return
							}
							// The task has finished, and we may move to a new stage.
							// if the current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
							if issue != nil {
								composedPipeline, err := s.store.GetPipelineByID(ctx, issue.PipelineUID)
								if err != nil {
									return
								}
								if activeStage := utils.GetActiveStage(composedPipeline); activeStage != nil && activeStage.ID != task.StageID {
									environmentID := activeStage.EnvironmentID
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
			ID:        taskPatch.ID,
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
		if _, err := s.store.UpdateIssueV2(ctx, issue.ID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to try to patch issue assignee_need_attention after updating task statement").SetInternal(err)
		}
	}

	// Trigger task checks.
	if taskPatch.Statement != nil {
		dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
		if err != nil {
			return nil, err
		}
		if dbSchema == nil {
			return nil, errors.Errorf("database schema ID not found %v", task.DatabaseID)
		}
		// it's ok to fail.
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonSQLModified); err != nil {
			log.Error("failed to cancel external approval on SQL modified", zap.Int("issue_id", issue.ID), zap.Error(err))
		}
		if taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostSync {
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &store.TaskCheckRunCreate{
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
				Charset:   dbSchema.Metadata.CharacterSet,
				Collation: dbSchema.Metadata.Collation,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name))
			}
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &store.TaskCheckRunCreate{
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
				Charset:   dbSchema.Metadata.CharacterSet,
				Collation: dbSchema.Metadata.Collation,
			})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, errors.Wrapf(err, "failed to marshal check statement type payload: %v", task.Name))
			}
			if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &store.TaskCheckRunCreate{
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
	dbSchema, err := s.store.GetDBSchema(ctx, *task.DatabaseID)
	if err != nil {
		return err
	}
	if dbSchema == nil {
		return errors.Errorf("database schema ID not found %v", task.DatabaseID)
	}

	payload, err := json.Marshal(api.TaskCheckDatabaseStatementAdvisePayload{
		Statement: statement,
		DbType:    task.Database.Instance.Engine,
		Charset:   dbSchema.Metadata.CharacterSet,
		Collation: dbSchema.Metadata.Collation,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal statement advise payload: %v", task.Name)
	}

	if _, err := s.store.CreateTaskCheckRunIfNeeded(ctx, &store.TaskCheckRunCreate{
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
	composedPipeline, err := s.store.GetPipelineByID(ctx, issue.PipelineUID)
	if err != nil {
		return false, err
	}
	groupValue, err := s.getGroupValueForTask(ctx, issue, composedPipeline, task)
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

func (s *Scheduler) getGroupValueForTask(ctx context.Context, issue *store.IssueMessage, composedPipeline *api.Pipeline, task *api.Task) (*api.AssigneeGroupValue, error) {
	environmentID := api.UnknownID
	for _, stage := range composedPipeline.StageList {
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
	runningTasks, err := s.store.FindTask(ctx, taskFind)
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

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return false, err
	}
	return utils.PassAllCheck(task, api.TaskCheckStatusWarn, task.TaskCheckRunList, instance.Engine)
}

// scheduleIfNeeded schedules the task if
//  1. its required check does not contain error in the latest run.
//  2. it has no blocking tasks.
//  3. it has passed the earliest allowed time.
func (s *Scheduler) scheduleIfNeeded(ctx context.Context, task *api.Task) error {
	schedule, err := s.canSchedule(ctx, task)
	if err != nil {
		return err
	}
	if !schedule {
		return nil
	}

	if _, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Status:    api.TaskRunning,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) isTaskBlocked(ctx context.Context, task *api.Task) (bool, error) {
	for _, blockingTaskIDString := range task.BlockedBy {
		blockingTaskID, err := strconv.Atoi(blockingTaskIDString)
		if err != nil {
			return true, errors.Wrapf(err, "failed to convert id string to int, id string: %v", blockingTaskIDString)
		}
		blockingTask, err := s.store.GetTaskV2ByID(ctx, blockingTaskID)
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
	taskList, err := s.store.FindTask(ctx, taskFind)
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

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.Database.InstanceID})
		if err != nil {
			return err
		}
		ok, err := utils.PassAllCheck(task, api.TaskCheckStatusSuccess, task.TaskCheckRunList, instance.Engine)
		if err != nil {
			return errors.Wrap(err, "failed to check if can auto-approve")
		}
		if ok {
			if _, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
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

// scheduleActiveStageToRunning tries to schedule the tasks in the active stage.
func (s *Scheduler) scheduleActiveStageToRunning(ctx context.Context) error {
	pipelineList, err := s.store.ListActivePipelines(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve open pipelines")
	}
	for _, pipeline := range pipelineList {
		stage := utils.GetActiveStage(pipeline)
		if stage == nil {
			continue
		}
		for _, task := range stage.TaskList {
			if task.Status != api.TaskPending {
				continue
			}

			if err := s.scheduleIfNeeded(ctx, task); err != nil {
				return errors.Wrap(err, "failed to schedule task")
			}
		}
	}
	return nil
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

	if taskStatusPatch.Skipped != nil && *taskStatusPatch.Skipped && !allowedSkippedTaskStatus[task.Status] {
		return nil, &common.Error{
			Code: common.Invalid,
			Err:  errors.Errorf("cannot skip task whose status is %v", task.Status),
		}
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

	taskPatched, err := s.store.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to change task %v(%v) status", task.ID, task.Name)
	}

	// Most tasks belong to a pipeline which in turns belongs to an issue. The followup code
	// behaves differently depending on whether the task is wrapped in an issue.
	// TODO(tianzhou): Refactor the followup code into chained onTaskStatusChange hook.
	issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue after changing the task status: %v", task.Name)
	}
	var composedIssue *api.Issue
	// Not all pipelines belong to an issue, so it's OK if issue is not found.
	if issue == nil {
		log.Debug("Pipeline has no linking issue",
			zap.Int("pipelineID", task.PipelineID),
			zap.String("task", task.Name))
	} else {
		composedIssue, err = s.store.GetIssueByID(ctx, issue.UID)
		if err != nil {
			return nil, err
		}
	}

	// Create an activity
	if err := s.createTaskStatusUpdateActivity(ctx, task, taskStatusPatch, composedIssue); err != nil {
		return nil, err
	}

	// Cancel every task depending on the canceled task.
	if taskPatched.Status == api.TaskCanceled {
		if err := s.cancelDependingTasks(ctx, taskPatched); err != nil {
			return nil, errors.Wrapf(err, "failed to cancel depending tasks for task %d", taskPatched.ID)
		}
	}

	if composedIssue != nil {
		if err := s.onTaskPatched(ctx, composedIssue, taskPatched); err != nil {
			return nil, err
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
		dags, err := s.store.ListTaskDags(ctx, &store.TaskDAGFind{FromTaskID: &fromTaskID})
		if err != nil {
			return err
		}
		for _, dag := range dags {
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
		if !s.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
			user.Role = api.Owner
		}
		if user.Role == api.Owner || user.Role == api.DBA {
			return true, nil
		}
	} else if *groupValue == api.AssigneeGroupValueProjectOwner {
		// the assignee group is the project owner.
		if !s.licenseService.IsFeatureEnabled(api.FeatureRBAC) {
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
func (s *Scheduler) ChangeIssueStatus(ctx context.Context, issue *store.IssueMessage, newStatus api.IssueStatus, updaterID int, comment string) (*api.Issue, error) {
	composedPipeline, err := s.store.GetPipelineByID(ctx, issue.PipelineUID)
	if err != nil {
		return nil, err
	}
	switch newStatus {
	case api.IssueOpen:
	case api.IssueDone:
		// Returns error if any of the tasks is not DONE.
		for _, stage := range composedPipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status != api.TaskDone {
					return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("failed to resolve issue: %v, task %v has not finished", issue.Title, task.Name)}
				}
			}
		}
	case api.IssueCanceled:
		// If we want to cancel the issue, we find the current running tasks, mark each of them CANCELED.
		// We keep PENDING and FAILED tasks as is since the issue maybe reopened later, and it's better to
		// keep those tasks in the same state before the issue was canceled.
		for _, stage := range composedPipeline.StageList {
			for _, task := range stage.TaskList {
				if task.Status == api.TaskRunning {
					if _, err := s.PatchTaskStatus(ctx, task, &api.TaskStatusPatch{
						ID:        task.ID,
						UpdaterID: updaterID,
						Status:    api.TaskCanceled,
					}); err != nil {
						return nil, errors.Wrapf(err, "failed to cancel issue: %v, failed to cancel task: %v", issue.Title, task.Name)
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
		return nil, errors.Wrapf(err, "failed to update issue %q's status", issue.Title)
	}
	composedIssue, err := s.store.GetIssueByID(ctx, issue.UID)
	if err != nil {
		return nil, err
	}

	// Cancel external approval, it's ok if we failed.
	if newStatus != api.IssueOpen {
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.UID, api.ExternalApprovalCancelReasonIssueNotOpen); err != nil {
			log.Error("failed to cancel external approval on issue cancellation or completion", zap.Error(err))
		}
	}

	payload, err := json.Marshal(api.ActivityIssueStatusUpdatePayload{
		OldStatus: issue.Status,
		NewStatus: newStatus,
		IssueName: updatedIssue.Title,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal activity after changing the issue status: %v", issue.Title)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   updaterID,
		ContainerID: issue.UID,
		Type:        api.ActivityIssueStatusUpdate,
		Level:       api.ActivityInfo,
		Comment:     comment,
		Payload:     string(payload),
	}

	if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
		Issue: composedIssue,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to create activity after changing the issue status: %v", issue.Title)
	}

	return composedIssue, nil
}

func (s *Scheduler) onTaskPatched(ctx context.Context, issue *api.Issue, taskPatched *api.Task) error {
	foundStage := false
	stageTaskHasPendingApproval := false
	stageTaskAllTerminated := true
	stageTaskAllDone := true
	var stageIndex int
	for i, stage := range issue.Pipeline.StageList {
		if stage.ID == taskPatched.StageID {
			foundStage = true
			stageIndex = i
			for _, task := range stage.TaskList {
				if task.Status == api.TaskPendingApproval {
					stageTaskHasPendingApproval = true
				}
				if task.Status != api.TaskDone {
					stageTaskAllDone = false
				}
				if !terminatedTaskStatus[task.Status] {
					stageTaskAllTerminated = false
				}
			}
			break
		}
	}
	if !foundStage {
		return errors.New("failed to find corresponding stage of the task in the issue pipeline")
	}

	// every task in the stage completes
	// cancel external approval, it's ok if we failed.
	if stageTaskAllDone {
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, api.ExternalApprovalCancelReasonNoTaskPendingApproval); err != nil {
			log.Error("failed to cancel external approval on stage tasks completion", zap.Int("issue_id", issue.ID), zap.Error(err))
		}
	}

	// every task in the stage terminated
	// create "stage ends" activity.
	if stageTaskAllTerminated {
		stage := issue.Pipeline.StageList[stageIndex]
		createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
			StageID:               stage.ID,
			StageStatusUpdateType: api.StageStatusUpdateTypeEnd,
			IssueName:             issue.Name,
			StageName:             stage.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return errors.Wrap(err, "failed to create ActivityPipelineStageStatusUpdate activity")
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.PipelineID,
			Type:        api.ActivityPipelineStageStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return errors.Wrap(err, "failed to create ActivityPipelineStageStatusUpdate activity")
		}
	}

	// every task in the stage completes and this is not the last stage.
	// create "stage begins" activity.
	if stageTaskAllDone && stageIndex+1 < len(issue.Pipeline.StageList) {
		stage := issue.Pipeline.StageList[stageIndex+1]
		createActivityPayload := api.ActivityPipelineStageStatusUpdatePayload{
			StageID:               stage.ID,
			StageStatusUpdateType: api.StageStatusUpdateTypeBegin,
			IssueName:             issue.Name,
			StageName:             stage.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return errors.Wrap(err, "failed to create ActivityPipelineStageStatusUpdate activity")
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   api.SystemBotID,
			ContainerID: issue.PipelineID,
			Type:        api.ActivityPipelineStageStatusUpdate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		if _, err := s.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{
			Issue: issue,
		}); err != nil {
			return errors.Wrap(err, "failed to create ActivityPipelineStageStatusUpdate activity")
		}
	}

	// there isn't a pendingApproval task
	// we need to set issue.AssigneeNeedAttention to false for UI workflow.
	if taskPatched.Status == api.TaskPending && issue.Project.WorkflowType == api.UIWorkflow && !stageTaskHasPendingApproval {
		needAttention := false
		if _, err := s.store.UpdateIssueV2(ctx, issue.ID, &store.UpdateIssueMessage{NeedAttention: &needAttention}, api.SystemBotID); err != nil {
			return errors.Wrapf(err, "failed to patch issue assigneeNeedAttention after finding out that there isn't any pendingApproval task in the stage")
		}
	}

	return nil
}
