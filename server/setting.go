package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

var (
	// Some settings contain secret info so we only return settings that are needed by the client.
	whitelistSettings = []api.SettingName{
		api.SettingBrandingLogo,
	}
)

func (s *Server) registerSettingRoutes(g *echo.Group) {
	g.GET("/setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		find := &api.SettingFind{}
		settingList, err := s.store.FindSetting(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch setting list").SetInternal(err)
		}

		filteredList := []*api.Setting{}
		for _, setting := range settingList {
			for _, whitelist := range whitelistSettings {
				if setting.Name == whitelist {
					filteredList = append(filteredList, setting)
					break
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal project list response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/setting/:name", func(c echo.Context) error {
		ctx := c.Request().Context()
		settingPatch := &api.SettingPatch{
			Name:      api.SettingName(c.Param("name")),
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}

		if settingPatch.Name == api.SettingBrandingLogo && !s.feature(api.FeatureBranding) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureBranding.AccessErrorMessage())
		}

		if err := jsonapi.UnmarshalPayload(c.Request().Body, settingPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update setting request").SetInternal(err)
		}

		if settingPatch.Name == api.SettingAppFeishu {
			var value api.SettingAppFeishuValue
			if err := json.Unmarshal([]byte(settingPatch.Value), &value); err != nil {
				return err
			}
			// check if we have one already.
			name := api.SettingAppFeishu
			setting, err := s.store.GetSetting(ctx, &api.SettingFind{Name: &name})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get setting").SetInternal(err)
			}
			if setting != nil {
				var oldValue api.SettingAppFeishuValue
				if err := json.Unmarshal([]byte(setting.Value), &oldValue); err != nil {
					return err
				}
				if oldValue.AppID == value.AppID && oldValue.AppSecret == value.AppSecret {
					return echo.NewHTTPError(http.StatusBadRequest, "Setting value has not changed")
				}
			} else {
				if _, err := s.store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					Name:        api.SettingAppFeishu,
					Value:       "",
					Description: "",
				}); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create setting").SetInternal(err)
				}
			}
			// TODO(p0ny): create approval definition
		}

		setting, err := s.store.PatchSetting(ctx, settingPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
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
