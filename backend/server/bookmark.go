package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerBookmarkRoutes(g *echo.Group) {
	g.POST("/bookmark", func(c echo.Context) error {
		bookmarkCreate := &api.BookmarkCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, bookmarkCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create bookmark request").SetInternal(err)
		}

		bookmarkCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		bookmark, err := s.BookmarkService.CreateBookmark(context.Background(), bookmarkCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Bookmark already exists: %s", bookmarkCreate.Link))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create bookmark").SetInternal(err)
		}

		if err := s.ComposeBookmarkRelationship(context.Background(), bookmark, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created bookmark relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, bookmark); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create bookmark response").SetInternal(err)
		}
		return nil
	})

	g.GET("/bookmark", func(c echo.Context) error {
		creatorId := c.Get(GetPrincipalIdContextKey()).(int)
		bookmarkFind := &api.BookmarkFind{
			CreatorId: &creatorId,
		}
		list, err := s.BookmarkService.FindBookmarkList(context.Background(), bookmarkFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch bookmark list").SetInternal(err)
		}

		for _, bookmark := range list {
			if err := s.ComposeBookmarkRelationship(context.Background(), bookmark, c.Get(getIncludeKey()).([]string)); err != nil {
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
		id, err := strconv.Atoi(c.Param("bookmarkId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("bookmarkId"))).SetInternal(err)
		}

		bookmarkDelete := &api.BookmarkDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.BookmarkService.DeleteBookmark(context.Background(), bookmarkDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete bookmark ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) ComposeBookmarkRelationship(ctx context.Context, bookmark *api.Bookmark, includeList []string) error {
	var err error

	bookmark.Creator, err = s.ComposePrincipalById(ctx, bookmark.CreatorId, includeList)
	if err != nil {
		return err
	}

	bookmark.Updater, err = s.ComposePrincipalById(context.Background(), bookmark.UpdaterId, includeList)
	if err != nil {
		return err
	}

	return nil
}
