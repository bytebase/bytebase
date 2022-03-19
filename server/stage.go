package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) composeStageListByPipelineID(ctx context.Context, pipelineID int) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineID: &pipelineID,
	}
	stageRawList, err := s.StageService.FindStageList(ctx, stageFind)
	if err != nil {
		return nil, err
	}

	var stageList []*api.Stage
	for _, stageRaw := range stageRawList {
		stage, err := s.composeStageRelationship(ctx, stageRaw)
		if err != nil {
			return nil, err
		}
		stageList = append(stageList, stage)
	}

	return stageList, nil
}

func (s *Server) composeStageRelationship(ctx context.Context, raw *api.StageRaw) (*api.Stage, error) {
	stage := raw.ToStage()

	creator, err := s.composePrincipalByID(ctx, stage.CreatorID)
	if err != nil {
		return nil, err
	}
	stage.Creator = creator

	updater, err := s.composePrincipalByID(ctx, stage.UpdaterID)
	if err != nil {
		return nil, err
	}
	stage.Updater = updater

	env, err := s.composeEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return nil, err
	}
	stage.Environment = env

	taskList, err := s.composeTaskListByPipelineAndStageID(ctx, stage.PipelineID, stage.ID)
	if err != nil {
		return nil, err
	}
	stage.TaskList = taskList

	return stage, nil
}

// TODO(dragonly): remove this hack
func (s *Server) composeStageRelationshipValidateOnly(ctx context.Context, stage *api.Stage) error {
	var err error
	stage.Creator, err = s.composePrincipalByID(ctx, stage.CreatorID)
	if err != nil {
		return err
	}

	stage.Updater, err = s.composePrincipalByID(ctx, stage.UpdaterID)
	if err != nil {
		return err
	}

	stage.Environment, err = s.composeEnvironmentByID(ctx, stage.EnvironmentID)
	if err != nil {
		return err
	}

	for _, task := range stage.TaskList {
		if err := s.composeTaskRelationshipValidateOnly(ctx, task); err != nil {
			return err
		}
	}

	return nil
}
