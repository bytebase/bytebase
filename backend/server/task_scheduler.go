package server

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

const (
	INTERVAL = time.Duration(1) * time.Second
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
			time.Sleep(INTERVAL)

			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						stack := make([]byte, STACK_SIZE)
						length := runtime.Stack(stack, ALL_STACK)
						msg := fmt.Sprintf("[Scheduler PANIC RECOVER] %v %s\n", err, stack[:length])

						s.l.Info(msg)
					}
				}()
				status := api.TaskRunPending
				taskRunFind := &api.TaskRunFind{
					Status: &status,
				}
				list, err := s.server.TaskRunService.FindTaskRunList(context.Background(), taskRunFind)
				if err != nil {
					s.l.Info(fmt.Sprintf("Failed to retrieve pending tasks: %v\n", err))
				}

				for _, taskRun := range list {
					executor, ok := s.executors[string(taskRun.Type)]
					if !ok {
						s.l.Info(fmt.Sprintf("Unknown task type: %v. Skip.", taskRun.Type))
						continue
					}

					s.l.Info(fmt.Sprintf("Try to change task '%v(%v)' to RUNNING.\n", taskRun.Name, taskRun.ID))
					if err := s.server.ChangeTaskStatus(taskRun, api.TaskRunning); err != nil {
						s.l.Info(fmt.Sprintf("Failed to change task: %v.\n", err))
						continue
					}

					done, err := executor.Run(context.Background(), s.server, *taskRun)
					if done {
						if err != nil {
							s.l.Info(fmt.Sprintf("Task failed '%v(%v)': %v.\n", taskRun.Name, taskRun.ID, err))
							if err := s.server.ChangeTaskStatus(taskRun, api.TaskFailed); err != nil {
								s.l.Info(fmt.Sprintf("Failed to change task: %v.\n", err))
							}
							continue
						}

						s.l.Info(fmt.Sprintf("Task completed '%v(%v)'.\n", taskRun.Name, taskRun.ID))
						if err := s.server.ChangeTaskStatus(taskRun, api.TaskDone); err != nil {
							s.l.Info(fmt.Sprintf("Failed to change task: %v.\n", err))
						}
					}
				}
			}()
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

func (s *TaskScheduler) Schedule(ctx context.Context, task api.Task, creatorId int) (*api.TaskRun, error) {
	taskRunCreate := &api.TaskRunCreate{
		CreatorId:   creatorId,
		WorkspaceId: api.DEFAULT_WORKPSACE_ID,
		TaskId:      task.ID,
		Name:        fmt.Sprintf("%s %d", task.Name, time.Now().Unix()),
		Type:        task.Type,
		Payload:     task.Payload,
	}
	createdTaskRun, err := s.server.TaskRunService.CreateTaskRun(ctx, taskRunCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return createdTaskRun, nil
}
