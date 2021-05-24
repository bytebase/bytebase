package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"time"
)

const (
	STACK_SIZE = 4 << 10 // 4 KB
	ALL_STACK  = true

	INTERVAL = time.Duration(1) * time.Second
)

func NewScheduler(logger *log.Logger, db *sql.DB) *Scheduler {
	return &Scheduler{
		l:              logger,
		executors:      make(map[string]Executor),
		taskRunService: newTaskRunService(logger, db),
	}
}

type Scheduler struct {
	l         *log.Logger
	executors map[string]Executor

	taskRunService TaskRunService
}

func (s *Scheduler) Run() error {
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

						s.l.Println(msg)
					}
				}()
				status := TaskRunPending
				taskRunFind := &TaskRunFind{
					Status: &status,
				}
				list, err := s.taskRunService.FindTaskRunList(context.Background(), taskRunFind)
				if err != nil {
					s.l.Printf("Failed to retrieve pending tasks: %v\n", err)
				}

				for _, taskRun := range list {
					executor, ok := s.executors[taskRun.Type]
					if !ok {
						s.l.Printf("Unknown task type: %v. Skip.", taskRun.Type)
						continue
					}

					s.l.Printf("Try to change task '%v(%v)' to RUNNING.\n", taskRun.Name, taskRun.ID)
					if err := s.changeTaskRunStatus(taskRun.ID, TaskRunRunning); err != nil {
						s.l.Printf("Failed to change task: %v.\n", err)
						continue
					}

					done, err := executor.Run(context.Background(), *taskRun)
					if done {
						if err != nil {
							s.l.Printf("Task failed '%v(%v)': %v.\n", taskRun.Name, taskRun.ID, err)
							if err := s.changeTaskRunStatus(taskRun.ID, TaskRunFailed); err != nil {
								s.l.Printf("Failed to change task: %v.\n", err)
							}
							continue
						}
						s.l.Printf("Task completed '%v(%v)'.\n", taskRun.Name, taskRun.ID)
						if err := s.changeTaskRunStatus(taskRun.ID, TaskRunDone); err != nil {
							s.l.Printf("Failed to change task: %v.\n", err)
						}
					}
				}
			}()
		}
	}()

	return nil
}

func (s *Scheduler) Register(taskType string, executor Executor) {
	if executor == nil {
		panic("scheduler: Register executor is nil for task type: " + taskType)
	}
	if _, dup := s.executors[taskType]; dup {
		panic("scheduler: Register called twice for task type: " + taskType)
	}
	s.executors[taskType] = executor
}

func (s *Scheduler) Schedule(task Task) (*TaskRun, error) {
	taskRunCreate := &TaskRunCreate{
		TaskId:  task.ID,
		Name:    task.Name,
		Type:    task.Type,
		Payload: task.Payload,
	}
	createdTaskRun, err := s.taskRunService.CreateTaskRun(context.Background(), taskRunCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return createdTaskRun, nil
}

// func (s *Scheduler) CheckTaskRunStatus(taskRunId TaskRunId) (TaskRunStatus, error) {
// 	taskRun, err := s.findTaskRun(context.Background(), taskRunId)
// 	if err != nil {
// 		return TaskRunUnknown, fmt.Errorf("failed to check task status %v: %w", taskRunId, err)
// 	}

// 	executor, ok := s.executors[taskRun.Type]
// 	if !ok {
// 		return TaskRunUnknown, fmt.Errorf("failed to check task status '%v(%v)': unknown task type: %v", taskRun.Name, taskRunId, taskRun.Type)
// 	}

// 	return executor.Status(taskRunId)
// }

// func (s *Scheduler) CancelTaskRun(taskRunId TaskRunId) error {
// 	taskRun, err := s.findTaskRun(context.Background(), taskRunId)
// 	if err != nil {
// 		return fmt.Errorf("failed to cancel task status %v: %w", taskRunId, err)
// 	}

// 	executor, ok := s.executors[taskRun.Type]
// 	if !ok {
// 		return fmt.Errorf("failed to cancel task status '%v(%v)': unknown task type: %v", taskRun.Name, taskRunId, taskRun.Type)
// 	}

// 	return executor.Cancel(taskRunId)
// }

// func (s *Scheduler) findTaskRun(ctx context.Context, taskRunId TaskRunId) (*TaskRun, error) {
// 	taskRunFind := &TaskRunFind{
// 		ID: &taskRunId,
// 	}
// 	taskRun, err := s.taskRunService.FindTaskRun(ctx, taskRunFind)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve task %v: %w", taskRunId, err)
// 	}

// 	return taskRun, nil
// }

func (s *Scheduler) changeTaskRunStatus(taskRunId TaskRunId, newStatus TaskRunStatus) error {
	taskRunStatusPatch := &TaskRunStatusPatch{
		ID:     taskRunId,
		Status: newStatus,
	}
	_, err := s.taskRunService.PatchTaskRunStatus(context.Background(), taskRunStatusPatch)
	if err != nil {
		return fmt.Errorf("failed to change task %v status: %w", taskRunId, err)
	}
	return nil
}
