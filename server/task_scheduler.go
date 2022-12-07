package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/store"
)

const (
	taskSchedulerInterval = time.Duration(1) * time.Second
)

// NewTaskScheduler creates a new task scheduler.
func NewTaskScheduler(server *Server, store *store.Store, applicationRunner *ApplicationRunner, schemaSyncer *SchemaSyncer, activityManager *activity.Manager, licenseService enterpriseAPI.LicenseService, profile config.Profile) *TaskScheduler {
	return &TaskScheduler{
		server:            server,
		store:             store,
		applicationRunner: applicationRunner,
		schemaSyncer:      schemaSyncer,
		activityManager:   activityManager,
		licenseService:    licenseService,
		profile:           profile,
		executorMap:       make(map[api.TaskType]TaskExecutor),
	}
}

// TaskScheduler is the task scheduler.
type TaskScheduler struct {
	server            *Server
	store             *store.Store
	applicationRunner *ApplicationRunner
	schemaSyncer      *SchemaSyncer
	activityManager   *activity.Manager
	licenseService    enterpriseAPI.LicenseService
	profile           config.Profile
	executorMap       map[api.TaskType]TaskExecutor

	runningTasks       sync.Map // map[taskID]bool
	runningTasksCancel sync.Map // map[taskID]context.CancelFunc
	taskProgress       sync.Map // map[taskID]api.Progress
	sharedTaskState    sync.Map // map[taskID]interface{}
}

