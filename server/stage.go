package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposeStageListByPipelineId(ctx context.Context, pipelineId int) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineId: &pipelineId,
	}
	stageList, err := s.StageService.FindStageList(context.Background(), stageFind)
	if err != nil {
		return nil, err
	}

	for _, stage := range stageList {
		if err := s.ComposeStageRelationship(ctx, stage); err != nil {
			return nil, err
		}
	}

	return stageList, nil
}

func (s *Server) ComposeStageRelationship(ctx context.Context, stage *api.Stage) error {
	var err error
	stage.Creator, err = s.ComposePrincipalById(context.Background(), stage.CreatorId)
	if err != nil {
		return err
	}

	stage.Updater, err = s.ComposePrincipalById(context.Background(), stage.UpdaterId)
	if err != nil {
		return err
	}

	stage.Environment, err = s.ComposeEnvironmentById(context.Background(), stage.EnvironmentId)
	if err != nil {
		return err
	}

	stage.TaskList, err = s.ComposeTaskListByPipelineAndStageId(context.Background(), stage.PipelineId, stage.ID)
	if err != nil {
		return err
	}

	return nil
}
