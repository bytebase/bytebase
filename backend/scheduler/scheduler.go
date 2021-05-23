package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	uuid "github.com/satori/go.uuid"
)

const (
	interval = time.Duration(1) * time.Second
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
			time.Sleep(interval)
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

				s.l.Printf("Try to change task %v(%v) to RUNNING.\n", taskRun.Name, taskRun.ID)
				if err := s.changeTaskRunStatus(taskRun.ID, TaskRunRunning); err != nil {
					s.l.Printf("Failed to change task: %v.\n", err)
					continue
				}

				if err := executor.Run(context.Background(), *taskRun); err != nil {
					s.l.Printf("Failed to start executing task %v(%v) to RUNNING: %v.\n", taskRun.Name, taskRun.ID, err)
					if err := s.changeTaskRunStatus(taskRun.ID, TaskRunFailed); err != nil {
						s.l.Printf("Failed to change task: %v.\n", err)
					}
					continue
				}
			}
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
		ID:      uuid.NewV4().String(),
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
// 		return TaskRunUnknown, fmt.Errorf("failed to check task status %v(%v): unknown task type: %v", taskRun.Name, taskRunId, taskRun.Type)
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
// 		return fmt.Errorf("failed to cancel task status %v(%v): unknown task type: %v", taskRun.Name, taskRunId, taskRun.Type)
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
