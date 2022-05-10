package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

// ScheduleNextTaskIfNeeded tries to schedule the next task if needed.
// Returns nil if no task applicable can be scheduled
func (s *Server) ScheduleNextTaskIfNeeded(ctx context.Context, pipeline *api.Pipeline) (*api.Task, error) {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			// Should short circuit upon reaching RUNNING or FAILED task.
			if task.Status == api.TaskRunning || task.Status == api.TaskFailed {
				return nil, nil
			}

			skipIfAlreadyTerminated := true
			if task.Status == api.TaskPendingApproval {
				return s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, skipIfAlreadyTerminated)
			}

			if task.Status == api.TaskPending {
				if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, skipIfAlreadyTerminated); err != nil {
					return nil, err
				}
				updatedTask, err := s.TaskScheduler.ScheduleIfNeeded(ctx, task)
				if err != nil {
					return nil, err
				}
				return updatedTask, nil
			}
		}
	}
	return nil, nil
}
