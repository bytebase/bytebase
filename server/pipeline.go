package server

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

func (s *Server) composePipelineByID(ctx context.Context, id int) (*api.Pipeline, error) {
	pipelineFind := &api.PipelineFind{
		ID: &id,
	}
	pipelineRaw, err := s.PipelineService.FindPipeline(ctx, pipelineFind)
	if err != nil {
		return nil, err
	}
	if pipelineRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("Pipeline not found with ID %v", id)}
	}

	pipeline, err := s.composePipelineRelationship(ctx, pipelineRaw)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

func (s *Server) composePipelineRelationship(ctx context.Context, raw *api.PipelineRaw) (*api.Pipeline, error) {
	pipeline := raw.ToPipeline()

	creator, err := s.store.GetPrincipalByID(ctx, pipeline.CreatorID)
	if err != nil {
		return nil, err
	}
	pipeline.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, pipeline.UpdaterID)
	if err != nil {
		return nil, err
	}
	pipeline.Updater = updater

	stageList, err := s.composeStageListByPipelineID(ctx, pipeline.ID)
	if err != nil {
		return nil, err
	}
	pipeline.StageList = stageList

	return pipeline, nil
}

// TODO(dragonly): remove this hack.
func (s *Server) composePipelineRelationshipValidateOnly(ctx context.Context, pipeline *api.Pipeline) error {
	var err error

	pipeline.Creator, err = s.store.GetPrincipalByID(ctx, pipeline.CreatorID)
	if err != nil {
		return err
	}

	pipeline.Updater, err = s.store.GetPrincipalByID(ctx, pipeline.UpdaterID)
	if err != nil {
		return err
	}

	for _, stage := range pipeline.StageList {
		if err := s.composeStageRelationshipValidateOnly(ctx, stage); err != nil {
			return err
		}
	}

	return nil
}

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
