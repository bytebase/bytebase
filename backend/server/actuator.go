package server

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerActuatorRoutes(g *echo.Group) {
	g.GET("/actuator/info", func(c echo.Context) error {
		ctx := c.Request().Context()

		settingName := api.SettingWorkspaceDisallowSignup
		setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
			Name: &settingName,
		})
		if err != nil || setting == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find disallow signup setting").SetInternal(err)
		}
		disallowSignup, err := strconv.ParseBool(setting.Value)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to convert disallow signup setting to bool").SetInternal(err)
		}

		serverInfo := api.ServerInfo{
			Version:        s.profile.Version,
			GitCommit:      s.profile.GitCommit,
			Readonly:       s.profile.Readonly,
			DemoName:       s.profile.DemoName,
			ExternalURL:    s.profile.ExternalURL,
			DisallowSignup: disallowSignup,
		}

		findRole := api.Owner
		users, err := s.store.ListUsers(ctx, &store.FindUserMessage{Role: &findRole})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch admin setup status").SetInternal(err)
		}
		count := 0
		for _, user := range users {
			if user.ID == api.SystemBotID {
				continue
			}
			count++
		}
		serverInfo.NeedAdminSetup = count == 0

		return c.JSON(http.StatusOK, serverInfo)
	})
}
