package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposeStageListByPipelineId(ctx context.Context, pipelineId int, includeList []string) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineId: &pipelineId,
	}
	stageList, err := s.StageService.FindStageList(context.Background(), stageFind)
	if err != nil {
		return nil, err
	}

	for _, stage := range stageList {
		if err := s.ComposeStageRelationship(ctx, stage, includeList); err != nil {
			return nil, err
		}
	}

	return stageList, nil
}

func (s *Server) ComposeStageRelationship(ctx context.Context, stage *api.Stage, includeList []string) error {
	var err error
	stage.Creator, err = s.ComposePrincipalById(context.Background(), stage.CreatorId, includeList)
	if err != nil {
		return err
	}

	stage.Updater, err = s.ComposePrincipalById(context.Background(), stage.UpdaterId, includeList)
	if err != nil {
		return err
	}

	stage.Environment, err = s.ComposeEnvironmentById(context.Background(), stage.EnvironmentId, includeList)
	if err != nil {
		return err
	}

	stage.TaskList, err = s.ComposeTaskListByStageId(context.Background(), stage.ID, includeList)
	if err != nil {
		return err
	}

	return nil
}
