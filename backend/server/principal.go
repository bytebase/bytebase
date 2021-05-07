package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerPrincipalRoutes(g *echo.Group) {
	g.GET("/principal", func(c echo.Context) error {
		list, err := s.PrincipalService.FindPrincipalList(context.Background())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed fetch principal list.")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal principal list.")
		}

		return nil
	})

	g.GET("/principal/:id", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s.", c.Param("id")))
		}

		principal, err := s.PrincipalService.FindPrincipalByID(context.Background(), id)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to find principal with ID: %v", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal.")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal principal.")
		}

		return nil
	})
}
