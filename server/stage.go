package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) registerStageRoutes(g *echo.Group) {
	// This function patches the status of all tasks in the stage.
	g.PATCH("/pipeline/:pipelineID/stage/:stageID/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		stageID, err := strconv.Atoi(c.Param("stageID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Stage ID is not a number: %s", c.Param("stageID"))).SetInternal(err)
		}
		pipelineID, err := strconv.Atoi(c.Param("pipelineID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Pipeline ID is not a number: %s", c.Param("pipelineID"))).SetInternal(err)
		}
		stageList, err := s.store.FindStage(ctx, &api.StageFind{ID: &stageID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find stage %v", stageID)).SetInternal(err)
		}
		if len(stageList) != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Find invalid number %v stages for stage %v", len(stageList), stageID)).SetInternal(err)
		}
		stage := stageList[0]

		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		stageAllTaskStatusPatch := &api.StageAllTaskStatusPatch{
			ID:        stageID,
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, stageAllTaskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update stage tasks status request").SetInternal(err)
		}

		if stageAllTaskStatusPatch.Status != api.TaskPending {
			return echo.NewHTTPError(http.StatusBadRequest, "Only support status transitioning from PENDING_APPROVAL to PENDING")
		}

		pendingApprovalStatus := []api.TaskStatus{api.TaskPendingApproval}
		tasks, err := s.store.FindTask(ctx, &api.TaskFind{PipelineID: &pipelineID, StageID: &stageID, StatusList: &pendingApprovalStatus}, true /* returnOnErr */)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tasks").SetInternal(err)
		}
		if len(tasks) == 0 {
			return echo.NewHTTPError(http.StatusInternalServerError, "No task to approve in the stage")
		}

		// pick any task in the stage to validate
		// because all tasks in the same stage share the issue & environment.
		ok, err := s.canPrincipalChangeTaskStatus(ctx, currentPrincipalID, tasks[0], stageAllTaskStatusPatch.Status)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not allowed to change task status")
		}
		var taskIDList []int
		for _, task := range tasks {
			taskIDList = append(taskIDList, task.ID)
		}
		if err := s.store.BatchPatchTaskStatus(ctx, taskIDList, api.TaskPending, currentPrincipalID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task %q status", taskIDList)).SetInternal(err)
		}
		issue, err := s.store.GetIssueByPipelineID(ctx, tasks[0].PipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch containing issue").SetInternal(err)
		}
		if err := s.ActivityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, tasks, currentPrincipalID, issue, stage); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task status update activity").SetInternal(err)
		}

		return c.String(http.StatusOK, "")
	})
}
