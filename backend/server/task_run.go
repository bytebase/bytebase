package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) ComposeTaskRunListByTaskId(ctx context.Context, taskId int, includeList []string) ([]*api.TaskRun, error) {
	taskRunFind := &api.TaskRunFind{
		TaskId: &taskId,
	}
	taskRunList, err := s.TaskRunService.FindTaskRunList(context.Background(), taskRunFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task run list for task ID: %v", taskId)).SetInternal(err)
	}
	for _, taskRun := range taskRunList {
		if err := s.ComposeTaskRunRelationship(ctx, taskRun, includeList); err != nil {
			return nil, err
		}
	}

	return taskRunList, nil
}

func (s *Server) ComposeTaskRunRelationship(ctx context.Context, taskRun *api.TaskRun, includeList []string) error {
	var err error

	taskRun.Creator, err = s.ComposePrincipalById(context.Background(), taskRun.CreatorId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch creator for taskRun: %v", taskRun.Name)).SetInternal(err)
	}

	taskRun.Updater, err = s.ComposePrincipalById(context.Background(), taskRun.UpdaterId, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updater for taskRun: %v", taskRun.Name)).SetInternal(err)
	}

	return nil
}
