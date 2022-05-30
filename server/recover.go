package server

import (
	"fmt"

	"github.com/bytebase/bytebase/common/log"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func recoverMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				log.Error("Middleware PANIC RECOVER", zap.Error(err), zap.Stack("stack"))

				c.Error(err)
			}
		}()
		return next(c)
	}
}
