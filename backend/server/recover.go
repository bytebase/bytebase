package server

import (
	"fmt"
	"runtime"

	"github.com/bytebase/bytebase"
	"github.com/labstack/echo/v4"
)

const (
	STACK_SIZE = 4 << 10 // 4 KB
	ALL_STACK  = true
)

func RecoverMiddleware(l *bytebase.Logger, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				stack := make([]byte, STACK_SIZE)
				length := runtime.Stack(stack, ALL_STACK)
				msg := fmt.Sprintf("[Middleware PANIC RECOVER] %v %s\n", err, stack[:length])

				l.Error(msg)

				c.Error(err)
			}
		}()
		return next(c)
	}
}
