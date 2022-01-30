//go:build release
// +build release

package server

import "github.com/labstack/echo/v4"

// We will not expose plan routes in release as they're only used for test in dev environment.
func (s *Server) registerPlanRoutes(g *echo.Group) {}
