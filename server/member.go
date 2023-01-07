package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
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

		composedPrincipal, err := s.store.GetPrincipalByID(ctx, user.ID)
		if err != nil {
			return err
		}
		member := convertMember(user, composedPrincipal)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create member response").SetInternal(err)
		}
		return nil
	})

	g.GET("/member", func(c echo.Context) error {
		ctx := c.Request().Context()
		users, err := s.store.ListUsers(ctx, &store.FindUserMessage{ShowDeleted: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user list").SetInternal(err)
		}

		var members []*api.Member
		for _, user := range users {
			if user.ID == api.SystemBotID {
				continue
			}
			composedPrincipal, err := s.store.GetPrincipalByID(ctx, user.ID)
			if err != nil {
				return err
			}
			member := convertMember(user, composedPrincipal)
			members = append(members, member)
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, members); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal member list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/member/:memberID", func(c echo.Context) error {
		ctx := c.Request().Context()
		// memberID is the same as user ID.
		id, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}
		updaterID := c.Get(getPrincipalIDContextKey()).(int)
		user, err := s.store.GetUserByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Server error to find user ID: %d", id)).SetInternal(err)
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Failed to find user ID: %d", id))
		}

		memberPatch := &api.MemberPatch{
			ID: id,
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, memberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch member request").SetInternal(err)
		}

		// When archiving an owner, make sure there are other active owners.
		if user.Role == api.Owner && memberPatch.RowStatus != nil && *memberPatch.RowStatus == string(api.Archived) {
			countResult, err := s.store.CountMemberGroupByRoleAndStatus(ctx)
			for _, count := range countResult {
				if count.Role == api.Owner && count.RowStatus == api.Normal && count.Count == 1 {
					return echo.NewHTTPError(http.StatusInternalServerError, "Cannot archive the only remaining owner in workspace").SetInternal(err)
				}
			}
		}
		oldRole := user.Role

		update := &store.UpdateUserMessage{}
		if memberPatch.Role != nil {
			role := api.Role(*memberPatch.Role)
			update.Role = &role
		}
		if memberPatch.RowStatus != nil {
			rowStatus := api.RowStatus(*memberPatch.RowStatus)
			if rowStatus == api.Normal {
				f := false
				update.Delete = &f
			}
			if rowStatus == api.Archived {
				t := true
				update.Delete = &t
			}
		}
		user, err = s.store.UpdateUser(ctx, user.ID, update, updaterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update user ID: %v", id)).SetInternal(err)
		}

		// Record activity
		{
			if memberPatch.Role != nil {
				bytes, err := json.Marshal(api.ActivityMemberRoleUpdatePayload{
					PrincipalID:    user.ID,
					PrincipalName:  user.Name,
					PrincipalEmail: user.Email,
					OldRole:        oldRole,
					NewRole:        user.Role,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
				}
				activityCreate := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: user.ID,
					Type:        api.ActivityMemberRoleUpdate,
					Level:       api.ActivityInfo,
					Payload:     string(bytes),
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", user.ID)).SetInternal(err)
				}
			} else if memberPatch.RowStatus != nil {
				bytes, err := json.Marshal(api.ActivityMemberActivateDeactivatePayload{
					PrincipalID:    user.ID,
					PrincipalName:  user.Name,
					PrincipalEmail: user.Email,
					Role:           user.Role,
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
					ContainerID: user.ID,
					Type:        theType,
					Level:       api.ActivityInfo,
					Payload:     string(bytes),
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after changing member role: %d", user.ID)).SetInternal(err)
				}
			}
		}

		composedPrincipal, err := s.store.GetPrincipalByID(ctx, user.ID)
		if err != nil {
			return err
		}
		member := convertMember(user, composedPrincipal)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, member); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal member ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func convertMember(user *store.UserMessage, composedPrincipal *api.Principal) *api.Member {
	member := &api.Member{
		ID:          user.ID,
		RowStatus:   api.Normal,
		Status:      api.Active,
		Role:        user.Role,
		PrincipalID: user.ID,
		Principal:   composedPrincipal,
	}
	if user.MemberDeleted {
		member.RowStatus = api.Archived
	}
	return member
}
