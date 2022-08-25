package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/pkg/errors"
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
			task, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, true /* skipIfAlreadyTerminated */)
			if err != nil {
				return errors.Wrap(err, "failed to schedule check")
			}
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
			if _, err := s.TaskCheckScheduler.ScheduleCheckIfNeeded(ctx, task, api.SystemBotID, true /* skipIfAlreadyTerminated */); err != nil {
				return errors.Wrap(err, "failed to schedule check")
			}
			_, err := s.TaskScheduler.ScheduleIfNeeded(ctx, task)
			if err != nil {
				return errors.Wrap(err, "failed to schedule task")
			}
		}
	}
	return nil
}
