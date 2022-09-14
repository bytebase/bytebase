package server

import (
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common/log"
)

func (s *Server) registerDebugRoutes(g *echo.Group) {
	g.GET("/debug", currentDebugState)

	g.PATCH("/debug", func(c echo.Context) error {
		var debugPatch api.DebugPatch
		if err := jsonapi.UnmarshalPayload(c.Request().Body, &debugPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to unmarshal debug patch request").SetInternal(err)
		}

		lvl := zap.InfoLevel
		if debugPatch.IsDebug {
			lvl = zap.DebugLevel
		}
		log.SetLevel(lvl)

		s.e.Debug = debugPatch.IsDebug

		return currentDebugState(c)
	})

	g.GET("/debug/log", func(c echo.Context) error {
		var errorRecordList []*api.DebugLog
		// incrementID is used as primary key in jsonapi.
		var incrementID int

		// Only Owner and DBA can see debug logs.
		role := c.Get(getRoleContextKey()).(api.Role)
		if role != api.Owner && role != api.DBA {
			return echo.NewHTTPError(http.StatusForbidden, "Not allowed to fetch debug logs")
		}

		s.errorRecordRing.RWMutex.RLock()
		defer s.errorRecordRing.RWMutex.RUnlock()

		s.errorRecordRing.Ring.Do(func(p interface{}) {
			if p == nil {
				return
			}
			errRecord, ok := p.(*api.ErrorRecord)
			if !ok {
				return
			}
			errorRecordList = append(errorRecordList, &api.DebugLog{
				ID:          incrementID,
				ErrorRecord: *errRecord,
			})
			incrementID++
		})

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, errorRecordList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal debug log response").SetInternal(err)
		}

		return nil
	})
}

func currentDebugState(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	if err := jsonapi.MarshalPayload(c.Response().Writer, &api.Debug{IsDebug: log.EnabledLevel(zap.DebugLevel)}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal debug info response").SetInternal(err)
	}
	return nil
}
