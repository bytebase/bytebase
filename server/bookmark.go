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

func (s *Server) registerBookmarkRoutes(g *echo.Group) {
	g.POST("/bookmark", func(c echo.Context) error {
		ctx := c.Request().Context()
		bookmarkCreate := &api.BookmarkCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, bookmarkCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create bookmark request").SetInternal(err)
		}

		bookmarkCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		bookmark, err := s.store.CreateBookmark(ctx, bookmarkCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Bookmark already exists: %s", bookmarkCreate.Link))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create bookmark").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, bookmark); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create bookmark response").SetInternal(err)
		}
		return nil
	})

	g.GET("/bookmark/user/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		bookmarkFind := &api.BookmarkFind{
			CreatorID: &userID,
		}
		bookmarkList, err := s.store.FindBookmark(ctx, bookmarkFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch bookmark list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, bookmarkList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal bookmark list response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/bookmark/:bookmarkID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("bookmarkID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("bookmarkID"))).SetInternal(err)
		}

		bookmarkDelete := &api.BookmarkDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		err = s.store.DeleteBookmark(ctx, bookmarkDelete)
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
