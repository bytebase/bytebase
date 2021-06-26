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

func (s *Server) registerPrincipalRoutes(g *echo.Group) {
	g.POST("/principal", func(c echo.Context) error {
		principalCreate := &api.PrincipalCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create principal request").SetInternal(err)
		}

		principalCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		principalCreate.Type = api.EndUser
		principalCreate.PasswordHash = ""

		principal, err := s.PrincipalService.CreatePrincipal(context.Background(), principalCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, "User already exists")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create principal").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create principal response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal", func(c echo.Context) error {
		list, err := s.PrincipalService.FindPrincipalList(context.Background(), &api.PrincipalFind{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal list").SetInternal(err)
		}

		for _, principal := range list {
			if err := s.ComposePrincipalRole(context.Background(), principal); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch role for principal: %v", principal.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal principal list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal/:principalId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("principalId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		principal, err := s.ComposePrincipalById(context.Background(), id, c.Get(getIncludeKey()).([]string))
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch principal ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/principal/:principalId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("principalId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		principalPatch := &api.PrincipalPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int)}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch principal request").SetInternal(err)
		}

		principal, err := s.PrincipalService.PatchPrincipal(context.Background(), principalPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch principal ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposePrincipalById(ctx context.Context, id int, includeList []string) (*api.Principal, error) {
	principalFind := &api.PrincipalFind{
		ID: &id,
	}
	principal, err := s.PrincipalService.FindPrincipal(context.Background(), principalFind)
	if err != nil {
		return nil, err
	}

	s.ComposePrincipalRole(ctx, principal)

	return principal, nil
}

func (s *Server) ComposePrincipalRole(ctx context.Context, principal *api.Principal) error {
	if principal.ID == api.SYSTEM_BOT_ID {
		principal.Role = api.Owner
	} else {
		memberFind := &api.MemberFind{
			PrincipalId: &principal.ID,
		}
		member, err := s.MemberService.FindMember(context.Background(), memberFind)
		if err != nil {
			return err
		}
		principal.Role = member.Role
	}
	return nil
}
