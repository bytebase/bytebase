package server

import (
	"context"
	"fmt"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

const (
	TASK_SCHEDULE_INTERVAL = time.Duration(1) * time.Second
)

func NewTaskScheduler(logger *zap.Logger, server *Server) *TaskScheduler {
	return &TaskScheduler{
		l:         logger,
		executors: make(map[string]TaskExecutor),
		server:    server,
	}
}

type TaskScheduler struct {
	l         *zap.Logger
	executors map[string]TaskExecutor

	server *Server
}

func (s *TaskScheduler) Run() error {
	go func() {
		s.l.Debug(fmt.Sprintf("Task scheduler started and will run every %v", TASK_SCHEDULE_INTERVAL))
		runningTasks := make(map[int]bool)
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Task scheduler PANIC RECOVER", zap.Error(err))
					}
				}()

				// Inspect all open pipelines and schedule the next PENDING task if applicable
				pipelineStatus := api.Pipeline_Open
				pipelineFind := &api.PipelineFind{
					Status: &pipelineStatus,
				}
				pipelineList, err := s.server.PipelineService.FindPipelineList(context.Background(), pipelineFind)
				if err != nil {
					s.l.Error("Failed to retrieve open pipelines", zap.Error(err))
				}
				for _, pipeline := range pipelineList {
					if pipeline.ID == api.ONBOARDING_PIPELINE_ID {
						continue
					}
					if err := s.server.ComposePipelineRelationship(context.Background(), pipeline); err != nil {
						s.l.Error("Failed to fetch pipeline relationship",
							zap.Int("id", pipeline.ID),
							zap.String("name", pipeline.Name),
							zap.Error(err),
						)
						continue
					}

					if err := s.server.ScheduleNextTaskIfNeeded(context.Background(), pipeline); err != nil {
						s.l.Error("Failed to schedule next running task",
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
				taskList, err := s.server.TaskService.FindTaskList(context.Background(), taskFind)
				if err != nil {
					s.l.Error("Failed to retrieve running tasks", zap.Error(err))
				}

				for _, task := range taskList {
					if task.ID == api.ONBOARDING_TASK_ID1 || task.ID == api.ONBOARDING_TASK_ID2 {
						continue
					}

					executor, ok := s.executors[string(task.Type)]
					if !ok {
						s.l.Error("Skip running task with unknown type",
							zap.Int("id", task.ID),
							zap.String("name", task.Name),
							zap.String("type", string(task.Type)),
						)
						continue
					}

					// This fetches quite a bit info and may cause performance issue if we have many ongoing tasks
					// We may optimize this in the future since only some relationship info is needed by the executor
					if err := s.server.ComposeTaskRelationship(context.Background(), task); err != nil {
						s.l.Error("Failed to fetch task relationship",
							zap.Int("id", task.ID),
							zap.String("name", task.Name),
							zap.String("type", string(task.Type)),
						)
						continue
					}

					if _, ok := runningTasks[task.ID]; ok {
						continue
					}
					runningTasks[task.ID] = true
					go func(task *api.Task) {
						defer func() {
							delete(runningTasks, task.ID)
						}()
						done, detail, err := executor.RunOnce(context.Background(), s.server, task)
						time.Sleep(time.Second * 30)
						if done {
							if err == nil {
								taskStatusPatch := &api.TaskStatusPatch{
									ID:        task.ID,
									UpdaterId: api.SYSTEM_BOT_ID,
									Status:    api.TaskDone,
									Comment:   detail,
								}
								_, err = s.server.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
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
								taskStatusPatch := &api.TaskStatusPatch{
									ID:        task.ID,
									UpdaterId: api.SYSTEM_BOT_ID,
									Status:    api.TaskFailed,
									Comment:   err.Error(),
								}
								_, err = s.server.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
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

			time.Sleep(TASK_SCHEDULE_INTERVAL)
		}
	}()

	return nil
}

func (s *TaskScheduler) Register(taskType string, executor TaskExecutor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

func (s *TaskScheduler) Schedule(ctx context.Context, task *api.Task) (*api.Task, error) {
	updatedTask, err := s.server.ChangeTaskStatus(ctx, task, api.TaskRunning, api.SYSTEM_BOT_ID)
	if err != nil {
		return nil, err
	}

	return updatedTask, nil
}
