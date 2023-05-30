// Package fake provides the fake implementation for several dependency services.
package fake

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func newEchoServer() *echo.Echo {
	e := echo.New()
	e.Debug = false
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	return e
}
