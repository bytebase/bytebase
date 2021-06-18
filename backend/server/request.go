package server

import (
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	includeKey = "include"
)

func getIncludeKey() string {
	return includeKey
}

func ApiRequestMiddleware(l *zap.Logger, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		includeList := (strings.Split(c.QueryParams().Get("include"), ","))
		c.Set(getIncludeKey(), includeList)
		return next(c)
	}
}
