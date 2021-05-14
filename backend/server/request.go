package server

import (
	"sort"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/labstack/echo/v4"
)

const (
	includeKey = "include"
)

func getIncludeKey() string {
	return includeKey
}

func RequestMiddleware(l *bytebase.Logger, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		includeList := (strings.Split(c.QueryParams().Get("include"), ","))
		sort.Strings(includeList)
		c.Set(getIncludeKey(), includeList)
		return next(c)
	}
}
