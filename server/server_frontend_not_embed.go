//go:build !embed_frontend
// +build !embed_frontend

package server

import (
	"net/http"

	"github.com/bytebase/bytebase/common/log"
	"github.com/labstack/echo/v4"
)

func embedFrontend(e *echo.Echo) {
	log.Info("Skip embedding frontend, build with 'embed_frontend' tag if you want embedded frontend.")

	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}
