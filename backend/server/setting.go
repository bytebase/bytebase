package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingAppIM,
	api.SettingWatermark,
	api.SettingWorkspaceProfile,
}

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

		if settingPatch.Name == api.SettingBrandingLogo && !s.licenseService.IsFeatureEnabled(api.FeatureBranding) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureBranding.AccessErrorMessage())
		}

		if s.profile.IsFeatureUnavailable(string(settingPatch.Name)) {
			return echo.NewHTTPError(http.StatusBadRequest, "Feature %v is unavailable in current mode", settingPatch.Name)
		}

		if err := jsonapi.UnmarshalPayload(c.Request().Body, settingPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed update setting request").SetInternal(err)
		}

		if settingPatch.Name == api.SettingWorkspaceProfile {
			payload := new(storepb.WorkspaceProfileSettingPayload)
			if err := json.Unmarshal([]byte(settingPatch.Value), payload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal setting value").SetInternal(err)
			}
			externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid external url").SetInternal(err)
			}
			payload.ExternalUrl = externalURL
			bytes, err := json.Marshal(payload)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal setting value").SetInternal(err)
			}
			settingPatch.Value = string(bytes)
		}

		if settingPatch.Name == api.SettingAppIM {
			var value api.SettingAppIMValue
			if err := json.Unmarshal([]byte(settingPatch.Value), &value); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Malformed setting value for IM").SetInternal(err)
			}
			if value.IMType != api.IMTypeFeishu {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unknown IM Type %s", value.IMType))
			}
			if value.ExternalApproval.Enabled && !s.licenseService.IsFeatureEnabled(api.FeatureIMApproval) {
				return echo.NewHTTPError(http.StatusBadRequest, api.FeatureIMApproval.AccessErrorMessage())
			}
			if value.ExternalApproval.Enabled {
				if value.AppID == "" || value.AppSecret == "" {
					return echo.NewHTTPError(http.StatusBadRequest, "Application ID and secret cannot be empty")
				}
				p := s.feishuProvider
				// clear token cache so that we won't use the previous token.
				p.ClearTokenCache()

				// check bot info
				if _, err := p.GetBotID(ctx, feishu.TokenCtx{
					AppID:     value.AppID,
					AppSecret: value.AppSecret,
				}); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Failed to get bot id. Hint: check if bot is enabled.").SetInternal(err)
				}

				// create approval definition
				approvalDefinitionID, err := p.CreateApprovalDefinition(ctx, feishu.TokenCtx{
					AppID:     value.AppID,
					AppSecret: value.AppSecret,
				}, "")
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Failed to create approval definition").SetInternal(err)
				}

				value.ExternalApproval.ApprovalDefinitionID = approvalDefinitionID
				b, err := json.Marshal(value)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal updated setting value").SetInternal(err)
				}
				settingPatch.Value = string(b)
			}
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
