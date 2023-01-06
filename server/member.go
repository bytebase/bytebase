package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/store"
)

func (s *Server) registerMemberRoutes(g *echo.Group) {
	g.POST("/member", func(c echo.Context) error {
		ctx := c.Request().Context()
		memberCreate := &api.MemberCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create member request").SetInternal(err)
		}
		updaterID := c.Get(getPrincipalIDContextKey()).(int)

		user, err := s.store.UpdateUser(ctx, memberCreate.PrincipalID, &store.UpdateUserMessage{Role: &memberCreate.Role}, updaterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create member").SetInternal(err)
		}

		// Record activity
		{
			user, err := s.store.GetPrincipalByID(ctx, user.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", user.ID)).SetInternal(err)
			}
			if user == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", user.ID))
			}

			bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
				PrincipalID:    user.ID,
				PrincipalName:  user.Name,
				PrincipalEmail: user.Email,
				MemberStatus:   api.Active,
				Role:           user.Role,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: user.ID,
				Type:        api.ActivityMemberCreate,
				Level:       api.ActivityInfo,
				Payload:     string(bytes),
			}
			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after updating user role: %d", user.ID)).SetInternal(err)
			}
		}

		member, err := s.store.GetMemberByPrincipalID(ctx, user.ID)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create member response").SetInternal(err)
		}
		return nil
	})

	g.GET("/member", func(c echo.Context) error {
		ctx := c.Request().Context()
		memberFind := &api.MemberFind{}
		memberList, err := s.store.FindMember(ctx, memberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch member list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, memberList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal member list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/member/:memberID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}

		member, err := s.store.GetMemberByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find member ID: %d", id)).SetInternal(err)
		}
		if member == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find member ID: %d", id))
		}

		memberPatch := &api.MemberPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch member request").SetInternal(err)
		}
		// When archiving an owner, make sure there are other active owners.
		if member.Role == api.Owner && memberPatch.RowStatus != nil && *memberPatch.RowStatus == string(api.Archived) {
			countResult, err := s.store.CountMemberGroupByRoleAndStatus(ctx)
			for _, count := range countResult {
				if count.Role == api.Owner && count.RowStatus == api.Normal && count.Count == 1 {
					return echo.NewHTTPError(http.StatusInternalServerError, "Cannot archive the only remaining owner in workspace").SetInternal(err)
				}
			}
		}

		updatedMember, err := s.store.PatchMember(ctx, memberPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch member ID: %v", id)).SetInternal(err)
		}

		// Record activity
		{
			user, err := s.store.GetPrincipalByID(ctx, updatedMember.PrincipalID)
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
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
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
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
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
