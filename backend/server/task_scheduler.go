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
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Scheduler PANIC RECOVER", zap.Error(err))
					}
				}()

				// Inspect all open pipelines and schedule the next PENDING task if applicable
				pipelineStatus := api.Pipeline_Open
				pipelineFind := &api.PipelineFind{
					Status: &pipelineStatus,
				}
				pipelineList, err := s.server.PipelineService.FindPipelineList(context.Background(), pipelineFind)
				if err != nil {
					s.l.Error(fmt.Sprintf("Failed to retrieve open pipelines: %v\n", err))
				}
				for _, pipeline := range pipelineList {
					if err := s.server.ComposePipelineRelationship(context.Background(), pipeline, []string{}); err != nil {
						s.l.Error(fmt.Sprintf("Failed to fetch stage info for pipeline: %v. Error: %v", pipeline.Name, err))
						continue
					}

					for _, stage := range pipeline.StageList {
						for _, task := range stage.TaskList {
							if task.Status != api.TaskDone {
								if task.Status == api.TaskPending {
									_, err = s.Schedule(context.Background(), task)
									if err != nil {
										s.l.Error(fmt.Sprintf("Failed to schedule next running task: %v. Error: %v", task.Name, err))
									}
								}
								goto PIPELINE_END
							}
						}
					}
				PIPELINE_END:
				}

				// Inspect all running tasks
				taskStatus := api.TaskRunning
				taskFind := &api.TaskFind{
					Status: &taskStatus,
				}
				taskList, err := s.server.TaskService.FindTaskList(context.Background(), taskFind)
				if err != nil {
					s.l.Error(fmt.Sprintf("Failed to retrieve running tasks: %v\n", err))
				}

				for _, task := range taskList {
					executor, ok := s.executors[string(task.Type)]
					if !ok {
						s.l.Error(fmt.Sprintf("Unknown task type: %v. Skip", task.Type))
						continue
					}

					done, err := executor.RunOnce(context.Background(), s.server, task)
					if done {
						if err != nil {
							s.l.Info(fmt.Sprintf("Task failed '%v(%v)': %v\n", task.Name, task.ID, err))
							taskStatusPatch := &api.TaskStatusPatch{
								ID:        task.ID,
								UpdaterId: api.SYSTEM_BOT_ID,
								Status:    api.TaskFailed,
								Comment:   err.Error(),
							}
							s.server.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
							continue
						}

						s.l.Info(fmt.Sprintf("Task completed '%v(%v)'\n", task.Name, task.ID))
						_, err = s.server.ChangeTaskStatus(context.Background(), task, api.TaskDone, api.SYSTEM_BOT_ID)
						if err != nil {
							s.l.Error("Failed to mark task as DONE",
								zap.Error(err),
								zap.Int("task_id", task.ID),
								zap.String("task_name", task.Name),
							)
						}
					}
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
	s.l.Info(fmt.Sprintf("Try to change task '%v(%v)' to RUNNING\n", task.Name, task.ID))
	updatedTask, err := s.server.ChangeTaskStatus(ctx, task, api.TaskRunning, api.SYSTEM_BOT_ID)
	if err != nil {
		return nil, err
	}

	return updatedTask, nil
}
