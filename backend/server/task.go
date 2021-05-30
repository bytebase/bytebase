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
)

var (
	applicableTaskStatusTransition = map[api.TaskStatus][]api.TaskStatus{
		"PENDING":          {"RUNNING", "SKIPPED"},
		"PENDING_APPROVAL": {"PENDING", "SKIPPED"},
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
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update task status.").SetInternal(err)
		}

		allowTransition := false
		for _, allowedStatus := range applicableTaskStatusTransition[task.Status] {
			if allowedStatus == taskStatusPatch.Status {
				allowTransition = true
				break
			}
		}

		if !allowTransition {
			return echo.NewHTTPError(http.StatusBadRequest,
				fmt.Sprintf("Invalid task status transition from %v to %v. Applicable transition(s) %v", task.Status, taskStatusPatch.Status, applicableTaskStatusTransition[task.Status]))
		}

		updatedTask, err := s.TaskService.PatchTaskStatus(context.Background(), taskStatusPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task status: %d", taskId)).SetInternal(err)
		}

		if err := s.ComposeTaskRelationship(context.Background(), updatedTask, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated task relationship: %v", updatedTask.Name)).SetInternal(err)
		}

		// Create an activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
			TaskId:    taskId,
			OldStatus: task.Status,
			NewStatus: taskStatusPatch.Status,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after changing the task status: %v", task.Name)).SetInternal(err)
		}

		activityCreate := &api.ActivityCreate{
			CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			ContainerId: taskStatusPatch.ContainerId,
			Type:        api.ActivityPipelineTaskStatusUpdate,
			Comment:     taskStatusPatch.Comment,
			Payload:     payload,
		}
		_, err = s.ActivityService.CreateActivity(context.Background(), activityCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing the task status: %v", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedTask); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal update task status response: %s", updatedTask.Name)).SetInternal(err)
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

		task, err := s.ComposeTaskById(context.Background(), taskId, []string{})
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Task ID not found: %d", taskId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch task ID: %v", taskId)).SetInternal(err)
		}

		if task.Status != api.TaskPendingApproval {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Task is not waiting for approval: %v", task.Name))
		}

		// Create an approval activity
		payload, err := json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
			TaskId:    taskId,
			OldStatus: task.Status,
			NewStatus: api.TaskPending,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after approving the task: %v", task.Name)).SetInternal(err)
		}

		activityApproval := &api.ActivityCreate{
			CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			ContainerId: taskApprove.ContainerId,
			Type:        api.ActivityPipelineTaskStatusUpdate,
			Comment:     taskApprove.Comment,
			Payload:     payload,
		}
		_, err = s.ActivityService.CreateActivity(context.Background(), activityApproval)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create approval activity: %v", task.Name)).SetInternal(err)
		}

		// Schedule the task
		_, err = s.TaskScheduler.Schedule(context.Background(), *task, c.Get(GetPrincipalIdContextKey()).(int))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to schedule task after approval: %v", task.Name))
		}

		// Create a task running activity
		payload, err = json.Marshal(api.ActivityPipelineTaskStatusUpdatePayload{
			TaskId:    taskId,
			OldStatus: api.TaskPending,
			NewStatus: api.TaskRunning,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal activity after scheduling the task: %v", task.Name)).SetInternal(err)
		}

		activityTaskRunning := &api.ActivityCreate{
			// Task is invoked by the system
			CreatorId:   api.SYSTEM_BOT_ID,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			ContainerId: taskApprove.ContainerId,
			Type:        api.ActivityPipelineTaskStatusUpdate,
			Payload:     payload,
		}
		_, err = s.ActivityService.CreateActivity(context.Background(), activityTaskRunning)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after scheduling the task: %v", task.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
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

	task.TaskRunList, err = s.ComposeTaskRunListByTaskId(context.Background(), task.ID, includeList)
	if err != nil {
		return err
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
