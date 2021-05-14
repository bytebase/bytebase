package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerEnvironmentRoutes(g *echo.Group) {
	g.POST("/environment", func(c echo.Context) error {
		environmentCreate := &api.EnvironmentCreate{WorkspaceId: api.DEFAULT_WORKPSACE_ID}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, environmentCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create environment request").SetInternal(err)
		}

		environmentCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		environment, err := s.EnvironmentService.CreateEnvironment(context.Background(), environmentCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Environment name already exists: %s", environmentCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create environment").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create environment response").SetInternal(err)
		}
		return nil
	})

	g.GET("/environment", func(c echo.Context) error {
		workspaceId := api.DEFAULT_WORKPSACE_ID
		environmentFind := &api.EnvironmentFind{
			WorkspaceId: &workspaceId,
		}
		list, err := s.EnvironmentService.FindEnvironmentList(context.Background(), environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		environmentPatch := &api.EnvironmentPatch{
			ID:          id,
			WorkspaceId: api.DEFAULT_WORKPSACE_ID,
			UpdaterId:   c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, environmentPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch environment request").SetInternal(err)
		}

		environment, err := s.EnvironmentService.PatchEnvironmentByID(context.Background(), environmentPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal environment ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/environment/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		environmentDelete := &api.EnvironmentDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.EnvironmentService.DeleteEnvironmentByID(context.Background(), environmentDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete environment ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.PATCH("/environment/reorder", func(c echo.Context) error {
		patchList, err := jsonapi.UnmarshalManyPayload(c.Request().Body, reflect.TypeOf(new(api.EnvironmentPatch)))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted environment batch update request").SetInternal(err)
		}

		for _, item := range patchList {
			environmentPatch, _ := item.(*api.EnvironmentPatch)
			environmentPatch.WorkspaceId = api.DEFAULT_WORKPSACE_ID
			environmentPatch.UpdaterId = c.Get(GetPrincipalIdContextKey()).(int)
			_, err = s.EnvironmentService.PatchEnvironmentByID(context.Background(), environmentPatch)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", environmentPatch.ID))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", environmentPatch.ID)).SetInternal(err)
			}
		}

		workspaceId := api.DEFAULT_WORKPSACE_ID
		environmentFind := &api.EnvironmentFind{
			WorkspaceId: &workspaceId,
		}
		list, err := s.EnvironmentService.FindEnvironmentList(context.Background(), environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for batch update").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment batch update response").SetInternal(err)
		}
		return nil
	})
}
