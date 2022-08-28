package server

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/bytebase/bytebase/api"

	"github.com/labstack/echo/v4"
)

func errorRecorderMiddleware(err error, s *Server, c echo.Context, e *echo.Echo) {
	e.DefaultHTTPErrorHandler(err, c)

	req := c.Request()
	res := c.Response()

	he, isHTTPError := err.(*echo.HTTPError)

	if res.Status == http.StatusInternalServerError {
		var role api.Role
		if r, ok := c.Get(getRoleContextKey()).(api.Role); ok {
			role = r
		}

		var stackTrace string
		// There're basically two kinds of errors: internal HTTP errors and runtime panics.
		// If we encounter runtime panics, we can report the stack trace for debugging.
		if !isHTTPError || he.Internal == nil {
			// Deal with runtime panics.
			// Since we don't know where future panics will occur, we will manually record the stack trace here
			// so that we can handle all possible panics.
			stackTrace = string(debug.Stack())
		}

		s.errorRecordRing.Mutex.Lock()
		defer s.errorRecordRing.Mutex.Unlock()

		s.errorRecordRing.Ring.Value = &api.ErrorRecord{
			RecordTs:    time.Now().Unix(),
			Method:      req.Method,
			RequestPath: req.URL.Path,
			Role:        role,
			Error:       err.Error(),
			StackTrace:  stackTrace,
		}
		s.errorRecordRing.Ring = s.errorRecordRing.Ring.Next()
	}
}
