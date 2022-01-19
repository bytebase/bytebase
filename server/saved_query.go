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

func (s *Server) registerSavedQueryRoutes(g *echo.Group) {
	g.POST("/savedquery", func(c echo.Context) error {
		ctx := context.Background()
		savedQueryCreate := &api.SavedQueryCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, savedQueryCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create saved query request").SetInternal(err)
		}

		if savedQueryCreate.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted saved query request, missing name")
		}
		if savedQueryCreate.Statement == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted saved query request, missing statement")
		}

		savedQueryCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		savedQuery, err := s.SavedQueryService.CreateSavedQuery(ctx, savedQueryCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create saved query").SetInternal(err)
		}

		if err := s.composeSavedQueryRelationship(ctx, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created saved query relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create saved query response").SetInternal(err)
		}
		return nil
	})

	g.GET("/savedquery", func(c echo.Context) error {
		ctx := context.Background()
		creatorID := c.Get(getPrincipalIDContextKey()).(int)
		savedQueryFind := &api.SavedQueryFind{
			CreatorID: &creatorID,
		}
		list, err := s.SavedQueryService.FindSavedQueryList(ctx, savedQueryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch saved query list").SetInternal(err)
		}

		for _, savedQuery := range list {
			if err := s.composeSavedQueryRelationship(ctx, savedQuery); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch saved query relationship: %v", savedQuery.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal saved query list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/savedquery/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		creatorID := c.Get(getPrincipalIDContextKey()).(int)
		savedQueryFind := &api.SavedQueryFind{
			ID:        &id,
			CreatorID: &creatorID,
		}

		savedQuery, err := s.SavedQueryService.FindSavedQuery(ctx, savedQueryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch saved query ID: %v", id)).SetInternal(err)
		}
		if savedQuery == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Saved query ID not found: %d", id))
		}

		if err := s.composeSavedQueryRelationship(ctx, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch saved query relationship: %v", savedQuery.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal saved query response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/savedquery/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		savedQueryPatch := &api.SavedQueryPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, savedQueryPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch saved query request").SetInternal(err)
		}

		savedQuery, err := s.SavedQueryService.PatchSavedQuery(ctx, savedQueryPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Saved query ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch saved query ID: %v", id)).SetInternal(err)
		}

		if err := s.composeSavedQueryRelationship(ctx, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated saved query relationship: %v", savedQuery.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, savedQuery); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal patch saved query response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/savedquery/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		savedQueryDelete := &api.SavedQueryDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		err = s.SavedQueryService.DeleteSavedQuery(ctx, savedQueryDelete)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete saved query ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeSavedQueryRelationship(ctx context.Context, savedQuery *api.SavedQuery) error {
	var err error

	savedQuery.Creator, err = s.composePrincipalByID(ctx, savedQuery.CreatorID)
	if err != nil {
		return err
	}

	savedQuery.Updater, err = s.composePrincipalByID(ctx, savedQuery.UpdaterID)
	if err != nil {
		return err
	}

	return nil
}
