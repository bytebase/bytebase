package server

import (
	"net/http"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func errorRecorderMiddleware(s *Server, next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		defer func() {
			if c.Response().Status == http.StatusInternalServerError {
				req := c.Request()
				// get error message and stack trace if any
				errorMessage := c.Get("errorMessage")
				stackTrace := c.Get("panicStackTrace")

				defer func() {
					c.Set("errorMessage", nil)
					c.Set("panicStackTrace", nil)
				}()

				if errorMessage == nil {
					errorMessage = ""
				}
				if stackTrace == nil {
					stackTrace = ""
				}

				var role api.Role
				if r, ok := c.Get(getRoleContextKey()).(api.Role); !ok {
					role = api.Role(api.Unknown)
				} else {
					role = r
				}

				s.errorRecordMu.Lock()
				s.errorRecordRing.Value = &api.ErrorRecord{
					RecordTs:    time.Now().Format("2006-01-02T15:04:05 -07:00:00"),
					Method:      req.Method,
					RequestPath: req.URL.Path,
					Role:        role,
					Error:       errorMessage.(string),
					StackTrace:  stackTrace.(string),
				}
				s.errorRecordRing = s.errorRecordRing.Next()
				s.errorRecordMu.Unlock()
			}
		}()

		return next(c)
	}
}
