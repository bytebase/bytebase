package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"

	"go.uber.org/zap"
)

const (
	taskSchedulerInterval = time.Duration(1) * time.Second
)

// NewTaskScheduler creates a new task scheduler.
func NewTaskScheduler(server *Server) *TaskScheduler {
	return &TaskScheduler{
		executorGetters:  make(map[api.TaskType]func() TaskExecutor),
		runningExecutors: make(map[int]TaskExecutor),
		server:           server,
	}
}

// TaskScheduler is the task scheduler.
type TaskScheduler struct {
	executorGetters  map[api.TaskType]func() TaskExecutor
	runningExecutors map[int]TaskExecutor
	taskProgress     sync.Map // map[taskID]api.Progress
	sharedTaskState  sync.Map // map[taskID]interface{}
	server           *Server
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

				// Collect completed tasks
				for i, executor := range s.runningExecutors {
					if executor.IsCompleted() {
						delete(s.runningExecutors, i)
						s.taskProgress.Delete(i)
					}
				}

				// Update task progress
				for i, executor := range s.runningExecutors {
					s.taskProgress.Store(i, executor.GetProgress())
				}

				// Inspect all open pipelines and schedule the next PENDING task if applicable
				pipelineStatus := api.PipelineOpen
				pipelineFind := &api.PipelineFind{
					Status: &pipelineStatus,
				}
				pipelineList, err := s.server.store.FindPipeline(ctx, pipelineFind, false)
				if err != nil {
					log.Error("Failed to retrieve open pipelines", zap.Error(err))
					return
				}
				for _, pipeline := range pipelineList {
					if err := s.server.ScheduleActiveStage(ctx, pipeline); err != nil {
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
				taskList, err := s.server.store.FindTask(ctx, taskFind, false)
				if err != nil {
					log.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}

				for _, task := range taskList {
					// Skip task belongs to archived instances
					if i := task.Instance; i == nil || i.RowStatus == api.Archived {
						continue
					}

					executorGetter, ok := s.executorGetters[task.Type]
					if !ok {
						log.Error("Skip running task with unknown type",
							zap.Int("id", task.ID),
							zap.String("name", task.Name),
							zap.String("type", string(task.Type)),
						)
						continue
					}

					if _, ok := s.runningExecutors[task.ID]; ok {
						continue
					}
					s.runningExecutors[task.ID] = executorGetter()

					go func(task *api.Task, executor TaskExecutor) {
						done, result, err := RunTaskExecutorOnce(ctx, executor, s.server, task)
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
							_, err = s.server.patchTaskStatus(ctx, task, taskStatusPatch)
							if err != nil {
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
							taskPatched, err := s.server.patchTaskStatus(ctx, task, taskStatusPatch)
							if err != nil {
								log.Error("Failed to mark task as DONE",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.Error(err),
								)
								return
							}

							issue, err := s.server.store.GetIssueByPipelineID(ctx, taskPatched.PipelineID)
							if err != nil {
								log.Error("failed to getIssueByPipelineID", zap.Int("pipelineID", taskPatched.PipelineID), zap.Error(err))
								return
							}
							// The task has finished, and we may move to a new stage.
							// if the current assignee doesn't fit in the new assignee group, we will reassign a new one based on the new assignee group.
							if issue != nil {
								if stage := getActiveStage(issue.Pipeline.StageList); stage != nil && stage.ID != taskPatched.StageID {
									environmentID := stage.EnvironmentID
									ok, err := s.server.canPrincipalBeAssignee(ctx, issue.AssigneeID, environmentID, issue.ProjectID, issue.Type)
									if err != nil {
										log.Error("failed to check if the current assignee still fits in the new assignee group", zap.Error(err))
										return
									}
									if !ok {
										// reassign the issue to a new assignee if the current one doesn't fit.
										assigneeID, err := s.server.getDefaultAssigneeID(ctx, environmentID, issue.ProjectID, issue.Type)
										if err != nil {
											log.Error("failed to get a default assignee", zap.Error(err))
											return
										}
										patch := &api.IssuePatch{
											ID:         issue.ID,
											UpdaterID:  api.SystemBotID,
											AssigneeID: &assigneeID,
										}
										if _, err := s.server.store.PatchIssue(ctx, patch); err != nil {
											log.Error("failed to update the issue assignee", zap.Any("issuePatch", patch))
											return
										}
									}
								}
							}
							return
						}
					}(task, s.runningExecutors[task.ID])
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Register will register a task executor factory.
func (s *TaskScheduler) Register(taskType api.TaskType, executorGetter func() TaskExecutor) {
	if executorGetter == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executorGetters[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executorGetters[taskType] = executorGetter
}

func (s *TaskScheduler) passAllCheck(ctx context.Context, task *api.Task, allowedStatus api.TaskCheckStatus) (bool, error) {
	// schema update, data update and gh-ost sync task have required task check.
	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseDataUpdate || task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := s.server.passCheck(ctx, task, api.TaskCheckDatabaseConnect, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		pass, err = s.server.passCheck(ctx, task, api.TaskCheckInstanceMigrationSchema, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}

		instance, err := s.server.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return false, err
		}
		if instance == nil {
			return false, errors.Errorf("instance ID not found %v", task.InstanceID)
		}

		if api.IsSyntaxCheckSupported(instance.Engine, s.server.profile.Mode) {
			pass, err = s.server.passCheck(ctx, task, api.TaskCheckDatabaseStatementSyntax, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}

		if s.server.feature(api.FeatureSQLReviewPolicy) && api.IsSQLReviewSupported(instance.Engine, s.server.profile.Mode) {
			pass, err = s.server.passCheck(ctx, task, api.TaskCheckDatabaseStatementAdvise, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}

		if instance.Engine == db.Postgres {
			pass, err = s.server.passCheck(ctx, task, api.TaskCheckDatabaseStatementType, allowedStatus)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}
	}

	if task.Type == api.TaskDatabaseSchemaUpdateGhostSync {
		pass, err := s.server.passCheck(ctx, task, api.TaskCheckGhostSync, allowedStatus)
		if err != nil {
			return false, err
		}
		if !pass {
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
	// timing task check
	if task.EarliestAllowedTs != 0 {
		pass, err := s.server.passCheck(ctx, task, api.TaskCheckGeneralEarliestAllowedTime, api.TaskCheckStatusSuccess)
		if err != nil {
			return false, err
		}
		if !pass {
			return false, nil
		}
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

	updatedTask, err := s.server.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
		ID:        task.ID,
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
		blockingTask, err := s.server.store.GetTaskByID(ctx, blockingTaskID)
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
