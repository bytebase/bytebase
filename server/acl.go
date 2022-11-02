package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/pkg/errors"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

const (
	roleContextKey = "role"
)

func getRoleContextKey() string {
	return roleContextKey
}

func enforceWorkspaceDeveloperProjectACL(path string, method string, quaryParams url.Values, principalID int) *echo.HTTPError {
	if strings.HasPrefix(path, "/project") {
		if path == "/project" {
			// Developer can only fetch projects from herself.
			if method == "GET" {
				userIDStr := quaryParams.Get("user")
				if userIDStr == "" {
					return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
						errors.Errorf("Not allowed to fetch all project list"))
				}
				if strconv.Itoa(principalID) != userIDStr {
					return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
						errors.Errorf("Not allowed to fetch projects from other user"))
				}
			}
		}
	}
	return nil
}

func aclMiddleware(s *Server, ce *casbin.Enforcer, next echo.HandlerFunc, readonly bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		// Skips auth, actuator, plan
		if common.HasPrefixes(c.Path(), "/api/auth", "/api/actuator", "/api/plan", "/api/oauth") {
			return next(c)
		}

		method := c.Request().Method
		// Skip GET /subscription request
		if common.HasPrefixes(c.Path(), "/api/subscription") && method == "GET" {
			return next(c)
		}
		// Skip OpenAPI request
		if common.HasPrefixes(c.Path(), openAPIPrefix) {
			return next(c)
		}

		if readonly && method != "GET" {
			return echo.NewHTTPError(http.StatusMethodNotAllowed, "Server is in readonly mode")
		}

		// Gets principal id from the context.
		principalID := c.Get(getPrincipalIDContextKey()).(int)

		member, err := s.store.GetMemberByPrincipalID(ctx, principalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}
		if member == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalID))
		}
		if member.RowStatus == api.Archived {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		role := member.Role
		// If admin feature is not enabled, then we treat all user as OWNER.
		if !s.feature(api.FeatureRBAC) {
			role = api.Owner
		}

		// Performs the ACL check.
		pass, err := ce.Enforce(string(role), path, method)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if !pass {
			// If the request is trying to GET/PATCH/DELETE itself, we will change the method signature to
			// XXX_SELF and try again. Because XXX is a superset of XXX_SELF, thus we only try XXX_SELF after
			// XXX fails.
			if method == "GET" || method == "PATCH" || method == "DELETE" {
				if isSelf, err := isOperatingSelf(ctx, c, s, principalID, method); err != nil {
					return err
				} else if isSelf {
					method += "_SELF"

					// Performs the ACL check with _SELF.
					pass, err = ce.Enforce(string(role), path, method)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
					}
				}
			}
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
				errors.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalID, role))
		}

		// Workspace Owner or DBA assumes project Owner role for all projects, so will
		// pass any project ACL.
		if role != api.Owner && role != api.DBA {
			projectACLErr := enforceWorkspaceDeveloperProjectACL(path, method, c.QueryParams(), principalID)
			if projectACLErr != nil {
				return projectACLErr
			}
		}

		// Stores role into context.
		c.Set(getRoleContextKey(), role)

		return next(c)
	}
}

func isOperatingSelf(ctx context.Context, c echo.Context, s *Server, curPrincipalID int, method string) (bool, error) {
	switch method {
	case http.MethodGet:
		return isGettingSelf(ctx, c, s, curPrincipalID)
	case http.MethodPatch, http.MethodDelete:
		return isUpdatingSelf(ctx, c, s, curPrincipalID)
	default:
		return false, nil
	}
}

func isGettingSelf(_ context.Context, c echo.Context, _ *Server, curPrincipalID int) (bool, error) {
	if strings.HasPrefix(c.Path(), "/api/inbox/user") {
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		return userID == curPrincipalID, nil
	} else if strings.HasPrefix(c.Path(), "/api/bookmark/user") {
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		return userID == curPrincipalID, nil
	}

	return false, nil
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
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Activity ID is not a number: %s"+activityIDStr)).SetInternal(err)
			}

			activity, err := s.store.GetActivityByID(ctx, activityID)
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
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Bookmark ID is not a number: %s"+bookmarkIDStr)).SetInternal(err)
			}

			bookmark, err := s.store.GetBookmarkByID(ctx, bookmarkID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if bookmark == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Bookmark ID not found: %d", bookmarkID)).SetInternal(err)
			}

			return bookmark.CreatorID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/inbox") {
		if inboxIDStr := c.Param("inboxID"); inboxIDStr != "" {
			inboxID, err := strconv.Atoi(inboxIDStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Inbox ID is not a number: %s", inboxIDStr)).SetInternal(err)
			}

			inbox, err := s.store.GetInboxByID(ctx, inboxID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if inbox == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Inbox ID not found: %d", inboxID)).SetInternal(err)
			}

			return inbox.ReceiverID == curPrincipalID, nil
		}
	} else if strings.HasPrefix(c.Path(), "/api/sheet") {
		if idStr := c.Param("id"); idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Sheet ID is not a number: %s", idStr)).SetInternal(err)
			}

			sheetFind := &api.SheetFind{
				ID: &id,
			}
			sheet, err := s.store.GetSheet(ctx, sheetFind, curPrincipalID)
			if err != nil {
				return false, echo.NewHTTPError(http.StatusInternalServerError, defaultErrMsg).SetInternal(err)
			}
			if sheet == nil {
				return false, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Sheet ID not found: %d", id)).SetInternal(err)
			}

			return sheet.CreatorID == curPrincipalID, nil
		}
	}

	return false, nil
}
