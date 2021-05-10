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

func (s *Server) registerMemberRoutes(g *echo.Group) {
	g.POST("/member", func(c echo.Context) error {
		memberCreate := &api.MemberCreate{WorkspaceId: api.DEFAULT_WORKPSACE_ID}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create member request").SetInternal(err)
		}

		memberCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		member, err := s.MemberService.CreateMember(context.Background(), memberCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Member for user ID already exists: %d", memberCreate.PrincipalId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create member").SetInternal(err)
		}

		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create member response").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/member", func(c echo.Context) error {
		wsId := api.DEFAULT_WORKPSACE_ID
		memberFilter := &api.MemberFilter{
			WorkspaceId: &wsId,
		}
		list, err := s.MemberService.FindMemberList(context.Background(), memberFilter)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch member list").SetInternal(err)
		}

		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal member list response").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.PATCH("/member/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		memberPatch := &api.MemberPatch{
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch member request").SetInternal(err)
		}

		member, err := s.MemberService.PatchMemberByID(context.Background(), id, memberPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch member ID: %v", id)).SetInternal(err)
		}

		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal member ID response: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.DELETE("/member/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		memberDelete := &api.MemberDelete{
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.MemberService.DeleteMemberByID(context.Background(), id, memberDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete member ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}
