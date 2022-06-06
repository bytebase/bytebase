//go:build !release
// +build !release

package server

import (
	"github.com/bytebase/bytebase/common/log"
	"github.com/labstack/echo/v4"
)

func embedFrontend(_ *echo.Echo) {
	log.Info("Dev mode, skip embedding frontend")
}
