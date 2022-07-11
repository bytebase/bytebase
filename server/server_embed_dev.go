//go:build !release
// +build !release

package server

import (
	"net/http"

	"github.com/bytebase/bytebase/common/log"
	"github.com/labstack/echo/v4"
)

func embedFrontend(e *echo.Echo) {
	log.Info("Dev mode, skip embedding frontend")

	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}
