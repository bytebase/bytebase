package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerMemberRoutes(g *echo.Group) {
	g.POST("/member", func(c echo.Context) error {
		ctx := context.Background()
		memberCreate := &api.MemberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create member request").SetInternal(err)
		}

		memberCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		memberRaw, err := s.MemberService.CreateMember(ctx, memberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Member for user ID already exists: %d", memberCreate.PrincipalID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create member").SetInternal(err)
		}
		member, err := s.composeMemberRelationship(ctx, memberRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create member").SetInternal(err)
		}

		// Record activity
		{
			principalFind := &api.PrincipalFind{
				ID: &member.PrincipalID,
			}
			user, err := s.PrincipalService.FindPrincipal(ctx, principalFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", member.PrincipalID)).SetInternal(err)
			}
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", member.PrincipalID))
			}

			bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
				PrincipalID:    member.PrincipalID,
				PrincipalName:  user.Name,
				PrincipalEmail: user.Email,
				MemberStatus:   member.Status,
				Role:           member.Role,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: member.ID,
				Type:        api.ActivityMemberCreate,
				Level:       api.ActivityInfo,
				Payload:     string(bytes),
			}
			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after creating member: %d", member.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create member response").SetInternal(err)
		}
		return nil
	})

	g.GET("/member", func(c echo.Context) error {
		ctx := context.Background()
		memberFind := &api.MemberFind{}
		memberRawList, err := s.MemberService.FindMemberList(ctx, memberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch member list").SetInternal(err)
		}
		var memberList []*api.Member
		for _, memberRaw := range memberRawList {
			member, err := s.composeMemberRelationship(ctx, memberRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch member relationship: %v", memberRaw.ID)).SetInternal(err)
			}
			memberList = append(memberList, member)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, memberList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal member list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/member/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		memberFind := &api.MemberFind{
			ID: &id,
		}
		memberRaw, err := s.MemberService.FindMember(ctx, memberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find member ID: %d", id)).SetInternal(err)
		}
		if memberRaw == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find member ID: %d", id))
		}
		member, err := s.composeMemberRelationship(ctx, memberRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose member relationship with ID %d", id)).SetInternal(err)
		}

		memberPatch := &api.MemberPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch member request").SetInternal(err)
		}

		updatedMemberRaw, err := s.MemberService.PatchMember(ctx, memberPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch member ID: %v", id)).SetInternal(err)
		}
		updatedMember, err := s.composeMemberRelationship(ctx, updatedMemberRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose member relationship with ID %v", id)).SetInternal(err)
		}

		// Record activity
		{
			principalFind := &api.PrincipalFind{
				ID: &updatedMember.PrincipalID,
			}
			user, err := s.PrincipalService.FindPrincipal(ctx, principalFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", updatedMember.PrincipalID)).SetInternal(err)
			}
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", updatedMember.PrincipalID))
			}

			if memberPatch.Role != nil {
				bytes, err := json.Marshal(api.ActivityMemberRoleUpdatePayload{
					PrincipalID:    updatedMember.PrincipalID,
					PrincipalName:  user.Name,
					PrincipalEmail: user.Email,
					OldRole:        member.Role,
					NewRole:        updatedMember.Role,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
				}
				activityCreate := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: updatedMember.ID,
					Type:        api.ActivityMemberRoleUpdate,
					Level:       api.ActivityInfo,
					Payload:     string(bytes),
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", updatedMember.ID)).SetInternal(err)
				}
			} else if memberPatch.RowStatus != nil {
				bytes, err := json.Marshal(api.ActivityMemberActivateDeactivatePayload{
					PrincipalID:    updatedMember.PrincipalID,
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
				activityCreate := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: updatedMember.ID,
					Type:        theType,
					Level:       api.ActivityInfo,
					Payload:     string(bytes),
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", updatedMember.ID)).SetInternal(err)
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal member ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeMemberRelationship(ctx context.Context, raw *api.MemberRaw) (*api.Member, error) {
	member := raw.ToMember()

	creator, err := s.composePrincipalByID(ctx, member.CreatorID)
	if err != nil {
		return nil, err
	}
	member.Creator = creator

	updater, err := s.composePrincipalByID(ctx, member.UpdaterID)
	if err != nil {
		return nil, err
	}
	member.Updater = updater

	principal, err := s.composePrincipalByID(ctx, member.PrincipalID)
	if err != nil {
		return nil, err
	}
	member.Principal = principal

	return member, nil
}
