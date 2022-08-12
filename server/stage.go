package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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

		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		stageAllTaskStatusPatch := &api.StageAllTaskStatusPatch{
			ID:        stageID,
			UpdaterID: currentPrincipalID,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, stageAllTaskStatusPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update stage tasks status request").SetInternal(err)
		}

		tasks, err := s.store.FindTask(ctx, &api.TaskFind{PipelineID: &pipelineID, StageID: &stageID}, true /* returnOnErr */)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tasks").SetInternal(err)
		}
		if len(tasks) == 0 {
			// which is impossible, because we make sure at least there is one task in each stage.
			return echo.NewHTTPError(http.StatusInternalServerError, "No task in the stage")
		}
		// pick any task in the stage to validate
		// because all tasks in the same stage share the issue & environment.
		ok, err := s.canPrincipalChangeTaskStatus(ctx, currentPrincipalID, tasks[0])
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate if the principal can change task status").SetInternal(err)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not allowed to change task status")
		}
		var tasksPatched []*api.Task
		for _, task := range tasks {
			taskPatched, err := s.patchTaskStatus(ctx, task, &api.TaskStatusPatch{
				ID:        task.ID,
				UpdaterID: stageAllTaskStatusPatch.UpdaterID,
				Status:    stageAllTaskStatusPatch.Status,
			})
			if err != nil {
				if common.ErrorCode(err) == common.Invalid {
					return echo.NewHTTPError(http.StatusBadRequest, common.ErrorMessage(err))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update task \"%v\" status", task.Name)).SetInternal(err)
			}
			tasksPatched = append(tasksPatched, taskPatched)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, tasksPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal update tasks status response").SetInternal(err)
		}
		return nil
	})
}
