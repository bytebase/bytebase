package server

import (
	"context"
	"errors"
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
		envCreate := &api.EnvironmentCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, envCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create environment request").SetInternal(err)
		}

		envCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		envRaw, err := s.EnvironmentService.CreateEnvironment(ctx, envCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Environment name already exists: %s", envCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create environment").SetInternal(err)
		}

		env, err := s.composeEnvironmentRelationship(ctx, envRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created environment relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, env); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create environment response").SetInternal(err)
		}
		return nil
	})

	g.GET("/environment", func(c echo.Context) error {
		ctx := context.Background()
		envFind := &api.EnvironmentFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			envFind.RowStatus = &rowStatus
		}
		envRawList, err := s.EnvironmentService.FindEnvironmentList(ctx, envFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
		}

		var envList []*api.Environment
		for _, envRaw := range envRawList {
			env, err := s.composeEnvironmentRelationship(ctx, envRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch environment relationship: %v", envRaw.Name)).SetInternal(err)
			}
			envList = append(envList, env)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, envList); err != nil {
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

		envPatch := &api.EnvironmentPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, envPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch environment request").SetInternal(err)
		}

		envRaw, err := s.EnvironmentService.PatchEnvironment(ctx, envPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", id)).SetInternal(err)
		}

		env, err := s.composeEnvironmentRelationship(ctx, envRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated environment relationship: %v", envRaw.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, env); err != nil {
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
			envPatch, ok := item.(*api.EnvironmentPatch)
			if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformatted environment reorder request").SetInternal(errors.New("Failed to convert request item to *api.EnvironmentPatch"))
			}
			envPatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)
			if _, err := s.EnvironmentService.PatchEnvironment(ctx, envPatch); err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", envPatch.ID))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", envPatch.ID)).SetInternal(err)
			}
		}

		envFind := &api.EnvironmentFind{}
		envRawList, err := s.EnvironmentService.FindEnvironmentList(ctx, envFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for reorder").SetInternal(err)
		}

		var envList []*api.Environment
		for _, envRaw := range envRawList {
			env, err := s.composeEnvironmentRelationship(ctx, envRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch reordered environment relationship: %v", envRaw.Name)).SetInternal(err)
			}
			envList = append(envList, env)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, envList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment reorder response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeEnvironmentByID(ctx context.Context, id int) (*api.Environment, error) {
	envFind := &api.EnvironmentFind{
		ID: &id,
	}
	envRaw, err := s.EnvironmentService.FindEnvironment(ctx, envFind)
	if err != nil {
		return nil, err
	}
	if envRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("environment not found with ID %v", id)}
	}

	env, err := s.composeEnvironmentRelationship(ctx, envRaw)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (s *Server) composeEnvironmentRelationship(ctx context.Context, raw *api.EnvironmentRaw) (*api.Environment, error) {
	env := raw.ToEnvironment()

	creator, err := s.store.GetPrincipalByID(ctx, env.CreatorID)
	if err != nil {
		return nil, err
	}
	env.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, env.UpdaterID)
	if err != nil {
		return nil, err
	}
	env.Updater = updater

	return env, nil
}
