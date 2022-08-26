package server

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func errorRecorderMiddleware(err error, s *Server, c echo.Context, e *echo.Echo) {
	req := c.Request()
	res := c.Response()
	var errorMessage string

	// -----------------------------------Echo's DefaultHTTPErrorHandler BEGIN----------------------------------------

	if res.Committed {
		return
	}

	he, isHTTPError := err.(*echo.HTTPError)

	if isHTTPError {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}
		errorMessage = he.Error()
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: http.StatusText(http.StatusInternalServerError),
		}
		errorMessage = err.Error()
	}

	code := he.Code
	message := he.Message
	if m, ok := he.Message.(string); ok {
		if e.Debug {
			message = echo.Map{"message": m, "error": errorMessage}
		} else {
			message = echo.Map{"message": m}
		}
	}

	// Send response
	if c.Request().Method == http.MethodHead {
		err = c.NoContent(he.Code)
	} else {
		err = c.JSON(code, message)
	}
	if err != nil {
		e.Logger.Error(err)
	}

	// -----------------------------------Echo's DefaultHTTPErrorHandler END------------------------------------------

	if res.Status == http.StatusInternalServerError {
		var role api.Role
		if r, ok := c.Get(getRoleContextKey()).(api.Role); ok {
			role = r
		}

		var stackTrace string
		// Record stackTrace for non-echo-internal errors.
		if !isHTTPError || he.Internal == nil {
			stackTrace = string(debug.Stack())
		}

		s.errorRecordRing.Mutex.Lock()
		defer s.errorRecordRing.Mutex.Unlock()

		s.errorRecordRing.Ring.Value = &api.ErrorRecord{
			RecordTs:    time.Now().Unix(),
			Method:      req.Method,
			RequestPath: req.URL.Path,
			Role:        role,
			Error:       errorMessage,
			StackTrace:  stackTrace,
		}
		s.errorRecordRing.Ring = s.errorRecordRing.Ring.Next()
	}
}
