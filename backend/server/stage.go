package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
		stages, err := s.store.ListStageV2(ctx, pipelineID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find stage %v", stageID)).SetInternal(err)
		}
		var stage *store.StageMessage
		for _, v := range stages {
			if v.ID == stageID {
				stage = v
				break
			}
		}
		if stage == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid stage %v", stageID))
		}
		activeStage := utils.GetActiveStage(stages)
		if activeStage == nil {
			return echo.NewHTTPError(http.StatusBadRequest, "all stages are done")
		}

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
		tasks, err := s.store.ListTasks(ctx, &api.TaskFind{PipelineID: &pipelineID, StageID: &stageID, StatusList: &pendingApprovalStatus})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tasks").SetInternal(err)
		}
		if len(tasks) == 0 {
			return echo.NewHTTPError(http.StatusInternalServerError, "No task to approve in the stage")
		}

		// pick any task in the stage to validate
		// because all tasks in the same stage share the issue & environment.
		ok, err := s.TaskScheduler.CanPrincipalChangeTaskStatus(ctx, currentPrincipalID, tasks[0], stageAllTaskStatusPatch.Status)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not allowed to change task status")
		}

		if stageAllTaskStatusPatch.Status == api.TaskPending {
			for _, task := range tasks {
				instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
				if err != nil {
					return err
				}
				taskCheckRuns, err := s.store.ListTaskCheckRuns(ctx, &store.TaskCheckRunFind{TaskID: &task.ID})
				if err != nil {
					return err
				}
				ok, err = utils.PassAllCheck(task, api.TaskCheckStatusWarn, taskCheckRuns, instance.Engine)
				if err != nil {
					return err
				}
				if !ok {
					return echo.NewHTTPError(http.StatusBadRequest, "The task has not passed all the checks yet")
				}
			}
			if tasks[0].StageID != activeStage.ID {
				return echo.NewHTTPError(http.StatusBadRequest, "We can only approve the earliest stage with incompleted tasks")
			}
		}

		var taskIDList []int
		for _, task := range tasks {
			taskIDList = append(taskIDList, task.ID)
		}
		if err := s.store.BatchPatchTaskStatus(ctx, taskIDList, api.TaskPending, currentPrincipalID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task %q status", taskIDList)).SetInternal(err)
		}
		issue, err := s.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &pipelineID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch containing issue").SetInternal(err)
		}
		if issue == nil {
			return echo.NewHTTPError(http.StatusNotFound, "issue not found")
		}
		if err := s.ActivityManager.BatchCreateTaskStatusUpdateApprovalActivity(ctx, tasks, currentPrincipalID, issue, stage.Name); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to create task status update activity").SetInternal(err)
		}

		return c.String(http.StatusOK, "")
	})
}
