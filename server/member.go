package server

import (
	"context"
	"encoding/json"
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
		memberCreate := &api.MemberCreate{}
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

		// Record activity
		{
			principalFind := &api.PrincipalFind{
				ID: &member.PrincipalId,
			}
			user, err := s.PrincipalService.FindPrincipal(context.Background(), principalFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", member.PrincipalId))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", member.PrincipalId)).SetInternal(err)
			}

			bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
				PrincipalId:    member.PrincipalId,
				PrincipalName:  user.Name,
				PrincipalEmail: user.Email,
				MemberStatus:   member.Status,
				Role:           member.Role,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}
			_, err = s.ActivityService.CreateActivity(context.Background(), &api.ActivityCreate{
				CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
				ContainerId: member.ID,
				Type:        api.ActivityMemberCreate,
				Level:       api.ACTIVITY_INFO,
				Payload:     string(bytes),
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after creating member: %d", member.ID)).SetInternal(err)
			}
		}

		if err := s.ComposeMemberRelationship(context.Background(), member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created member relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create member response").SetInternal(err)
		}
		return nil
	})

	g.GET("/member", func(c echo.Context) error {
		memberFind := &api.MemberFind{}
		list, err := s.MemberService.FindMemberList(context.Background(), memberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch member list").SetInternal(err)
		}

		for _, member := range list {
			if err := s.ComposeMemberRelationship(context.Background(), member); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch member relationship: %v", member.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal member list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/member/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		memberFind := &api.MemberFind{
			ID: &id,
		}
		member, err := s.MemberService.FindMember(context.Background(), memberFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find member ID: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find member ID: %d", id)).SetInternal(err)
		}

		memberPatch := &api.MemberPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch member request").SetInternal(err)
		}

		updatedMember, err := s.MemberService.PatchMember(context.Background(), memberPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch member ID: %v", id)).SetInternal(err)
		}

		// Record activity
		{
			principalFind := &api.PrincipalFind{
				ID: &updatedMember.PrincipalId,
			}
			user, err := s.PrincipalService.FindPrincipal(context.Background(), principalFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", updatedMember.PrincipalId))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", updatedMember.PrincipalId)).SetInternal(err)
			}

			if memberPatch.Role != nil {
				bytes, err := json.Marshal(api.ActivityMemberRoleUpdatePayload{
					PrincipalId:    updatedMember.PrincipalId,
					PrincipalName:  user.Name,
					PrincipalEmail: user.Email,
					OldRole:        member.Role,
					NewRole:        updatedMember.Role,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
				}
				_, err = s.ActivityService.CreateActivity(context.Background(), &api.ActivityCreate{
					CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
					ContainerId: updatedMember.ID,
					Type:        api.ActivityMemberRoleUpdate,
					Level:       api.ACTIVITY_INFO,
					Payload:     string(bytes),
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", updatedMember.ID)).SetInternal(err)
				}
			} else if memberPatch.RowStatus != nil {
				bytes, err := json.Marshal(api.ActivityMemberActivateDeactivatePayload{
					PrincipalId:    updatedMember.PrincipalId,
					PrincipalName:  user.Name,
					PrincipalEmail: user.Email,
					Role:           member.Role,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
				}
				theType := api.ActivityMemberActivate
				if *memberPatch.RowStatus == "ARCHIVED" {
					theType = api.ActivityMemberDeactivate
				}
				_, err = s.ActivityService.CreateActivity(context.Background(), &api.ActivityCreate{
					CreatorId:   c.Get(GetPrincipalIdContextKey()).(int),
					ContainerId: updatedMember.ID,
					Type:        theType,
					Level:       api.ACTIVITY_INFO,
					Payload:     string(bytes),
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", updatedMember.ID)).SetInternal(err)
				}
			}
		}

		if err := s.ComposeMemberRelationship(context.Background(), updatedMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated member relationship: %v", updatedMember.ID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal member ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeMemberRelationship(ctx context.Context, member *api.Member) error {
	var err error

	member.Creator, err = s.ComposePrincipalById(ctx, member.CreatorId)
	if err != nil {
		return err
	}

	member.Updater, err = s.ComposePrincipalById(context.Background(), member.UpdaterId)
	if err != nil {
		return err
	}

	member.Principal, err = s.ComposePrincipalById(context.Background(), member.PrincipalId)
	if err != nil {
		return err
	}

	return nil
}
