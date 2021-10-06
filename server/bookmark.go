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

func (s *Server) registerBookmarkRoutes(g *echo.Group) {
	g.POST("/bookmark", func(c echo.Context) error {
		ctx := context.Background()
		bookmarkCreate := &api.BookmarkCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, bookmarkCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create bookmark request").SetInternal(err)
		}

		bookmarkCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		bookmark, err := s.BookmarkService.CreateBookmark(ctx, bookmarkCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Bookmark already exists: %s", bookmarkCreate.Link))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create bookmark").SetInternal(err)
		}

		if err := s.ComposeBookmarkRelationship(ctx, bookmark); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created bookmark relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, bookmark); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create bookmark response").SetInternal(err)
		}
		return nil
	})

	g.GET("/bookmark", func(c echo.Context) error {
		ctx := context.Background()
		creatorId := c.Get(GetPrincipalIdContextKey()).(int)
		bookmarkFind := &api.BookmarkFind{
			CreatorId: &creatorId,
		}
		list, err := s.BookmarkService.FindBookmarkList(ctx, bookmarkFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch bookmark list").SetInternal(err)
		}

		for _, bookmark := range list {
			if err := s.ComposeBookmarkRelationship(ctx, bookmark); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch bookmark relationship: %v", bookmark.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal bookmark list response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/bookmark/:bookmarkId", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("bookmarkId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("bookmarkId"))).SetInternal(err)
		}

		bookmarkDelete := &api.BookmarkDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.BookmarkService.DeleteBookmark(ctx, bookmarkDelete)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete bookmark ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeBookmarkRelationship(ctx context.Context, bookmark *api.Bookmark) error {
	var err error

	bookmark.Creator, err = s.ComposePrincipalById(ctx, bookmark.CreatorId)
	if err != nil {
		return err
	}

	bookmark.Updater, err = s.ComposePrincipalById(ctx, bookmark.UpdaterId)
	if err != nil {
		return err
	}

	return nil
}
