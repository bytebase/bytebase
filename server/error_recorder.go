package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func errorRecorderMiddleware(_ *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		defer func() {
			responseCode := c.Response().Status

			if responseCode == http.StatusInternalServerError { //nolint
				// TODO(ZhengX): collect and store error details
			}
		}()

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}
