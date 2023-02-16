package server

import (
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
)

func defaultAPIRequestSkipper(c echo.Context) bool {
	path := c.Path()
	return common.HasPrefixes(path, "/api", "/v1")
}
