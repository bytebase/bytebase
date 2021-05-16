package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) ComposeStageListByPipelineId(ctx context.Context, pipelineId int, incluedList []string) ([]*api.Stage, error) {
	stageFind := &api.StageFind{
		PipelineId: &pipelineId,
	}
	stageList, err := s.StageService.FindStageList(context.Background(), stageFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch stage for pipeline: %v", pipelineId)).SetInternal(err)
	}

	for _, stage := range stageList {
		if err := s.ComposeStageRelationship(ctx, stage, incluedList); err != nil {
			return nil, err
		}
	}

	return stageList, nil
}

func (s *Server) ComposeStageRelationship(ctx context.Context, stage *api.Stage, includeList []string) error {
	var err error
	stage.Creator, err = s.ComposePrincipalById(context.Background(), stage.CreatorId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for stage: %v", stage.Name)).SetInternal(err)
	}

	stage.Updater, err = s.ComposePrincipalById(context.Background(), stage.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for stage: %v", stage.Name)).SetInternal(err)
	}

	environmentFind := &api.EnvironmentFind{
		ID: &stage.EnvironmentId,
	}
	stage.Environment, err = s.EnvironmentService.FindEnvironment(context.Background(), environmentFind)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch environment for stage: %v", stage.Name)).SetInternal(err)
	}

	stage.TaskList, err = s.ComposeTaskListByStageId(context.Background(), stage.ID, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task list for stage: %v", stage.Name)).SetInternal(err)
	}

	return nil
}
