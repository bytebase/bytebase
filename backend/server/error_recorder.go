package server

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

func errorRecorderMiddleware(err error, s *Server, c echo.Context, e *echo.Echo) {
	e.DefaultHTTPErrorHandler(err, c)

	he, isHTTPError := err.(*echo.HTTPError)

	if c.Response().Status == http.StatusInternalServerError {
		var role api.Role
		if r, ok := c.Get(getRoleContextKey()).(api.Role); ok {
			role = r
		}

		var stackTrace string
		// There're basically two kinds of errors: internal HTTP errors and runtime panics.
		// If we encounter runtime panics, we can report the stack trace for debugging.
		if !isHTTPError || he.Internal == nil {
			// Get a snapshot of the stack trace here, which will include where the panic occurred.
			// We only do this for runtime errors. Because for internal HTTP errors,
			// the stack trace here won't indicate where the internal error is from.
			// To debug internal errors, we mainly check the error mesasges.
			stackTrace = string(debug.Stack())
		}

		s.errorRecordRing.RWMutex.Lock()
		defer s.errorRecordRing.RWMutex.Unlock()

		s.errorRecordRing.Ring.Value = &api.ErrorRecord{
			RecordTs:    time.Now().Unix(),
			Method:      c.Request().Method,
			RequestPath: c.Request().URL.Path,
			Role:        role,
			Error:       err.Error(),
			StackTrace:  stackTrace,
		}
		s.errorRecordRing.Ring = s.errorRecordRing.Ring.Next()
	}
}
