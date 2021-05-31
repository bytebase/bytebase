package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		"PENDING":          {"RUNNING", "SKIPPED"},
		"PENDING_APPROVAL": {"PENDING"},
		"RUNNING":          {"DONE", "FAILED", "CANCELED"},
		"DONE":             {},
		"FAILED":           {"RUNNING"},
		"CANCELED":         {"RUNNING"},
		"SKIPPED":          {},
	}
)

func (s *Server) registerTaskRoutes(g *echo.Group) {
	g.PATCH("/pipeline/:pipelineId/task/:taskId/status", func(c echo.Context) error {
		taskId, err := strconv.Atoi(c.Param("taskId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskId"))).SetInternal(err)
		}

		taskStatusPatch := &api.TaskStatusPatch{
			ID:          taskId,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update task status request").SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskId,
		}
		task, err := s.TaskService.FindTask(context.Background(), taskFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status").SetInternal(err)
		}

		updatedTask, err := s.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.EINVALID {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
		}

		if err := s.ComposeTaskRelationship(context.Background(), updatedTask, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task \"%v\" status response", updatedTask.Name)).SetInternal(err)
		}
		return nil
	})

	g.POST("/pipeline/:pipelineId/task/:taskId/approve", func(c echo.Context) error {
		taskId, err := strconv.Atoi(c.Param("taskId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task ID is not a number: %s", c.Param("taskId"))).SetInternal(err)
		}

		taskApprove := &api.TaskApprove{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, taskApprove); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change task status request").SetInternal(err)
		}

		taskFind := &api.TaskFind{
			ID: &taskId,
		}
		task, err := s.TaskService.FindTask(context.Background(), taskFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task ID: %v", taskId)).SetInternal(err)
		}

		taskStatusPatch := &api.TaskStatusPatch{
			ID:          task.ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			Status:      api.TaskPending,
			Comment:     taskApprove.Comment,
		}
		updatedTask, err := s.ChangeTaskStatusWithPatch(context.Background(), task, taskStatusPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.EINVALID {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to approve task \"%v\"", task.Name)).SetInternal(err)
		}

		// Schedule the task
		scheduledTask, err := s.TaskScheduler.Schedule(context.Background(), updatedTask)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to schedule task \"%v\" after approval", task.Name))
		}

		if err := s.ComposeTaskRelationship(context.Background(), scheduledTask, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch approved task \"%v\" relationship", updatedTask.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, scheduledTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal approved task \"%v\" status response", updatedTask.Name)).SetInternal(err)
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
		return nil, err
	}

	for _, task := range taskList {
		if err := s.ComposeTaskRelationship(ctx, task, includeList); err != nil {
			return nil, err
		}
	}

	return taskList, nil
}

func (s *Server) ComposeTaskById(ctx context.Context, id int, includeList []string) (*api.Task, error) {
	taskFind := &api.TaskFind{
		ID: &id,
	}
	task, err := s.TaskService.FindTask(context.Background(), taskFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeTaskRelationship(ctx, task, includeList); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *Server) ComposeTaskRelationship(ctx context.Context, task *api.Task, includeList []string) error {
	var err error

	task.Creator, err = s.ComposePrincipalById(context.Background(), task.CreatorId, includeList)
	if err != nil {
		return err
	}

	task.Updater, err = s.ComposePrincipalById(context.Background(), task.UpdaterId, includeList)
	if err != nil {
		return err
	}

	task.Database, err = s.ComposeDatabaseById(context.Background(), task.DatabaseId, includeList)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) ChangeTaskStatus(ctx context.Context, task *api.Task, newStatus api.TaskStatus, updatorId int) (*api.Task, error) {
	taskStatusPatch := &api.TaskStatusPatch{
		ID:          task.ID,
		UpdaterId:   updatorId,
		WorkspaceId: task.WorkspaceId,
		Status:      newStatus,
	}
	return s.ChangeTaskStatusWithPatch(ctx, task, taskStatusPatch)
}

func (s *Server) ChangeTaskStatusWithPatch(ctx context.Context, task *api.Task, taskStatusPatch *api.TaskStatusPatch) (_ *api.Task, err error) {
	defer func() {
		if err != nil {
			s.l.Error("Failed to change task status.",
				zap.Int("task_id", task.ID),
				zap.String("task_name", task.Name),
				zap.String("old_status", string(task.Status)),
				zap.String("new_status", string(taskStatusPatch.Status)),
				zap.Error(err))
		}
	}()
	allowTransition := false
	for _, allowedStatus := range applicableTaskStatusTransition[task.Status] {
		if allowedStatus == taskStatusPatch.Status {
			allowTransition = true
			break
		}
	}

	if !allowTransition {
		return nil, &bytebase.Error{
			Code:    bytebase.ENOTFOUND,
			Message: fmt.Sprintf("Invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status])}
	}

	updatedTask, err := s.TaskService.PatchTaskStatus(ctx, taskStatusPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to change task %v(%v) status: %w", task.ID, task.Name, err)
	}

	// Create an activity
	payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
		TaskId:    task.ID,
		OldStatus: task.Status,
		NewStatus: updatedTask.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal activity after changing the task status: %v, err: %w", task.Name, err)
	}

	// TODO: This indiciates a coupling that pipeline belongs to an issue.
	// A better way is to implement this as an onTaskStatusChange callback
	issueFind := &api.IssueFind{
		PipelineId: &task.PipelineId,
	}
	issue, err := s.IssueService.FindIssue(context.Background(), issueFind)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch containing issue for creating activity after changing the task status: %v, err: %w", task.Name, err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorId:   taskStatusPatch.UpdaterId,
		WorkspaceId: api.DEFAULT_WORKPSACE_ID,
		ContainerId: issue.ID,
		Type:        api.ActivityPipelineTaskStatusUpdate,
		Comment:     taskStatusPatch.Comment,
		Payload:     string(payload),
	}
	_, err = s.ActivityService.CreateActivity(context.Background(), activityCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity after changing the task status: %v, err: %w", task.Name, err)
	}

	return updatedTask, nil
}
