package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

var (
	// Some settings contain secret info so we only return settings that are needed by the client.
	whitelistSettings = []api.SettingName{api.SettingConsoleDatabase, api.SettingConsoleTable}
)

func (s *Server) registerSettingRoutes(g *echo.Group) {
	g.GET("/setting", func(c echo.Context) error {
		find := &api.SettingFind{}
		list, err := s.SettingService.FindSettingList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch setting list").SetInternal(err)
		}

		filteredList := []*api.Setting{}
		for _, setting := range list {
			for _, whitelist := range whitelistSettings {
				if setting.Name == whitelist {
					filteredList = append(filteredList, setting)
					break
				}
			}
		}

		for _, setting := range filteredList {
			if err := s.ComposeSettingRelationship(context.Background(), setting); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch setting relationship: %v", setting.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/setting/:name", func(c echo.Context) error {
		settingPatch := &api.SettingPatch{
			Name:      api.SettingName(c.Param("name")),
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, settingPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted update setting request").SetInternal(err)
		}

		setting, err := s.SettingService.PatchSetting(context.Background(), settingPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Setting name not found: %s", settingPatch.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update setting: %v", settingPatch.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, setting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal setting response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeSettingRelationship(ctx context.Context, setting *api.Setting) error {
	var err error

	setting.Creator, err = s.ComposePrincipalById(ctx, setting.CreatorId)
	if err != nil {
		return err
	}

	setting.Updater, err = s.ComposePrincipalById(context.Background(), setting.UpdaterId)
	if err != nil {
		return err
	}

	return nil
}
