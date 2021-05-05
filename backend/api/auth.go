package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

var (
	_ Service = (*AuthService)(nil)
)

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthService struct {
}

func (s *AuthService) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/login", func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	})
}
