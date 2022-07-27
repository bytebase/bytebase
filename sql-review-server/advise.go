package sqlreviewserver

import (
	"github.com/labstack/echo/v4"
)

func (s *Server) registerAdvisorRoutes(g *echo.Group) {
	g.GET("/sql/advise", s.sqlCheckController)
}

func (s *Server) sqlCheckController(c echo.Context) error {
	return nil
}
