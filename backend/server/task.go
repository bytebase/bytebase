package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) ComposeTaskListByStageId(ctx context.Context, stageId int, incluedList []string) ([]*api.Task, error) {
	taskFind := &api.TaskFind{
		StageId: &stageId,
	}
	taskList, err := s.TaskService.FindTaskList(context.Background(), taskFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task for stage: %v", stageId)).SetInternal(err)
	}

	for _, task := range taskList {
		if err := s.ComposeTaskRelationship(ctx, task, incluedList); err != nil {
			return nil, err
		}
	}

	return taskList, nil
}

func (s *Server) ComposeTaskRelationship(ctx context.Context, task *api.Task, includeList []string) error {
	var err error

	task.Creator, err = s.ComposePrincipalById(context.Background(), task.CreatorId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for task: %v", task.Name)).SetInternal(err)
	}

	task.Updater, err = s.ComposePrincipalById(context.Background(), task.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for task: %v", task.Name)).SetInternal(err)
	}

	task.Database, err = s.ComposeDatabaseById(context.Background(), task.DatabaseId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database for task: %v", task.Name)).SetInternal(err)
	}

	return nil
}
