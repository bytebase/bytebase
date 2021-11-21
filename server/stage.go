package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) ComposeStageListByPipelineID(ctx context.Context, pipelineID int) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineID: &pipelineID,
	}
	stageList, err := s.StageService.FindStageList(ctx, stageFind)
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
	stage.Creator, err = s.ComposePrincipalByID(ctx, stage.CreatorID)
	if err != nil {
		return err
	}

	stage.Updater, err = s.ComposePrincipalByID(ctx, stage.UpdaterID)
	if err != nil {
		return err
	}

	stage.Environment, err = s.ComposeEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return err
	}

	stage.TaskList, err = s.ComposeTaskListByPipelineAndStageID(ctx, stage.PipelineID, stage.ID)
	if err != nil {
		return err
	}

	return nil
}
