package api

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

var (
	_ Service = (*EnvironmentService)(nil)
)

type Environment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order uint   `json:"order"`
}

type EnvironmentService struct {
}

func (s *EnvironmentService) RegisterRoutes(g *echo.Group) {
	fmt.Println("registerEnvironmentRoutes")
}
