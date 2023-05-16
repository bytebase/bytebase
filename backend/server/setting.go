package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingAppIM,
	api.SettingWatermark,
	api.SettingWorkspaceProfile,
	api.SettingPluginOpenAIKey,
	api.SettingPluginOpenAIEndpoint,
	api.SettingWorkspaceMailDelivery,
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
			if setting.Name == api.SettingWorkspaceMailDelivery {
				// We don't want to return the mail password to the client.
				var value storepb.SMTPMailDeliverySetting
				if err := protojson.Unmarshal([]byte(setting.Value), &value); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal mail delivery setting value").SetInternal(err)
				}
				value.Password = ""
				apiValue := convertStorepbToAPIMailDeliveryValue(&value)
				bytes, err := json.Marshal(apiValue)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal mail delivery setting value").SetInternal(err)
				}
				setting.Value = string(bytes)
			}
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
			payload := new(storepb.WorkspaceProfileSetting)
			if err := protojson.Unmarshal([]byte(settingPatch.Value), payload); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal setting value").SetInternal(err)
			}
			if payload.ExternalUrl != "" {
				externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Invalid external url").SetInternal(err)
				}
				payload.ExternalUrl = externalURL
			}
			bytes, err := protojson.Marshal(payload)
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

		if settingPatch.Name == api.SettingWorkspaceMailDelivery {
			value := api.SettingWorkspaceMailDeliveryValue{}
			if err := json.Unmarshal([]byte(settingPatch.Value), &value); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal mail delivery setting value").SetInternal(err)
			}

			if settingPatch.ValidateOnly {
				var password string
				if v := value.SMTPPassword; v != nil {
					password = *v
				} else {
					var storeValue api.SettingWorkspaceMailDeliveryValue
					settingName := api.SettingWorkspaceMailDelivery
					storeMailDelivery, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
						Name: &settingName,
					})
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get mail delivery setting").SetInternal(err)
					}
					if storeMailDelivery == nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Cannot get mail delivery setting").SetInternal(err)
					}
					if err := json.Unmarshal([]byte(storeMailDelivery.Value), &storeValue); err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal mail delivery setting value").SetInternal(err)
					}
					if storeValue.SMTPPassword != nil {
						password = *storeValue.SMTPPassword
					}
				}
				email := mail.NewEmailMsg()
				email.SetFrom(fmt.Sprintf("Bytebase <%s>", value.SMTPFrom)).AddTo(value.SMTPTo).SetSubject("Test Email Subject").SetBody(`
	<!DOCTYPE html>
	<html>
	<head>
		<title>A test mail from Bytebase.</title>
	</head>
	<body>
		<h1>A test mail from Bytebase.</h1>
	</body>
	</html>
	`)
				client := mail.NewSMTPClient(value.SMTPServerHost, value.SMTPServerPort)
				client.SetAuthType(convertSMTPAuthType(value.SMTPAuthenticationType))
				client.SetAuthCredentials(value.SMTPUsername, password)
				client.SetEncryptionType(convertSMTPEncryptionType(value.SMTPEncryptionType))
				if err := client.SendMail(email); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to send test email").SetInternal(err)
				}
				if _, err := c.Response().Write([]byte("OK")); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to write response").SetInternal(err)
				}
				return nil
			}

			storepbValue := convertAPIMailDeliveryValueToStorePb(&value)
			bytes, err := protojson.Marshal(storepbValue)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal mail delivery setting value").SetInternal(err)
			}
			settingPatch.Value = string(bytes)
		}

		setting, err := s.store.PatchSetting(ctx, settingPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Setting name not found: %s", settingPatch.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update setting: %v", settingPatch.Name)).SetInternal(err)
		}
		if setting.Name == api.SettingWorkspaceMailDelivery {
			// We don't want to return the mail password to the client.
			var value storepb.SMTPMailDeliverySetting
			if err := protojson.Unmarshal([]byte(setting.Value), &value); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal mail delivery setting value").SetInternal(err)
			}
			value.Password = ""
			apiValue := convertStorepbToAPIMailDeliveryValue(&value)
			bytes, err := json.Marshal(apiValue)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal mail delivery setting value").SetInternal(err)
			}
			setting.Value = string(bytes)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, setting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal setting response").SetInternal(err)
		}
		return nil
	})
}

func convertSMTPAuthType(authType storepb.SMTPMailDeliverySetting_Authentication) mail.SMTPAuthType {
	switch authType {
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE:
		return mail.SMTPAuthTypeNone
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_PLAIN:
		return mail.SMTPAuthTypePlain
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_LOGIN:
		return mail.SMTPAuthTypeLogin
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_CRAM_MD5:
		return mail.SMTPAuthTypeCRAMMD5
	}
	return mail.SMTPAuthTypeNone
}

func convertSMTPEncryptionType(encryptionType storepb.SMTPMailDeliverySetting_Encryption) mail.SMTPEncryptionType {
	switch encryptionType {
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE:
		return mail.SMTPEncryptionTypeNone
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_STARTTLS:
		return mail.SMTPEncryptionTypeSTARTTLS
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_SSL_TLS:
		return mail.SMTPEncryptionTypeSSLTLS
	}
	return mail.SMTPEncryptionTypeNone
}

func convertAPIMailDeliveryValueToStorePb(value *api.SettingWorkspaceMailDeliveryValue) *storepb.SMTPMailDeliverySetting {
	if value == nil {
		return nil
	}
	password := ""
	if value.SMTPPassword != nil {
		password = *value.SMTPPassword
	}

	pb := storepb.SMTPMailDeliverySetting{
		Server:         value.SMTPServerHost,
		Port:           int32(value.SMTPServerPort),
		Encryption:     value.SMTPEncryptionType,
		Ca:             "",
		Key:            "",
		Cert:           "",
		Authentication: value.SMTPAuthenticationType,
		Username:       value.SMTPUsername,
		Password:       password,
		From:           value.SMTPFrom,
	}
	return &pb
}

func convertStorepbToAPIMailDeliveryValue(pb *storepb.SMTPMailDeliverySetting) *api.SettingWorkspaceMailDeliveryValue {
	if pb == nil {
		return nil
	}
	password := pb.Password
	value := api.SettingWorkspaceMailDeliveryValue{
		SMTPServerHost:         pb.Server,
		SMTPServerPort:         int(pb.Port),
		SMTPEncryptionType:     pb.Encryption,
		SMTPAuthenticationType: pb.Authentication,
		SMTPUsername:           pb.Username,
		SMTPPassword:           &password,
		SMTPFrom:               pb.From,
	}
	return &value
}
