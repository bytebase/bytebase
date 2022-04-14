package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

const (
	taskSchedulerInterval = time.Duration(1) * time.Second
)

// NewTaskScheduler creates a new task scheduler.
func NewTaskScheduler(logger *zap.Logger, server *Server) *TaskScheduler {
	return &TaskScheduler{
		l:         logger,
		executors: make(map[string]TaskExecutor),
		server:    server,
	}
}

// TaskScheduler is the task scheduler.
type TaskScheduler struct {
	l         *zap.Logger
	executors map[string]TaskExecutor

	server *Server
}

// Run will run the task scheduler.
func (s *TaskScheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(taskSchedulerInterval)
	defer ticker.Stop()
	defer wg.Done()
	s.l.Debug(fmt.Sprintf("Task scheduler started and will run every %v", taskSchedulerInterval))
	tasks := struct {
		running map[int]bool // task id set
		mu      sync.RWMutex
	}{
		running: make(map[int]bool),
		mu:      sync.RWMutex{},
	}
	for {
		select {
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Task scheduler PANIC RECOVER", zap.Error(err), zap.Stack("stack"))
					}
				}()

				ctx := context.Background()

				// Inspect all open pipelines and schedule the next PENDING task if applicable
				pipelineStatus := api.PipelineOpen
				pipelineFind := &api.PipelineFind{
					Status: &pipelineStatus,
				}
				pipelineRawList, err := s.server.PipelineService.FindPipelineList(ctx, pipelineFind)
				if err != nil {
					s.l.Error("Failed to retrieve open pipelines", zap.Error(err))
					return
				}
				for _, pipelineRaw := range pipelineRawList {
					if pipelineRaw.ID == api.OnboardingPipelineID {
						continue
					}
					pipeline, err := s.server.composePipelineRelationship(ctx, pipelineRaw)
					if err != nil {
						s.l.Error("Failed to fetch pipeline relationship",
							zap.Int("id", pipelineRaw.ID),
							zap.String("name", pipelineRaw.Name),
							zap.Error(err),
						)
						continue
					}

					if _, err := s.server.ScheduleNextTaskIfNeeded(ctx, pipeline); err != nil {
						s.l.Error("Failed to schedule next running task",
							zap.Int("pipeline_id", pipelineRaw.ID),
							zap.Error(err),
						)
					}
				}

				// Inspect all running tasks
				taskStatusList := []api.TaskStatus{api.TaskRunning}
				taskFind := &api.TaskFind{
					StatusList: &taskStatusList,
				}
				taskRawList, err := s.server.TaskService.FindTaskList(ctx, taskFind)
				if err != nil {
					s.l.Error("Failed to retrieve running tasks", zap.Error(err))
					return
				}

				for _, taskRaw := range taskRawList {
					if taskRaw.ID == api.OnboardingTaskID1 || taskRaw.ID == api.OnboardingTaskID2 {
						continue
					}

					executor, ok := s.executors[string(taskRaw.Type)]
					if !ok {
						s.l.Error("Skip running task with unknown type",
							zap.Int("id", taskRaw.ID),
							zap.String("name", taskRaw.Name),
							zap.String("type", string(taskRaw.Type)),
						)
						continue
					}

					// This fetches quite a bit info and may cause performance issue if we have many ongoing tasks
					// We may optimize this in the future since only some relationship info is needed by the executor
					task, err := s.server.composeTaskRelationship(ctx, taskRaw)
					if err != nil {
						s.l.Error("Failed to fetch task relationship",
							zap.Int("id", taskRaw.ID),
							zap.String("name", taskRaw.Name),
							zap.String("type", string(taskRaw.Type)),
						)
						continue
					}

					tasks.mu.Lock()
					if _, ok := tasks.running[task.ID]; ok {
						tasks.mu.Unlock()
						continue
					}
					tasks.running[task.ID] = true
					tasks.mu.Unlock()

					go func(task *api.Task) {
						defer func() {
							tasks.mu.Lock()
							delete(tasks.running, task.ID)
							tasks.mu.Unlock()
						}()
						done, result, err := executor.RunOnce(ctx, s.server, task)
						if done {
							if err == nil {
								bytes, err := json.Marshal(*result)
								if err != nil {
									s.l.Error("Failed to marshal task run result",
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
								_, err = s.server.changeTaskStatusWithPatch(ctx, task, taskStatusPatch)
								if err != nil {
									s.l.Error("Failed to mark task as DONE",
										zap.Int("id", task.ID),
										zap.String("name", task.Name),
										zap.Error(err),
									)
								}
							} else {
								s.l.Debug("Failed to run task",
									zap.Int("id", task.ID),
									zap.String("name", task.Name),
									zap.String("type", string(task.Type)),
									zap.Error(err),
								)
								bytes, marshalErr := json.Marshal(api.TaskRunResultPayload{
									Detail: err.Error(),
								})
								if marshalErr != nil {
									s.l.Error("Failed to marshal task run result",
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
								_, err = s.server.changeTaskStatusWithPatch(ctx, task, taskStatusPatch)
								if err != nil {
									s.l.Error("Failed to mark task as FAILED",
										zap.Int("id", task.ID),
										zap.String("name", task.Name),
										zap.Error(err),
									)
								}
							}
						} else if err != nil {
							s.l.Debug("Encountered transient error running task, will retry",
								zap.Int("id", task.ID),
								zap.String("name", task.Name),
								zap.String("type", string(task.Type)),
								zap.Error(err),
							)
						}
					}(task)
				}
			}()
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

// Register will register a task executor.
func (s *TaskScheduler) Register(taskType string, executor TaskExecutor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

// ScheduleIfNeeded schedules the task if its required check does not contain error in the latest run
func (s *TaskScheduler) ScheduleIfNeeded(ctx context.Context, task *api.Task) (*api.Task, error) {
	// timing task check
	if task.EarliestAllowedTs != 0 {
		pass, err := s.server.passCheck(ctx, s.server, task, api.TaskCheckGeneralEarliestAllowedTime)
		if err != nil {
			return nil, err
		}
		if !pass {
			return task, nil
		}
	}

	// only schema update or data update task has required task check
	if task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseDataUpdate {
		pass, err := s.server.passCheck(ctx, s.server, task, api.TaskCheckDatabaseConnect)
		if err != nil {
			return nil, err
		}
		if !pass {
			return task, nil
		}

		pass, err = s.server.passCheck(ctx, s.server, task, api.TaskCheckInstanceMigrationSchema)
		if err != nil {
			return nil, err
		}
		if !pass {
			return task, nil
		}

		instance, err := s.server.store.GetInstanceByID(ctx, task.InstanceID)
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, fmt.Errorf("instance ID not found %v", task.InstanceID)
		}
		// For now we only supported MySQL dialect syntax and compatibility check
		if instance.Engine == db.MySQL || instance.Engine == db.TiDB {
			pass, err = s.server.passCheck(ctx, s.server, task, api.TaskCheckDatabaseStatementSyntax)
			if err != nil {
				return nil, err
			}
			if !pass {
				return task, nil
			}

			if s.server.feature(api.FeatureBackwardCompatibility) {
				pass, err = s.server.passCheck(ctx, s.server, task, api.TaskCheckDatabaseStatementCompatibility)
				if err != nil {
					return nil, err
				}
				if !pass {
					return task, nil
				}
			}
		}
	}
	updatedTask, err := s.server.changeTaskStatus(ctx, task, api.TaskRunning, api.SystemBotID)
	if err != nil {
		return nil, err
	}

	return updatedTask, nil
}
