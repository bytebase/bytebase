package api

import (
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

var (
	_ Service = (*AuthService)(nil)
)

type Login struct {
	Email    string `jsonapi:"attr,email"`
	Password string `jsonapi:"attr,password"`
}

type AuthService struct {
}

func (s *AuthService) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/login", func(c echo.Context) error {
		login := &Login{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, login); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted login request")
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if err := jsonapi.MarshalPayload(c.Response().Writer, login); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal login response")
		}

		return nil
	})
}
