package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func ACLMiddleware(l *bytebase.Logger, m api.MemberService, ce *casbin.Enforcer, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth end point
		if strings.HasPrefix(c.Path(), "/api/auth") {
			return next(c)
		}

		// Gets principal id from the context.
		principalId := c.Get(GetPrincipalIdContextKey()).(int)

		wsId := api.DEFAULT_WORKPSACE_ID
		memberFilter := &api.MemberFilter{
			WorkspaceId: &wsId,
			PrincipalId: &principalId,
		}
		member, err := m.FindMember(context.Background(), memberFilter)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		method := c.Request().Method
		// If the requests is trying to PATCH herself, we will change the method signature to
		// PATCH_SELF so that the policy can differentiate between PATCH and PATCH_SELF
		if method == "PATCH" {
			pathPrincipalId := c.Param("principalId")
			if pathPrincipalId != "" {
				if pathPrincipalId == strconv.Itoa(principalId) {
					method = "PATCH_SELF"
				}
			}
		}

		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		// Performs the ACL check.
		pass, err := ce.Enforce(member.Role.String(), path, method)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process authorize request.").SetInternal(err)
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
				fmt.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalId, member.Role))
		}

		return next(c)
	}
}
