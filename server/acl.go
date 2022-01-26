package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/labstack/echo/v4"
)

const (
	roleContextKey = "role"
)

func getRoleContextKey() string {
	return roleContextKey
}

func aclMiddleware(l *zap.Logger, s *Server, ce *casbin.Enforcer, next echo.HandlerFunc, readonly bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.Background()
		// Skips auth, actuator, plan
		if common.HasPrefixes(c.Path(), "/api/auth", "/api/actuator", "/api/plan") {
			return next(c)
		}

		method := c.Request().Method
		if readonly && method != "GET" {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, "Server is in readonly mode")
		}

		// Gets principal id from the context.
		principalID := c.Get(getPrincipalIDContextKey()).(int)

		memberFind := &api.MemberFind{
			PrincipalID: &principalID,
		}
		member, err := s.MemberService.FindMember(ctx, memberFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if member == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalID))
		}
		if member.RowStatus == api.Archived {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		// If the requests is trying to PATCH/DELETE herself, we will change the method signature to
		// XXX_SELF so that the policy can differentiate between XXX and XXX_SELF
		if method == "PATCH" || method == "DELETE" {
			if isSelf, err := isUpdatingSelf(ctx, c, s, principalID); err != nil {
				return err
			} else if isSelf {
				method = method + "_SELF"
			}
		}

		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		role := member.Role
		// If admin feature is not enabled, then we treat all user as OWNER.
		if !s.feature("bb.feature.rbac") {
			role = api.Owner
		}
		// Performs the ACL check.
		pass, err := ce.Enforce(role.String(), path, method)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
				fmt.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalID, role))
		}

		// Stores role into context.
		c.Set(getRoleContextKey(), role)

		return next(c)
	}
}

func isUpdatingSelf(ctx context.Context, c echo.Context, s *Server, curPrincipalID int) (bool, error) {
	const defaultErrMsg = "Failed to process authorize request."
	if strings.HasPrefix(c.Path(), "/api/principal") {
		pathPrincipalID := c.Param("principalID")
		if pathPrincipalID != "" {
			return pathPrincipalID == strconv.Itoa(curPrincipalID), nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/activity") {
		if activityIDStr := c.Param("activityID"); activityIDStr != "" {
			activityID, err := strconv.Atoi(activityIDStr)
			if err != nil {
				msg := "Activity ID is not a number: " + activityIDStr
				httpErr := echo.NewHTTPError(http.StatusBadRequest, msg)
				return false, httpErr
			}
			activityFind := &api.ActivityFind{
				ID: &activityID,
			}
			activity, err := s.ActivityService.FindActivity(ctx, activityFind)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if activity == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Activity ID not found: %d", activityID))
			}
			return activity.CreatorID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/bookmark") {
		if bookmarkIDStr := c.Param("bookmarkID"); bookmarkIDStr != "" {
			bookmarkID, err := strconv.Atoi(bookmarkIDStr)
			if err != nil {
				msg := "Bookmark ID is not a number: " + bookmarkIDStr
				httpErr := echo.NewHTTPError(http.StatusBadRequest, msg)
				return false, httpErr
			}
			bookmarkFind := &api.BookmarkFind{
				ID: &bookmarkID,
			}
			bookmark, err := s.BookmarkService.FindBookmark(ctx, bookmarkFind)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if bookmark == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", bookmarkID))
			}
			return bookmark.CreatorID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/inbox") {
		if inboxIDStr := c.Param("inboxID"); inboxIDStr != "" {
			inboxID, err := strconv.Atoi(inboxIDStr)
			if err != nil {
				msg := "Inbox ID is not a number: " + inboxIDStr
				httpErr := echo.NewHTTPError(http.StatusBadRequest, msg)
				return false, httpErr
			}
			inboxFind := &api.InboxFind{
				ID: &inboxID,
			}
			inbox, err := s.InboxService.FindInbox(ctx, inboxFind)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if inbox == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Inbox ID not found: %d", inboxID))
			}
			return inbox.ReceiverID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/savedquery") {
		if idStr := c.Param("id"); idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Saved query ID is not a number: %s", idStr))
			}
			savedQueryFind := &api.SavedQueryFind{
				ID: &id,
			}
			savedQuery, err := s.SavedQueryService.FindSavedQuery(ctx, savedQueryFind)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if savedQuery == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Saved query ID not found: %d", id))
			}
			return savedQuery.CreatorID == curPrincipalID, nil
		}
	}
	return false, nil
}
