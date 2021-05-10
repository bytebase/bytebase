package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"

	"github.com/bytebase/bytebase"
	"github.com/labstack/echo/v4"
)

func AclMiddleware(l *bytebase.Logger, ce *casbin.Enforcer, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skips auth end point
		if strings.HasPrefix(c.Path(), "/api/auth") {
			return next(c)
		}

		// Gets principal id from the context.
		user := strconv.Itoa(c.Get(GetPrincipalIdContextKey()).(int))

		method := c.Request().Method
		path := strings.TrimPrefix(c.Request().URL.Path, "/api")

		pass, err := ce.Enforce(user, path, method)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authorize.").SetInternal(err)
		}

		if !pass {
			return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(fmt.Errorf("rejected by the ACL policy; user: %s, path: %s, method: %s", user, path, method))
		}

		return next(c)
	}
}
