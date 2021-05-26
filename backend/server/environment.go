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

		if err := s.ComposeEnvironmentRelationship(context.Background(), environment, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created environment relationship").SetInternal(err)
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
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			environmentFind.RowStatus = &rowStatus
		}
		list, err := s.EnvironmentService.FindEnvironmentList(context.Background(), environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
		}

		for _, environment := range list {
			if err := s.ComposeEnvironmentRelationship(context.Background(), environment, c.Get(getIncludeKey()).([]string)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch environment relationship: %v", environment.Name)).SetInternal(err)
			}
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

		environment, err := s.EnvironmentService.PatchEnvironment(context.Background(), environmentPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeEnvironmentRelationship(context.Background(), environment, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated environment relationship: %v", environment.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal environment ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/reorder", func(c echo.Context) error {
		patchList, err := jsonapi.UnmarshalManyPayload(c.Request().Body, reflect.TypeOf(new(api.EnvironmentPatch)))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted environment reorder request").SetInternal(err)
		}

		for _, item := range patchList {
			environmentPatch, _ := item.(*api.EnvironmentPatch)
			environmentPatch.WorkspaceId = api.DEFAULT_WORKPSACE_ID
			environmentPatch.UpdaterId = c.Get(GetPrincipalIdContextKey()).(int)
			_, err = s.EnvironmentService.PatchEnvironment(context.Background(), environmentPatch)
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
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for reorder").SetInternal(err)
		}

		for _, environment := range list {
			if err := s.ComposeEnvironmentRelationship(context.Background(), environment, c.Get(getIncludeKey()).([]string)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch reordered environment relationship: %v", environment.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment reorder response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeEnvironmentById(ctx context.Context, id int, includeList []string) (*api.Environment, error) {
	environmentFind := &api.EnvironmentFind{
		ID: &id,
	}
	environment, err := s.EnvironmentService.FindEnvironment(context.Background(), environmentFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeEnvironmentRelationship(ctx, environment, includeList); err != nil {
		return nil, err
	}

	return environment, nil
}

func (s *Server) ComposeEnvironmentRelationship(ctx context.Context, environment *api.Environment, includeList []string) error {
	var err error

	environment.Creator, err = s.ComposePrincipalById(context.Background(), environment.CreatorId, includeList)
	if err != nil {
		return err
	}

	environment.Updater, err = s.ComposePrincipalById(context.Background(), environment.UpdaterId, includeList)
	if err != nil {
		return err
	}

	return nil
}
