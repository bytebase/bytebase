package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func AclMiddleware(l *bytebase.Logger, m api.MemberService, ce *casbin.Enforcer, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth end point
		if strings.HasPrefix(c.Path(), "/api/auth") {
			return next(c)
		}

		// Gets principal id from the context.
		principalId := c.Get(GetPrincipalIdContextKey()).(int)

		member, err := m.FindMemberByPrincipalID(context.Background(), api.DEFAULT_WORKPSACE_ID, principalId)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User ID is not a member: %d", principalId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authorize.").SetInternal(err)
		}

		method := c.Request().Method
		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		pass, err := ce.Enforce(member.Role.String(), path, method)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authorize.").SetInternal(err)
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(
				fmt.Errorf("rejected by the ACL policy; %s %s u%d/%s", method, path, principalId, member.Role))
		}

		return next(c)
	}
}
