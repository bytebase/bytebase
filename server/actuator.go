package server

import (
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerActuatorRoutes(g *echo.Group) {
	g.GET("/actuator/info", func(c echo.Context) error {
		ctx := c.Request().Context()
		demoName := s.profile.DemoDataDir[len("demo/"):len(s.profile.DemoDataDir)]
		serverInfo := api.ServerInfo{
			Version:   s.profile.Version,
			GitCommit: s.profile.GitCommit,
			Readonly:  s.profile.Readonly,
			Demo:      s.profile.Demo,
			DemoName:  demoName,
			Host:      s.profile.BackendHost,
			Port:      strconv.Itoa(s.profile.BackendPort),
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
