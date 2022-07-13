package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/labstack/echo/v4"
)

// demoDataPath is the path prefix for demo data.
var demoDataPath = "demo/"

func (s *Server) registerActuatorRoutes(g *echo.Group) {
	g.GET("/actuator/info", func(c echo.Context) error {
		ctx := c.Request().Context()

		serverInfo := api.ServerInfo{
			Version:   s.profile.Version,
			GitCommit: s.profile.GitCommit,
			Readonly:  s.profile.Readonly,
			Demo:      s.profile.Demo,
			Host:      s.profile.BackendHost,
			Port:      strconv.Itoa(s.profile.BackendPort),
		}

		if s.profile.Demo && strings.HasPrefix(s.profile.DemoDataDir, demoDataPath) {
			serverInfo.DemoName = s.profile.DemoDataDir[len(demoDataPath):len(s.profile.DemoDataDir)]
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
