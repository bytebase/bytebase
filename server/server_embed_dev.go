//go:build !release
// +build !release

package server

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func embedFrontend(logger *zap.Logger, _ *echo.Echo) {
	logger.Info("Dev mode, skip embedding frontend")
}
