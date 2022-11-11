package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
)

// ScheduleActiveStage tries to schedule the tasks in the active stage.
func (s *Server) ScheduleActiveStage(ctx context.Context, pipeline *api.Pipeline) error {
	stage := GetActiveStage(pipeline.StageList)
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
						IDList:    []int{task.ID},
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
	var createList []*api.TaskCheckRunCreate
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			create, err := s.TaskCheckScheduler.getTaskCheck(ctx, task, api.SystemBotID)
			if err != nil {
				return errors.Wrapf(err, "failed to get task check for task %d", task.ID)
			}
			createList = append(createList, create...)
		}
	}
	if _, err := s.store.BatchCreateTaskCheckRun(ctx, createList); err != nil {
		return errors.Wrap(err, "failed to batch insert TaskCheckRunCreate")
	}
	return nil
}
