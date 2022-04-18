package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerActuatorRoutes(g *echo.Group) {
	g.GET("/actuator/info", func(c echo.Context) error {
		ctx := context.Background()
		serverInfo := api.ServerInfo{
			Version:  s.version,
			Readonly: s.readonly,
			Demo:     s.demo,
			Host:     s.host,
			Port:     strconv.Itoa(s.port),
		}

		findRole := api.Owner
		find := &api.MemberFind{
			Role: &findRole,
		}
		memberList, err := s.store.FindMember(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch admin setup status").SetInternal(err)
		}
		serverInfo.NeedAdminSetup = len(memberList) == 0

		return c.JSON(http.StatusOK, serverInfo)
	})
}
