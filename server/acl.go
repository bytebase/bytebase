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

func GetRoleContextKey() string {
	return roleContextKey
}

func ACLMiddleware(l *zap.Logger, s *Server, ce *casbin.Enforcer, next echo.HandlerFunc, readonly bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.Background()
		// Skips auth, actuator, plan
		if strings.HasPrefix(c.Path(), "/api/auth") || strings.HasPrefix(c.Path(), "/api/actuator") || strings.HasPrefix(c.Path(), "/api/plan") {
			return next(c)
		}

		method := c.Request().Method
		if readonly && method != "GET" {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, "Server is in readonly mode")
		}

		// Gets principal id from the context.
		principalID := c.Get(GetPrincipalIDContextKey()).(int)

		memberFind := &api.MemberFind{
			PrincipalID: &principalID,
		}
		member, err := s.MemberService.FindMember(ctx, memberFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if member.RowStatus == api.Archived {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		// If the requests is trying to PATCH/DELETE herself, we will change the method signature to
		// XXX_SELF so that the policy can differentiate between XXX and XXX_SELF
		if method == "PATCH" || method == "DELETE" {
			if strings.HasPrefix(c.Path(), "/api/principal") {
				pathPrincipalID := c.Param("principalID")
				if pathPrincipalID != "" {
					if pathPrincipalID == strconv.Itoa(principalID) {
						method = method + "_SELF"
					}
				}
			} else if strings.HasPrefix(c.Path(), "/api/activity") {
				activityIDStr := c.Param("activityID")
				if activityIDStr != "" {
					activityID, err := strconv.Atoi(activityIDStr)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Activity ID is not a number: %s", activityIDStr))
					}
					activityFind := &api.ActivityFind{
						ID: &activityID,
					}
					activity, err := s.ActivityService.FindActivity(ctx, activityFind)
					if err != nil {
						if common.ErrorCode(err) == common.NotFound {
							return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Activity ID not found: %d", activityID))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
					if activity.CreatorID == principalID {
						method = method + "_SELF"
					}
				}
			} else if strings.HasPrefix(c.Path(), "/api/bookmark") {
				bookmarkIDStr := c.Param("bookmarkID")
				if bookmarkIDStr != "" {
					bookmarkID, err := strconv.Atoi(bookmarkIDStr)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bookmark ID is not a number: %s", bookmarkIDStr))
					}
					bookmarkFind := &api.BookmarkFind{
						ID: &bookmarkID,
					}
					bookmark, err := s.BookmarkService.FindBookmark(ctx, bookmarkFind)
					if err != nil {
						if common.ErrorCode(err) == common.NotFound {
							return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Bookmark ID not found: %d", bookmarkID))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
					if bookmark.CreatorID == principalID {
						method = method + "_SELF"
					}
				}
			} else if strings.HasPrefix(c.Path(), "/api/inbox") {
				inboxIDStr := c.Param("inboxID")
				if inboxIDStr != "" {
					inboxID, err := strconv.Atoi(inboxIDStr)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Inbox ID is not a number: %s", inboxIDStr))
					}
					inboxFind := &api.InboxFind{
						ID: &inboxID,
					}
					inbox, err := s.InboxService.FindInbox(ctx, inboxFind)
					if err != nil {
						if common.ErrorCode(err) == common.NotFound {
							return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Inbox ID not found: %d", inboxID))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
					if inbox.ReceiverID == principalID {
						method = method + "_SELF"
					}
				}
			}
		}

		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		role := member.Role
		// If admin feature is not enabled, then we treat all user as OWNER.
		if !s.feature("bb.admin") {
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
		c.Set(GetRoleContextKey(), role)

		return next(c)
	}
}
