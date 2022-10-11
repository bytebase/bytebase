package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// ScheduleActiveStage tries to schedule the tasks in the active stage.
func (s *Server) ScheduleActiveStage(ctx context.Context, pipeline *api.Pipeline) error {
	stage := getActiveStage(pipeline.StageList)
	if stage == nil {
		return nil
	}
	for _, task := range stage.TaskList {
		switch task.Status {
		case api.TaskPendingApproval:
			policy, err := s.store.GetPipelineApprovalPolicy(ctx, task.Instance.EnvironmentID)
			if err != nil {
				return errors.Wrapf(err, "failed to get approval policy for environment ID %d", task.Instance.EnvironmentID)
			}
			if policy.Value == api.PipelineApprovalValueManualNever {
				// transit into Pending for ManualNever (auto-approval) tasks if all required task checks passed.
				ok, err := s.TaskScheduler.canAutoApprove(ctx, task)
				if err != nil {
					return errors.Wrap(err, "failed to check if can auto-approve")
				}
				if ok {
					if _, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
						ID:        task.ID,
						UpdaterID: api.SystemBotID,
						Status:    api.TaskPending,
					}); err != nil {
						return errors.Wrap(err, "failed to change task status")
					}
				}
			}
		case api.TaskPending:
			_, err := s.TaskScheduler.ScheduleIfNeeded(ctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to schedule task")
			}
		}
	}
	return nil
}

func (s *Server) schedulePipelineTaskCheck(ctx context.Context, pipeline *api.Pipeline) error {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID); err != nil {
				return errors.Wrapf(err, "failed to schedule task check for task %d", task.ID)
			}
		}
	}
	return nil
}
