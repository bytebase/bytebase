package server

import (
	"context"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) composeStageListByPipelineID(ctx context.Context, pipelineID int) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineID: &pipelineID,
	}
	stageList, err := s.StageService.FindStageList(ctx, stageFind)
	if err != nil {
		return nil, err
	}

	for _, stage := range stageList {
		if err := s.composeStageRelationship(ctx, stage); err != nil {
			return nil, err
		}
	}

	return stageList, nil
}

func (s *Server) composeStageRelationship(ctx context.Context, stage *api.Stage) error {
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

	if stage.TaskList == nil {
		stage.TaskList, err = s.composeTaskListByPipelineAndStageID(ctx, stage.PipelineID, stage.ID)
		if err != nil {
			return err
		}
	} else {
		for i, task := range stage.TaskList {
			// TODO(dragonly): remove this hack
			taskRaw := task.ToRaw()
			tmp, err := s.composeTaskRelationship(ctx, taskRaw)
			if err != nil {
				return err
			}
			stage.TaskList[i] = tmp
		}
	}

	return nil
}

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
