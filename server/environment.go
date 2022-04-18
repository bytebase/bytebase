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

		env, err := s.store.CreateEnvironment(ctx, envCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Environment name already exists: %s", envCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create environment").SetInternal(err)
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
		envList, err := s.store.FindEnvironment(ctx, envFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list").SetInternal(err)
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

		env, err := s.store.PatchEnvironment(ctx, envPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", id)).SetInternal(err)
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
				return echo.NewHTTPError(http.StatusBadRequest, "Malformatted environment reorder request").SetInternal(errors.New("failed to convert request item to *api.EnvironmentPatch"))
			}
			envPatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)
			if _, err := s.store.PatchEnvironment(ctx, envPatch); err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %d", envPatch.ID))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch environment ID: %v", envPatch.ID)).SetInternal(err)
			}
		}

		envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environment list for reorder").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, envList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal environment reorder response").SetInternal(err)
		}
		return nil
	})
}
