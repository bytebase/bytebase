package server

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerEnvironmentRoutes(g *echo.Group) {
	g.POST("/environment", func(c echo.Context) error {
		ctx := context.Background()
		environmentCreate := &api.EnvironmentCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, environmentCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create environment request").SetInternal(err)
		}

		environmentCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		environment, err := s.EnvironmentService.CreateEnvironment(ctx, environmentCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Environment name already exists: %s", environmentCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create environment").SetInternal(err)
		}

		if err := s.composeEnvironmentRelationship(ctx, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created environment relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create environment response").SetInternal(err)
		}
		return nil
	})

	g.GET("/environment", func(c echo.Context) error {
		ctx := context.Background()
		environmentFind := &api.EnvironmentFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			environmentFind.RowStatus = &rowStatus
		}
		list, err := s.EnvironmentService.FindEnvironmentList(ctx, environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
		}

		for _, environment := range list {
			if err := s.composeEnvironmentRelationship(ctx, environment); err != nil {
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
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Environment ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		environmentPatch := &api.EnvironmentPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, environmentPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch environment request").SetInternal(err)
		}

		environment, err := s.EnvironmentService.PatchEnvironment(ctx, environmentPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", id)).SetInternal(err)
		}

		if err := s.composeEnvironmentRelationship(ctx, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated environment relationship: %v", environment.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, environment); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal environment ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/environment/reorder", func(c echo.Context) error {
		ctx := context.Background()
		patchList, err := jsonapi.UnmarshalManyPayload(c.Request().Body, reflect.TypeOf(new(api.EnvironmentPatch)))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted environment reorder request").SetInternal(err)
		}

		for _, item := range patchList {
			environmentPatch, _ := item.(*api.EnvironmentPatch)
			environmentPatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)
			_, err = s.EnvironmentService.PatchEnvironment(ctx, environmentPatch)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", environmentPatch.ID))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", environmentPatch.ID)).SetInternal(err)
			}
		}

		environmentFind := &api.EnvironmentFind{}
		list, err := s.EnvironmentService.FindEnvironmentList(ctx, environmentFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for reorder").SetInternal(err)
		}

		for _, environment := range list {
			if err := s.composeEnvironmentRelationship(ctx, environment); err != nil {
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

func (s *Server) composeEnvironmentByID(ctx context.Context, id int) (*api.Environment, error) {
	environmentFind := &api.EnvironmentFind{
		ID: &id,
	}
	environment, err := s.EnvironmentService.FindEnvironment(ctx, environmentFind)
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, fmt.Errorf("environment ID not found %v", id)
	}

	if err := s.composeEnvironmentRelationship(ctx, environment); err != nil {
		return nil, err
	}

	return environment, nil
}

func (s *Server) composeEnvironmentRelationship(ctx context.Context, environment *api.Environment) error {
	var err error

	environment.Creator, err = s.composePrincipalByID(ctx, environment.CreatorID)
	if err != nil {
		return err
	}

	environment.Updater, err = s.composePrincipalByID(ctx, environment.UpdaterID)
	if err != nil {
		return err
	}

	return nil
}
