package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

var ()

func (s *Server) registerLabelRoutes(g *echo.Group) {
	g.GET("/label", func(c echo.Context) error {
		ctx := context.Background()
		rowStatus := api.Normal
		find := &api.LabelKeyFind{
			RowStatus: &rowStatus,
		}
		labelKeyRawList, err := s.LabelService.FindLabelKeyList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch label keys").SetInternal(err)
		}
		// TODO(dragonly): compose this
		var labelKeyList []*api.LabelKey
		for _, raw := range labelKeyRawList {
			labelKeyList = append(labelKeyList, raw.ToLabelKey())
		}

		// Add reserved environment key.
		envRawList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{RowStatus: &rowStatus})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch environments").SetInternal(err)
		}
		envKey := &api.LabelKey{Key: api.EnvironmentKeyName}
		for _, envRaw := range envRawList {
			envKey.ValueList = append(envKey.ValueList, envRaw.Name)
		}
		labelKeyList = append(labelKeyList, envKey)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, labelKeyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal label keys response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/label/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("id is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		patch := &api.LabelKeyPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch label key request").SetInternal(err)
		}

		if err := patch.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid patch label key request").SetInternal(err)
		}
		// We don't allow updating reserved environment label keys. Since its ID is zero, it cannot be updated by default.

		labelKeyRaw, err := s.LabelService.PatchLabelKey(ctx, patch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Label ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch label ID: %v", id)).SetInternal(err)
		}

		// TODO(dragonly): implement composeLabelKey
		labelKey := labelKeyRaw.ToLabelKey()
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, labelKey); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal label key response: %v", id)).SetInternal(err)
		}
		return nil
	})
}
