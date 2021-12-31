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
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerPrincipalRoutes(g *echo.Group) {
	g.POST("/principal", func(c echo.Context) error {
		ctx := context.Background()
		principalCreate := &api.PrincipalCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create principal request").SetInternal(err)
		}

		principalCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		principalCreate.Type = api.EndUser
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(principalCreate.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
		}
		principalCreate.PasswordHash = string(passwordHash)

		principal, err := s.PrincipalService.CreatePrincipal(ctx, principalCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, "User already exists")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create principal").SetInternal(err)
		}
		// Assign Developer role to the just created principal
		principal.Role = api.Developer

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create principal response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal", func(c echo.Context) error {
		ctx := context.Background()
		list, err := s.PrincipalService.FindPrincipalList(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal list").SetInternal(err)
		}

		filteredList := []*api.Principal{}
		for _, principal := range list {
			if err := s.composePrincipalRole(ctx, principal); err != nil {
				// Normally this should not happen since we create the member together with the principal
				// and we don't allow deleting the member. Just in case.
				if common.ErrorCode(err) == common.NotFound {
					s.l.Error("Principal has not been assigned a role. Skip",
						zap.Int("id", principal.ID),
						zap.String("name", principal.Name),
					)
					continue
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch role for principal: %v", principal.Name)).SetInternal(err)
			}
			filteredList = append(filteredList, principal)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal principal list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal/:principalID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("principalID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("principalID"))).SetInternal(err)
		}

		principal, err := s.composePrincipalByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
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

	g.PATCH("/principal/:principalID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("principalID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("principalID"))).SetInternal(err)
		}

		principalPatch := &api.PrincipalPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch principal request").SetInternal(err)
		}
		if principalPatch.Password != nil && *principalPatch.Password != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(*principalPatch.Password), bcrypt.DefaultCost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
			}
			passwordHashStr := string(passwordHash)
			principalPatch.PasswordHash = &passwordHashStr
		}

		principal, err := s.PrincipalService.PatchPrincipal(ctx, principalPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch principal ID: %v", id)).SetInternal(err)
		}
		if err := s.composePrincipalRole(ctx, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch role for principal: %v", principal.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composePrincipalByID(ctx context.Context, id int) (*api.Principal, error) {
	principalFind := &api.PrincipalFind{
		ID: &id,
	}
	principal, err := s.PrincipalService.FindPrincipal(ctx, principalFind)
	if err != nil {
		return nil, err
	}
	if id > 0 && principal == nil {
		return nil, fmt.Errorf("principal not found for ID %v", id)
	}

	if err = s.composePrincipalRole(ctx, principal); err != nil {
		return nil, err
	}

	return principal, nil
}

func (s *Server) composePrincipalRole(ctx context.Context, principal *api.Principal) error {
	if principal.ID == api.SystemBotID {
		principal.Role = api.Owner
	} else {
		memberFind := &api.MemberFind{
			PrincipalID: &principal.ID,
		}
		member, err := s.MemberService.FindMember(ctx, memberFind)
		if err != nil {
			return err
		}
		if principal.ID > 0 && member == nil {
			return fmt.Errorf("Member not found for ID %v", principal.ID)
		}
		principal.Role = member.Role
	}
	return nil
}
