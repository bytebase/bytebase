package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposePipelineById(ctx context.Context, id int, includeList []string) (*api.Pipeline, error) {
	pipelineFind := &api.PipelineFind{
		ID: &id,
	}
	pipeline, err := s.PipelineService.FindPipeline(context.Background(), pipelineFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposePipelineRelationship(ctx, pipeline, includeList); err != nil {
		return nil, err
	}

	return pipeline, nil
}

func (s *Server) ComposePipelineRelationship(ctx context.Context, pipeline *api.Pipeline, includeList []string) error {
	var err error

	pipeline.Creator, err = s.ComposePrincipalById(context.Background(), pipeline.CreatorId, includeList)
	if err != nil {
		return err
	}

	pipeline.Updater, err = s.ComposePrincipalById(context.Background(), pipeline.UpdaterId, includeList)
	if err != nil {
		return err
	}

	pipeline.StageList, err = s.ComposeStageListByPipelineId(context.Background(), pipeline.ID, includeList)
	if err != nil {
		return err
	}

	return nil
}

// Try to schedule the next task if needed
func (s *Server) ScheduleNextTaskIfNeeded(ctx context.Context, pipeline *api.Pipeline) error {
	for _, stage := range pipeline.StageList {
		for _, task := range stage.TaskList {
			if task.Status == api.TaskPending {
				_, err := s.TaskScheduler.Schedule(context.Background(), task)
				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}
