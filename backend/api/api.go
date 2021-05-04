package api

import "github.com/labstack/echo/v4"

type Service interface {
	RegisterRoutes(g *echo.Group)
}
