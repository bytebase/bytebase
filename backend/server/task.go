package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineId/task/:taskId/status", func(c echo.Context) error {
		taskId, err := strconv.Atoi(c.Param("taskId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskId"))).SetInternal(err)
		}

		taskStatusPatch := &api.TaskStatusPatch{
			ID:          taskId,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task status request").SetInternal(err)
		}

		updatedTask, err := s.TaskService.PatchTaskStatus(context.Background(), taskStatusPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task status: %d", taskId)).SetInternal(err)
		}

		if err := s.ComposeTaskRelationship(context.Background(), updatedTask, c.Get(getIncludeKey()).([]string)); err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task status response: %s", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeTaskListByStageId(ctx context.Context, stageId int, includeList []string) ([]*api.Task, error) {
	taskFind := &api.TaskFind{
		StageId: &stageId,
	}
	taskList, err := s.TaskService.FindTaskList(context.Background(), taskFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task for stage: %v", stageId)).SetInternal(err)
	}

	for _, task := range taskList {
		if err := s.ComposeTaskRelationship(ctx, task, includeList); err != nil {
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

	task.TaskRunList, err = s.ComposeTaskRunListByTaskId(context.Background(), task.ID, includeList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task run list for task: %v", task.Name)).SetInternal(err)
	}

	return nil
}

func (s *Server) ChangeTaskStatus(taskRun *api.TaskRun, newStatus api.TaskStatus) error {
	newTaskRunStatus := api.TaskRunUnknown
	switch newStatus {
	case api.TaskPending:
		newTaskRunStatus = api.TaskRunPending
	case api.TaskRunning:
		newTaskRunStatus = api.TaskRunRunning
	case api.TaskDone:
		newTaskRunStatus = api.TaskRunDone
	case api.TaskFailed:
		newTaskRunStatus = api.TaskRunFailed
	case api.TaskCanceled:
		newTaskRunStatus = api.TaskRunCanceled
	case api.TaskSkipped:
		newTaskRunStatus = api.TaskRunCanceled
	}
	taskStatusPatch := &api.TaskStatusPatch{
		ID:          taskRun.TaskId,
		UpdaterId:   taskRun.UpdaterId,
		WorkspaceId: taskRun.WorkspaceId,
		Status:      newStatus,
	}

	if newTaskRunStatus != api.TaskRunUnknown {
		taskStatusPatch.TaskRunId = &taskRun.ID
		taskStatusPatch.TaskRunStatus = &newTaskRunStatus
	}
	_, err := s.TaskService.PatchTaskStatus(context.Background(), taskStatusPatch)
	if err != nil {
		return fmt.Errorf("failed to change task %v status: %w", taskRun.TaskId, err)
	}
	return nil
}
