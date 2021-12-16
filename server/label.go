package server

import (
	"context"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

var ()

func (s *Server) registerLabelRoutes(g *echo.Group) {
	g.GET("/label", func(c echo.Context) error {
		ctx := context.Background()
		find := &api.LabelKeyFind{}
		list, err := s.LabelService.FindLabelKeysList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch label keys").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal label keys response").SetInternal(err)
		}
		return nil
	})
}
