package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/pkg/errors"
)

// ScheduleNextTaskIfNeeded tries to schedule the next task if needed.
// Returns nil if no task applicable can be scheduled.
func (s *Server) ScheduleNextTaskIfNeeded(ctx context.Context, pipeline *api.Pipeline) error {
	//TODO(p0ny): concurrent
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			// Should short circuit upon reaching RUNNING or FAILED task.
			if task.Status == api.TaskRunning || task.Status == api.TaskFailed {
				return nil
			}

			skipIfAlreadyTerminated := true
			if task.Status == api.TaskPendingApproval {
				task, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, skipIfAlreadyTerminated)
				if err != nil {
					return err
				}

				policy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
				if err != nil {
					return errors.Wrapf(err, "failed to get approval policy for environment ID %d", task.Instance.EnvironmentID)
				}
				if policy.Value == api.PipelineApprovalValueManualNever {
					// transit into Pending for ManualNever (auto-approval) tasks if all required task checks passed.
					ok, err := s.TaskScheduler.canAutoApprove(ctx, task)
					if err != nil {
						return err
					}
					if ok {
						if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
							ID:        task.ID,
							UpdaterID: api.SystemBotID,
							Status:    api.TaskPending,
						}); err != nil {
							return err
						}
					}
				}
				return nil
			}

			if task.Status == api.TaskPending {
				if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, skipIfAlreadyTerminated); err != nil {
					return err
				}
				_, err := s.TaskScheduler.ScheduleIfNeeded(ctx, task)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}