// Run will run the task scheduler.
func (s *TaskScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
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

				// Inspect all open pipelines and schedule the next PENDING task if applicable
				pipelineStatus := api.PipelineOpen
				pipelineFind := &api.PipelineFind{
					Status: &pipelineStatus,
				}
				pipelineList, err := s.store.FindPipeline(ctx, pipelineFind, false)
				if err != nil {
					log.Error("Failed to retrieve open pipelines", zap.Error(err))
					return
				}
				for _, pipeline := range pipelineList {
					if err := s.ScheduleActiveStage(ctx, pipeline); err != nil {
						log.Error("Failed to schedule tasks in the active stage",
							zap.Int("pipeline_id", pipeline.ID),
							zap.Error(err),
						)
					}
				}

				// Inspect all running tasks
				taskStatusList := []api.TaskStatus{api.TaskRunning}
				taskFind := &api.TaskFind{
					StatusList: &taskStatusList,
				}
				// This fetches quite a bit info and may cause performance issue if we have many ongoing tasks
				// We may optimize this in the future since only some relationship info is needed by the executor
				taskList, err := s.store.FindTask(ctx, taskFind, false)
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
					if _, ok := s.runningTasks.Load(task.ID); ok {
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

					s.runningTasks.Store(task.ID, true)
					go func(ctx context.Context, task *api.Task, executor TaskExecutor) {
						defer func() {
							s.runningTasks.Delete(task.ID)
							s.runningTasksCancel.Delete(task.ID)
							s.taskProgress.Delete(task.ID)
						}()

						executorCtx, cancel := context.WithCancel(ctx)
						s.runningTasksCancel.Store(task.ID, cancel)

						done, result, err := RunTaskExecutorOnce(executorCtx, executor, task)

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
							if _, err := s.patchTaskStatus(ctx, task, taskStatusPatch); err != nil {
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
							taskPatched, err := s.patchTaskStatus(ctx, task, taskStatusPatch)
							if err != nil {
								log.Error("Failed to mark task as DONE",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.Error(err),
								)
								return
							}

							issue, err := s.store.GetIssueByPipelineID(ctx, taskPatched.PipelineID)
							if err != nil {
								log.Error("failed to getIssueByPipelineID", zap.Int("pipelineID", taskPatched.PipelineID), zap.Error(err))
								return
							}
							// The task has finished, and we may move to a new stage.
							// if the current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
							if issue != nil {
								if stage := getActiveStage(issue.Pipeline.StageList); stage != nil && stage.ID != taskPatched.StageID {
									environmentID := stage.EnvironmentID
									ok, err := s.canPrincipalBeAssignee(ctx, issue.AssigneeID, environmentID, issue.ProjectID, issue.Type)
									if err != nil {
										log.Error("failed to check if the current assignee still fits in the new assignee group", zap.Error(err))
										return
									}
									if !ok {
										// reassign the issue to a new assignee if the current one doesn't fit.
										assigneeID, err := s.getDefaultAssigneeID(ctx, environmentID, issue.ProjectID, issue.Type)
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

// Register will register a task executor factory.
func (s *TaskScheduler) Register(taskType api.TaskType, executorGetter TaskExecutor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorMap[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorMap[taskType] = executorGetter
}

// ClearRunningTasks changes all RUNNING tasks to CANCELED.
// When there are running tasks and Bytebase server is shutdown, these task executors are stopped, but the tasks' status are still RUNNING.
// When Bytebase is restarted, the task scheduler will re-schedule those RUNNING tasks, which should be CANCELED instead.
// So we change their status to CANCELED before starting the scheduler.
func (s *TaskScheduler) ClearRunningTasks(ctx context.Context) error {
	taskFind := &api.TaskFind{StatusList: &[]api.TaskStatus{api.TaskRunning}}
	runningTasks, err := s.store.FindTask(ctx, taskFind, false)
	if err != nil {
		return errors.Wrap(err, "failed to get running tasks")
	}
	for _, task := range runningTasks {
		if _, err := s.store.PatchTaskStatus(ctx, &api.TaskStatusPatch{
			IDList:    []int{task.ID},
			UpdaterID: api.SystemBotID,
			Status:    api.TaskCanceled,
		}); err != nil {
			return errors.Wrapf(err, "failed to change task %d's status to %s", task.ID, api.TaskFailed)
		}
		log.Debug(fmt.Sprintf("Changed task %d's status from RUNNING to %s", task.ID, api.TaskFailed))
		// If it's a backup task, we also change the corresponding backup's status to FAILED, because the task is canceled just now.
		if task.Type == api.TaskDatabaseBackup {
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
	}
	return nil
}

func (s *TaskScheduler) passAllCheck(ctx context.Context, task *api.Task, allowedStatus api.TaskCheckStatus) (bool, error) {
	// schema update, data update and gh-ost sync task have required task check.
	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseSchemaUpdateSDL || task.Type == api.TaskDatabaseDataUpdate || task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := s.passCheck(ctx, task, api.TaskCheckDatabaseConnect, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		pass, err = s.passCheck(ctx, task, api.TaskCheckInstanceMigrationSchema, allowedStatus)
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
			pass, err = s.passCheck(ctx, task, api.TaskCheckDatabaseStatementSyntax, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}

		if api.IsSQLReviewSupported(instance.Engine) {
			pass, err = s.passCheck(ctx, task, api.TaskCheckDatabaseStatementAdvise, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}

		if instance.Engine == db.Postgres {
			pass, err = s.passCheck(ctx, task, api.TaskCheckDatabaseStatementType, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := s.passCheck(ctx, task, api.TaskCheckGhostSync, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}
	}

	return true, nil
}

// Returns true only if the task check run result is at least the minimum required level.
// For PendingApproval->Pending transitions, the minimum level is SUCCESS.
// For Pending->Running transitions, the minimum level is WARN.
// TODO(dragonly): refactor arguments.
func (s *TaskScheduler) passCheck(ctx context.Context, task *api.Task, checkType api.TaskCheckType, allowedStatus api.TaskCheckStatus) (bool, error) {
	statusList := []api.TaskCheckRunStatus{api.TaskCheckRunDone, api.TaskCheckRunFailed}
	taskCheckRunFind := &api.TaskCheckRunFind{
		TaskID:     &task.ID,
		Type:       &checkType,
		StatusList: &statusList,
		Latest:     true,
	}

	taskCheckRunList, err := s.store.FindTaskCheckRun(ctx, taskCheckRunFind)
	if err != nil {
		return false, err
	}

	if len(taskCheckRunList) == 0 || taskCheckRunList[0].Status == api.TaskCheckRunFailed {
		return false, nil
	}

	checkResult := &api.TaskCheckRunResultPayload{}
	if err := json.Unmarshal([]byte(taskCheckRunList[0].Result), checkResult); err != nil {
		return false, err
	}
	for _, result := range checkResult.ResultList {
		if result.Status.LessThan(allowedStatus) {
			return false, nil
		}
	}

	return true, nil
}

// auto transit PendingApproval to Pending if all required task checks pass.
func (s *TaskScheduler) canAutoApprove(ctx context.Context, task *api.Task) (bool, error) {
	return s.passAllCheck(ctx, task, api.TaskCheckStatusSuccess)
}

func (s *TaskScheduler) canSchedule(ctx context.Context, task *api.Task) (bool, error) {
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

// ScheduleIfNeeded schedules the task if
//  1. its required check does not contain error in the latest run.
//  2. it has no blocking tasks.
//  3. it has passed the earliest allowed time.
func (s *TaskScheduler) ScheduleIfNeeded(ctx context.Context, task *api.Task) (*api.Task, error) {
	schedule, err := s.canSchedule(ctx, task)
	if err != nil {
		return nil, err
	}
	if !schedule {
		return task, nil
	}

	updatedTask, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
		IDList:    []int{task.ID},
		UpdaterID: api.SystemBotID,
		Status:    api.TaskRunning,
	})
	if err != nil {
		return nil, err
	}

	return updatedTask, nil
}

func (s *TaskScheduler) isTaskBlocked(ctx context.Context, task *api.Task) (bool, error) {
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

func getActiveStage(stageList []*api.Stage) *api.Stage {
	for _, stage := range stageList {
		for _, task := range stage.TaskList {
			if task.Status != api.TaskDone {
				return stage
			}
		}
	}
	return nil
}

// ScheduleActiveStage tries to schedule the tasks in the active stage.
func (s *TaskScheduler) ScheduleActiveStage(ctx context.Context, pipeline *api.Pipeline) error {
	stage := getActiveStage(pipeline.StageList)
	if stage == nil {
		return nil
	}
	for _, task := range stage.TaskList {
		switch task.Status {
		case api.TaskPendingApproval:
			policy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
			if err != nil {
				return errors.Wrapf(err, "failed to get approval policy for environment ID %d", task.Instance.EnvironmentID)
			}
			if policy.Value == api.PipelineApprovalValueManualNever {
				// transit into Pending for ManualNever (auto-approval) tasks if all required task checks passed.
				ok, err := s.canAutoApprove(ctx, task)
				if err != nil {
					return errors.Wrap(err, "failed to check if can auto-approve")
				}
				if ok {
					if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
						IDList:    []int{task.ID},
						UpdaterID: api.SystemBotID,
						Status:    api.TaskPending,
					}); err != nil {
						return errors.Wrap(err, "failed to change task status")
					}
				}
			}
		case api.TaskPending:
			_, err := s.ScheduleIfNeeded(ctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to schedule task")
			}
		}
	}
	return nil
}

// patchTaskStatus patches a single task.
func (s *TaskScheduler) patchTaskStatus(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
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
		cancelAny, ok := s.runningTasksCancel.Load(task.ID)
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

	// If create database, schema update and gh-ost cutover task completes, we sync the corresponding instance schema immediately.
	if (taskPatched.Type == api.TaskDatabaseCreate || taskPatched.Type == api.TaskDatabaseSchemaUpdate || taskPatched.Type == api.TaskDatabaseSchemaUpdateSDL || taskPatched.Type == api.TaskDatabaseSchemaUpdateGhostCutover) && taskPatched.Status == api.TaskDone {
		instance, err := s.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to sync instance schema after completing task")
		}
		if err := s.schemaSyncer.syncDatabaseSchema(ctx, instance, taskPatched.Database.Name); err != nil {
			log.Error("failed to sync database schema",
				zap.String("instanceName", instance.Name),
				zap.String("databaseName", taskPatched.Database.Name),
				zap.Error(err),
			)
		}
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
			if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, externalApprovalCancelReasonNoTaskPendingApproval); err != nil {
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
				if _, err := s.changeIssueStatus(ctx, issue, api.IssueDone, taskStatusPatch.UpdaterID, ""); err != nil {
					return nil, errors.Wrapf(err, "failed to mark issue %v as DONE after completing task %v", issue.Name, taskPatched.Name)
				}
			}
		}
	}

	return taskPatched, nil
}

func (s *TaskScheduler) createTaskStatusUpdateActivity(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch, issue *api.Issue) error {
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

func (s *TaskScheduler) cancelDependingTasks(ctx context.Context, task *api.Task) error {
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
	if _, err := s.store.PatchTaskStatus(ctx, &api.TaskStatusPatch{
		IDList:    idList,
		UpdaterID: api.SystemBotID,
		Status:    api.TaskCanceled,
	}); err != nil {
		return err
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

func (s *TaskScheduler) getDefaultAssigneeID(ctx context.Context, environmentID int, projectID int, issueType api.IssueType) (int, error) {
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
func (s *TaskScheduler) getAnyWorkspaceOwnerOrDBA(ctx context.Context) (*api.Member, error) {
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
func (s *TaskScheduler) getAnyProjectOwner(ctx context.Context, projectID int) (*api.ProjectMember, error) {
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

// canPrincipalBeAssignee checks if a principal could be the assignee of an issue, judging by the principal role and the environment policy.
func (s *TaskScheduler) canPrincipalBeAssignee(ctx context.Context, principalID int, environmentID int, projectID int, issueType api.IssueType) (bool, error) {
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

func (s *TaskScheduler) changeIssueStatus(ctx context.Context, issue *api.Issue, newStatus api.IssueStatus, updaterID int, comment string) (*api.Issue, error) {
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
					if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
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
		if err := s.applicationRunner.CancelExternalApproval(ctx, issue.ID, externalApprovalCancelReasonIssueNotOpen); err != nil {
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
