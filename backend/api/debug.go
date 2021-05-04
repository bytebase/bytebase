package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

var (
	_ Service = (*DebugService)(nil)
)

type DebugService struct {
}

func (s *DebugService) RegisterRoutes(g *echo.Group) {
	g.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, fmt.Sprintf("[%v] Pong from backend", time.Now().Format("2006-01-02 15:04:05")))
	})
}
