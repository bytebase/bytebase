package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func errorRecorderMiddleware(s *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		defer func() {
			responseCode := c.Response().Status

			if responseCode == http.StatusInternalServerError {
				// TODO(ZhengX): collect and store error details
			}
		}()

		if err := next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}
