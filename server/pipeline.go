package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
)

// ScheduleNextTaskIfNeeded tries to schedule the next task if needed.
// Returns nil if no task applicable can be scheduled.
func (s *Server) ScheduleNextTaskIfNeeded(ctx context.Context, pipeline *api.Pipeline) (*api.Task, error) {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			// Should short circuit upon reaching RUNNING or FAILED task.
			if task.Status == api.TaskRunning || task.Status == api.TaskFailed {
				return nil, nil
			}

			skipIfAlreadyTerminated := true
			if task.Status == api.TaskPendingApproval {
				task, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, skipIfAlreadyTerminated)
				if err != nil {
					return nil, err
				}

				policy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
				if err != nil {
					return nil, fmt.Errorf("failed to get approval policy for environment ID %d, error: %w", task.Instance.EnvironmentID, err)
				}
				if policy.Value == api.PipelineApprovalValueManualNever {
					// transit into Pending for ManualNever (auto-approval) tasks if all required task checks passed.
					ok, err := s.TaskScheduler.canAutoApprove(ctx, task)
					if err != nil {
						return nil, err
					}
					if ok {
						if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
							ID:        task.ID,
							UpdaterID: api.SystemBotID,
							Status:    api.TaskPending,
						}); err != nil {
							return nil, err
						}
					}
				}
				return task, nil
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
