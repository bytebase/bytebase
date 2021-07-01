package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
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
		// Skips auth, actuator and plan
		if strings.HasPrefix(c.Path(), "/api/auth") || strings.HasPrefix(c.Path(), "/api/actuator") || strings.HasPrefix(c.Path(), "/api/plan") {
			return next(c)
		}

		method := c.Request().Method
		if readonly && method != "GET" {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, "Server is in readonly mode")
		}

		// Gets principal id from the context.
		principalId := c.Get(GetPrincipalIdContextKey()).(int)

		memberFind := &api.MemberFind{
			PrincipalId: &principalId,
		}
		member, err := s.MemberService.FindMember(context.Background(), memberFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		// If the requests is trying to PATCH/DELETE herself, we will change the method signature to
		// XXX_SELF so that the policy can differentiate between XXX and XXX_SELF
		if method == "PATCH" || method == "DELETE" {
			if strings.HasPrefix(c.Path(), "/api/principal") {
				pathPrincipalId := c.Param("principalId")
				if pathPrincipalId != "" {
					if pathPrincipalId == strconv.Itoa(principalId) {
						method = method + "_SELF"
					}
				}
			} else if strings.HasPrefix(c.Path(), "/api/activity") {
				activityIdStr := c.Param("activityId")
				if activityIdStr != "" {
					activityId, err := strconv.Atoi(activityIdStr)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Activity ID is not a number: %s", activityIdStr))
					}
					activityFind := &api.ActivityFind{
						ID: &activityId,
					}
					activity, err := s.ActivityService.FindActivity(context.Background(), activityFind)
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Activity ID not found: %d", activityId))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
					if activity.CreatorId == principalId {
						method = method + "_SELF"
					}
				}
			} else if strings.HasPrefix(c.Path(), "/api/bookmark") {
				bookmarkIdStr := c.Param("bookmarkId")
				if bookmarkIdStr != "" {
					bookmarkId, err := strconv.Atoi(bookmarkIdStr)
					if err != nil {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bookmark ID is not a number: %s", bookmarkIdStr))
					}
					bookmarkFind := &api.BookmarkFind{
						ID: &bookmarkId,
					}
					bookmark, err := s.BookmarkService.FindBookmark(context.Background(), bookmarkFind)
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Bookmark ID not found: %d", bookmarkId))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
					if bookmark.CreatorId == principalId {
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
				fmt.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalId, role))
		}

		// Stores role into context.
		c.Set(GetRoleContextKey(), role)

		return next(c)
	}
}
