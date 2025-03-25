package v1

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	"github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	"github.com/bytebase/bytebase/backend/plugin/webhook/lark"
	"github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	"github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SettingService implements the setting service.
type SettingService struct {
	v1pb.UnimplementedSettingServiceServer
	store          *store.Store
	profile        *config.Profile
	licenseService enterprise.LicenseService
	stateCfg       *state.State
}

// NewSettingService creates a new setting service.
func NewSettingService(
	store *store.Store,
	profile *config.Profile,
	licenseService enterprise.LicenseService,
	stateCfg *state.State,
) *SettingService {
	return &SettingService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
		stateCfg:       stateCfg,
	}
}

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingWorkspaceID,
	api.SettingAppIM,
	api.SettingWatermark,
	api.SettingAI,
	api.SettingWorkspaceApproval,
	api.SettingWorkspaceMailDelivery,
	api.SettingWorkspaceProfile,
	api.SettingWorkspaceExternalApproval,
	api.SettingSchemaTemplate,
	api.SettingDataClassification,
	api.SettingSemanticTypes,
	api.SettingSQLResultSizeLimit,
	api.SettingSCIM,
	api.SettingPasswordRestriction,
}

//go:embed mail_templates/testmail/template.html
//go:embed mail_templates/testmail/statics/logo-full.png
//go:embed mail_templates/testmail/statics/banner.png
var testEmailFs embed.FS

// ListSettings lists all settings.
func (s *SettingService) ListSettings(ctx context.Context, _ *v1pb.ListSettingsRequest) (*v1pb.ListSettingsResponse, error) {
	settings, err := s.store.ListSettingV2(ctx, &store.FindSettingMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list settings: %v", err)
	}

	response := &v1pb.ListSettingsResponse{}
	for _, setting := range settings {
		if !settingInWhitelist(setting.Name) {
			continue
		}
		settingMessage, err := s.convertToSettingMessage(ctx, setting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
		}
		response.Settings = append(response.Settings, settingMessage)
	}
	return response, nil
}

// GetSetting gets the setting by name.
func (s *SettingService) GetSetting(ctx context.Context, request *v1pb.GetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := common.GetSettingName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is invalid: %v", err)
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	apiSettingName := api.SettingName(settingName)
	if !settingInWhitelist(apiSettingName) {
		return nil, status.Errorf(codes.InvalidArgument, "setting is not available")
	}

	setting, err := s.store.GetSettingV2(ctx, apiSettingName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get setting: %v", err)
	}
	if setting == nil {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
	}
	// Only return whitelisted setting.
	settingMessage, err := s.convertToSettingMessage(ctx, setting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
	}
	return settingMessage, nil
}

// SetSetting set the setting by name.
func (s *SettingService) UpdateSetting(ctx context.Context, request *v1pb.UpdateSettingRequest) (*v1pb.Setting, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	settingName, err := common.GetSettingName(request.Setting.Name)
	if err != nil {
		return nil, err
	}
	if settingName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "setting name is empty")
	}
	if s.profile.IsFeatureUnavailable(settingName) {
		return nil, status.Errorf(codes.InvalidArgument, "feature %s is unavailable in current mode", settingName)
	}
	apiSettingName := api.SettingName(settingName)
	existedSetting, err := s.store.GetSettingV2(ctx, apiSettingName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find setting %s with error: %v", settingName, err)
	}
	if existedSetting == nil && !request.AllowMissing {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
	}
	// audit log.
	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok && existedSetting != nil {
		v1pbSetting, err := s.convertToSettingMessage(ctx, existedSetting)
		if err != nil {
			slog.Warn("audit: failed to convert to v1.Setting", log.BBError(err))
		}
		p, err := anypb.New(v1pbSetting)
		if err != nil {
			slog.Warn("audit: failed to convert to anypb.Any", log.BBError(err))
		}
		setServiceData(p)
	}

	var storeSettingValue string
	switch apiSettingName {
	case api.SettingWorkspaceProfile:
		if request.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
		}
		payload := new(storepb.WorkspaceProfileSetting)
		if err := convertProtoToProto(request.Setting.Value.GetWorkspaceProfileSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		oldSetting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find setting %s with error: %v", apiSettingName, err)
		}

		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "value.workspace_profile_setting_value.disallow_signup":
				if s.profile.SaaS {
					return nil, status.Errorf(codes.InvalidArgument, "feature %s is unavailable in current mode", settingName)
				}
				if err := s.licenseService.IsFeatureEnabled(api.FeatureDisallowSignup); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				oldSetting.DisallowSignup = payload.DisallowSignup
			case "value.workspace_profile_setting_value.external_url":
				if s.profile.SaaS {
					return nil, status.Errorf(codes.InvalidArgument, "feature %s is unavailable in current mode", settingName)
				}
				if payload.ExternalUrl != "" {
					externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
					if err != nil {
						return nil, status.Errorf(codes.InvalidArgument, "invalid external url: %v", err)
					}
					payload.ExternalUrl = externalURL
				}
				oldSetting.ExternalUrl = payload.ExternalUrl
			case "value.workspace_profile_setting_value.require_2fa":
				if err := s.licenseService.IsFeatureEnabled(api.Feature2FA); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				oldSetting.Require_2Fa = payload.Require_2Fa
			case "value.workspace_profile_setting_value.outbound_ip_list":
				// We're not support update outbound_ip_list via api.
			case "value.workspace_profile_setting_value.token_duration":
				if err := s.licenseService.IsFeatureEnabled(api.FeatureSecureToken); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				if payload.TokenDuration != nil && payload.TokenDuration.Seconds > 0 && payload.TokenDuration.AsDuration() < time.Hour {
					return nil, status.Errorf(codes.InvalidArgument, "refresh token duration should be at least one hour")
				}
				oldSetting.TokenDuration = payload.TokenDuration
			case "value.workspace_profile_setting_value.announcement":
				if err := s.licenseService.IsFeatureEnabled(api.FeatureAnnouncement); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				oldSetting.Announcement = payload.Announcement
			case "value.workspace_profile_setting_value.maximum_role_expiration":
				if payload.MaximumRoleExpiration != nil {
					// If the value is less than or equal to 0, we will remove the setting. AKA no limit.
					if payload.MaximumRoleExpiration.Seconds <= 0 {
						payload.MaximumRoleExpiration = nil
					}
				}
				oldSetting.MaximumRoleExpiration = payload.MaximumRoleExpiration
			case "value.workspace_profile_setting_value.domains":
				if err := validateDomains(payload.Domains); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid domains, error %v", err)
				}
				oldSetting.Domains = payload.Domains
			case "value.workspace_profile_setting_value.enforce_identity_domain":
				if payload.EnforceIdentityDomain {
					if err := s.licenseService.IsFeatureEnabled(api.FeatureDomainRestriction); err != nil {
						return nil, status.Error(codes.PermissionDenied, err.Error())
					}
				}
				oldSetting.EnforceIdentityDomain = payload.EnforceIdentityDomain
			case "value.workspace_profile_setting_value.database_change_mode":
				oldSetting.DatabaseChangeMode = payload.DatabaseChangeMode
			case "value.workspace_profile_setting_value.disallow_password_signin":
				// TODO(steven): add feature flag checks.
				if payload.DisallowPasswordSignin {
					identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to list identity providers: %v", err)
					}
					if len(identityProviders) == 0 {
						return nil, status.Errorf(codes.InvalidArgument, "cannot disallow password signin when no identity provider is set")
					}
				}
				oldSetting.DisallowPasswordSignin = payload.DisallowPasswordSignin
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid update mask path %v", path)
			}
		}

		if len(oldSetting.Domains) == 0 && oldSetting.EnforceIdentityDomain {
			return nil, status.Errorf(codes.InvalidArgument, "identity domain can be enforced only when workspace domains are set")
		}
		bytes, err := protojson.Marshal(oldSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingWorkspaceApproval:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureCustomApproval); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		payload := &storepb.WorkspaceApprovalSetting{}
		for _, rule := range request.Setting.Value.GetWorkspaceApprovalSettingValue().Rules {
			// Validate the condition.
			if _, err := common.ConvertUnparsedApproval(rule.Condition); err != nil {
				return nil, err
			}
			if err := validateApprovalTemplate(rule.Template); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid approval template: %v, err: %v", rule.Template, err)
			}

			creatorID := 0
			email, err := common.GetUserEmail(rule.Template.Creator)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to get creator: %v", err)
			}
			creator, err := s.store.GetUserByEmail(ctx, email)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get creator: %v", err)
			}
			if creator == nil {
				return nil, status.Errorf(codes.InvalidArgument, "creator %s not found", rule.Template.Creator)
			}
			creatorID = creator.ID

			flow := new(storepb.ApprovalFlow)
			if err := convertProtoToProto(rule.Template.Flow, flow); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal approval flow with error: %v", err)
			}
			payload.Rules = append(payload.Rules, &storepb.WorkspaceApprovalSetting_Rule{
				Condition: rule.Condition,
				Template: &storepb.ApprovalTemplate{
					Flow:        flow,
					Title:       rule.Template.Title,
					Description: rule.Template.Description,
					CreatorId:   int32(creatorID),
				},
			})
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingWorkspaceMailDelivery:
		apiValue := request.Setting.Value.GetSmtpMailDeliverySettingValue()
		// We will fill the password read from the store if it is not set.
		if apiValue.Password == nil {
			oldStoreSetting, err := s.store.GetSettingV2(ctx, apiSettingName)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get setting %q: %v", apiSettingName, err)
			}
			if oldStoreSetting == nil {
				return nil, status.Errorf(codes.InvalidArgument, "should set the password for the first time")
			}
			oldValue := new(storepb.SMTPMailDeliverySetting)
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(oldStoreSetting.Value), oldValue); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", err, apiSettingName)
			}
			apiValue.Password = &oldValue.Password
		}
		if request.ValidateOnly {
			if err := s.sendTestEmail(ctx, apiValue); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to validate smtp setting: %v", err)
			}
			apiValue.Password = nil
			return &v1pb.Setting{
				Name: request.Setting.Name,
				Value: &v1pb.Value{
					Value: &v1pb.Value_SmtpMailDeliverySettingValue{
						SmtpMailDeliverySettingValue: apiValue,
					},
				},
			}, nil
		}
		password := ""
		if apiValue.Password != nil {
			password = *apiValue.Password
		}
		storeMailDeliveryValue := &storepb.SMTPMailDeliverySetting{
			Server:         apiValue.Server,
			Port:           apiValue.Port,
			Encryption:     convertToStorePbSMTPEncryptionType(apiValue.Encryption),
			Authentication: convertToStorePbSMTPAuthType(apiValue.Authentication),
			Username:       apiValue.Username,
			Password:       password,
			Ca:             "",
			Key:            "",
			Cert:           "",
			From:           apiValue.From,
		}
		bytes, err := protojson.Marshal(storeMailDeliveryValue)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting value for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingBrandingLogo:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureBranding); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case api.SettingPluginAgent:
		payload := new(storepb.AgentPluginSetting)
		if err := convertProtoToProto(request.Setting.Value.GetAgentPluginSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}

		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)

	case api.SettingAppIM:
		payload := new(storepb.AppIMSetting)
		if err := convertProtoToProto(request.Setting.Value.GetAppImSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s, error: %v", apiSettingName, err)
		}
		setting, err := s.store.GetAppIMSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get old app im setting")
		}
		if request.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
		}
		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "value.app_im_setting_value.slack":
				if err := slack.ValidateToken(ctx, payload.Slack.GetToken()); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed, error: %v", err)
				}
				setting.Slack = payload.Slack

			case "value.app_im_setting_value.feishu":
				if err := feishu.Validate(ctx, payload.GetFeishu().GetAppId(), payload.GetFeishu().GetAppSecret(), user.Email); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed, error: %v", err)
				}
				setting.Feishu = payload.Feishu

			case "value.app_im_setting_value.wecom":
				if err := wecom.Validate(ctx, payload.GetWecom().GetCorpId(), payload.GetWecom().GetAgentId(), payload.GetWecom().GetSecret()); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed, error: %v", err)
				}
				setting.Wecom = payload.Wecom

			case "value.app_im_setting_value.lark":
				if err := lark.Validate(ctx, payload.GetLark().GetAppId(), payload.GetLark().GetAppSecret(), user.Email); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed, error: %v", err)
				}
				setting.Lark = payload.Lark
			case "value.app_im_setting_value.dingtalk":
				if err := dingtalk.Validate(ctx, payload.GetDingtalk().GetClientId(), payload.GetDingtalk().GetClientSecret(), payload.GetDingtalk().RobotCode, user.Phone); err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "validation failed, error: %v", err)
				}
				setting.Dingtalk = payload.Dingtalk

			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid update mask path %v", path)
			}
		}
		if request.ValidateOnly {
			return &v1pb.Setting{
				Name: request.Setting.Name,
				Value: &v1pb.Value{
					Value: &v1pb.Value_AppImSettingValue{
						AppImSettingValue: &v1pb.AppIMSetting{},
					},
				},
			}, nil
		}

		bytes, err := protojson.Marshal(setting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s, error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)

	case api.SettingSchemaTemplate:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSchemaTemplate); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		schemaTemplateSetting := request.Setting.Value.GetSchemaTemplateSettingValue()
		if schemaTemplateSetting == nil {
			return nil, status.Errorf(codes.InvalidArgument, "value cannot be nil when setting schema template setting")
		}

		if err := s.validateSchemaTemplate(ctx, schemaTemplateSetting); err != nil {
			return nil, err
		}

		payload, err := convertV1SchemaTemplateSetting(schemaTemplateSetting)
		if err != nil {
			return nil, err
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal external approval setting, error: %v", err)
		}
		storeSettingValue = string(bytes)
	case api.SettingDataClassification:
		payload := new(storepb.DataClassificationSetting)
		if err := convertProtoToProto(request.Setting.Value.GetDataClassificationSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		// it's a temporary solution to limit only 1 classification config before we support manage it in the UX.
		if len(payload.Configs) != 1 {
			return nil, status.Errorf(codes.InvalidArgument, "only support define 1 classification config for now")
		}
		if len(payload.Configs[0].Classification) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "missing classification map")
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingSemanticTypes:
		storeSemanticTypeSetting := new(storepb.SemanticTypeSetting)
		if err := convertProtoToProto(request.Setting.Value.GetSemanticTypeSettingValue(), storeSemanticTypeSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		idMap := make(map[string]struct{})
		for _, tp := range storeSemanticTypeSetting.Types {
			if tp.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "category title cannot be empty: %s", tp.Id)
			}
			if _, ok := idMap[tp.Id]; ok {
				return nil, status.Errorf(codes.InvalidArgument, "duplicate semantic type id: %s", tp.Id)
			}
			idMap[tp.Id] = struct{}{}
		}
		bytes, err := protojson.Marshal(storeSemanticTypeSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingWatermark:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureWatermark); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case api.SettingSQLResultSizeLimit:
		maximumSQLResultSizeSetting := new(storepb.MaximumSQLResultSizeSetting)
		if err := convertProtoToProto(request.Setting.Value.GetMaximumSqlResultSizeSetting(), maximumSQLResultSizeSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		if maximumSQLResultSizeSetting.Limit <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid maximum sql result size")
		}
		bytes, err := protojson.Marshal(maximumSQLResultSizeSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingSCIM:
		scimToken, err := common.RandomString(32)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate random SCIM secret with error: %v", err)
		}
		bytes, err := protojson.Marshal(&storepb.SCIMSetting{
			Token: scimToken,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal SCIM setting with error: %v", err)
		}
		storeSettingValue = string(bytes)
	case api.SettingPasswordRestriction:
		if err := s.licenseService.IsFeatureEnabled(api.FeaturePasswordRestriction); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		passwordSetting := new(storepb.PasswordRestrictionSetting)
		if err := convertProtoToProto(request.Setting.Value.GetPasswordRestrictionSetting(), passwordSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		if passwordSetting.MinLength < 8 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid password minimum length, should no less than 8")
		}
		bytes, err := protojson.Marshal(passwordSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingAI:
		aiSetting := &storepb.AISetting{}
		if err := convertProtoToProto(request.Setting.Value.GetAiSetting(), aiSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		if aiSetting.Enabled {
			if aiSetting.Endpoint == "" || aiSetting.Model == "" {
				return nil, status.Errorf(codes.InvalidArgument, "API endpoint and model are required")
			}
			if existedSetting != nil {
				existedAISetting, err := s.convertToSettingMessage(ctx, existedSetting)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to unmarshal existed ai setting with error: %v", err)
				}
				if aiSetting.ApiKey == "" {
					aiSetting.ApiKey = existedAISetting.Value.GetAiSetting().GetApiKey()
				}
			}
			if aiSetting.ApiKey == "" {
				return nil, status.Errorf(codes.InvalidArgument, "API key is required")
			}
		}

		bytes, err := protojson.Marshal(aiSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	default:
		storeSettingValue = request.Setting.Value.GetStringValue()
	}
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set setting: %v", err)
	}

	settingMessage, err := s.convertToSettingMessage(ctx, setting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
	}

	// it's a temporary solution to map the classification to all projects before we support it in the UX.
	if apiSettingName == api.SettingDataClassification && len(settingMessage.Value.GetDataClassificationSettingValue().Configs) == 1 {
		classificationID := settingMessage.Value.GetDataClassificationSettingValue().Configs[0].Id
		projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{ShowDeleted: false})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list projects with error: %v", err.Error())
		}
		for _, project := range projects {
			patch := &store.UpdateProjectMessage{
				ResourceID:                 project.ResourceID,
				DataClassificationConfigID: &classificationID,
			}
			if _, err = s.store.UpdateProjectV2(ctx, patch); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to patch project %s with error: %v", project.Title, err.Error())
			}
		}
	}

	return settingMessage, nil
}

func convertProtoToProto(inputPB, outputPB protoreflect.ProtoMessage) error {
	bytes, err := protojson.Marshal(inputPB)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal setting: %v", err)
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(bytes, outputPB); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal setting: %v", err)
	}
	return nil
}

func (s *SettingService) convertToSettingMessage(ctx context.Context, setting *store.SettingMessage) (*v1pb.Setting, error) {
	settingName := fmt.Sprintf("%s%s", common.SettingNamePrefix, setting.Name)
	switch setting.Name {
	case api.SettingWorkspaceMailDelivery:
		storeValue := new(storepb.SMTPMailDeliverySetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return stripSensitiveData(&v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SmtpMailDeliverySettingValue{
					SmtpMailDeliverySettingValue: &v1pb.SMTPMailDeliverySettingValue{
						Server:         storeValue.Server,
						Port:           storeValue.Port,
						Encryption:     convertToSMTPEncryptionType(storeValue.Encryption),
						Authentication: convertToSMTPAuthType(storeValue.Authentication),
						Ca:             &storeValue.Ca,
						Key:            &storeValue.Key,
						Cert:           &storeValue.Cert,
						Username:       storeValue.Username,
						Password:       &storeValue.Password,
						From:           storeValue.From,
					},
				},
			},
		})
	case api.SettingAppIM:
		storeValue := new(storepb.AppIMSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AppImSettingValue{
					AppImSettingValue: &v1pb.AppIMSetting{
						Slack: &v1pb.AppIMSetting_Slack{
							Enabled: storeValue.Slack != nil && storeValue.Slack.Enabled,
						},
						Feishu: &v1pb.AppIMSetting_Feishu{
							Enabled: storeValue.Feishu != nil && storeValue.Feishu.Enabled,
						},
						Wecom: &v1pb.AppIMSetting_Wecom{
							Enabled: storeValue.Wecom != nil && storeValue.Wecom.Enabled,
						},
						Lark: &v1pb.AppIMSetting_Lark{
							Enabled: storeValue.Lark != nil && storeValue.Lark.Enabled,
						},
						Dingtalk: &v1pb.AppIMSetting_DingTalk{
							Enabled: storeValue.Dingtalk != nil && storeValue.Dingtalk.Enabled,
						},
					},
				},
			},
		}, nil
	case api.SettingPluginAgent:
		v1Value := new(v1pb.AgentPluginSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AgentPluginSettingValue{
					AgentPluginSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingWorkspaceProfile:
		v1Value := new(v1pb.WorkspaceProfileSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceProfileSettingValue{
					WorkspaceProfileSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingWorkspaceApproval:
		storeValue := new(storepb.WorkspaceApprovalSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		v1Value := &v1pb.WorkspaceApprovalSetting{}
		for _, rule := range storeValue.Rules {
			template := convertToApprovalTemplate(rule.Template)
			creator, err := s.store.GetUserByID(ctx, int(rule.Template.CreatorId))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get creator: %v", err)
			}
			if creator != nil {
				template.Creator = common.FormatUserEmail(creator.Email)
			}
			v1Value.Rules = append(v1Value.Rules, &v1pb.WorkspaceApprovalSetting_Rule{
				Condition: rule.Condition,
				Template:  template,
			})
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceApprovalSettingValue{
					WorkspaceApprovalSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingSchemaTemplate:
		value := new(storepb.SchemaTemplateSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}

		sts, err := convertSchemaTemplateSetting(value)
		if err != nil {
			return nil, err
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SchemaTemplateSettingValue{
					SchemaTemplateSettingValue: sts,
				},
			},
		}, nil
	case api.SettingDataClassification:
		v1Value := new(v1pb.DataClassificationSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_DataClassificationSettingValue{
					DataClassificationSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingSemanticTypes:
		v1Value := new(v1pb.SemanticTypeSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SemanticTypeSettingValue{
					SemanticTypeSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingSQLResultSizeLimit:
		v1Value := new(v1pb.MaximumSQLResultSizeSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		if v1Value.GetLimit() <= 0 {
			v1Value.Limit = common.DefaultMaximumSQLResultSize
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_MaximumSqlResultSizeSetting{
					MaximumSqlResultSizeSetting: v1Value,
				},
			},
		}, nil
	case api.SettingSCIM:
		v1Value := new(v1pb.SCIMSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_ScimSetting{
					ScimSetting: v1Value,
				},
			},
		}, nil
	case api.SettingPasswordRestriction:
		v1Value := new(v1pb.PasswordRestrictionSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_PasswordRestrictionSetting{
					PasswordRestrictionSetting: v1Value,
				},
			},
		}, nil
	case api.SettingAI:
		v1Value := &v1pb.AISetting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		// DO NOT expose the api key.
		v1Value.ApiKey = ""
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AiSetting{
					AiSetting: v1Value,
				},
			},
		}, nil
	default:
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_StringValue{
					StringValue: setting.Value,
				},
			},
		}, nil
	}
}

func (s *SettingService) validateSchemaTemplate(ctx context.Context, schemaTemplateSetting *v1pb.SchemaTemplateSetting) error {
	oldStoreSetting, err := s.store.GetSettingV2(ctx, api.SettingSchemaTemplate)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get setting %q: %v", api.SettingSchemaTemplate, err)
	}
	settingValue := "{}"
	if oldStoreSetting != nil {
		settingValue = oldStoreSetting.Value
	}

	value := new(storepb.SchemaTemplateSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(settingValue), value); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal setting value for %v with error: %v", api.SettingSchemaTemplate, err)
	}
	v1Value, err := convertSchemaTemplateSetting(value)
	if err != nil {
		return err
	}

	// validate the changed field(column) template.
	oldFieldTemplateMap := map[string]*v1pb.SchemaTemplateSetting_FieldTemplate{}
	for _, template := range v1Value.FieldTemplates {
		oldFieldTemplateMap[template.Id] = template
	}
	for _, template := range schemaTemplateSetting.FieldTemplates {
		oldTemplate, ok := oldFieldTemplateMap[template.Id]
		if ok && cmp.Equal(oldTemplate, template, protocmp.Transform()) {
			continue
		}
		tableMetadata := &v1pb.TableMetadata{
			Name:    "temp_table",
			Columns: []*v1pb.ColumnMetadata{template.Column},
		}
		if err := validateTableMetadata(template.Engine, tableMetadata); err != nil {
			return err
		}
	}

	// validate the changed table template.
	oldTableTemplateMap := map[string]*v1pb.SchemaTemplateSetting_TableTemplate{}
	for _, template := range v1Value.TableTemplates {
		oldTableTemplateMap[template.Id] = template
	}
	for _, template := range schemaTemplateSetting.TableTemplates {
		oldTemplate, ok := oldTableTemplateMap[template.Id]
		if ok && cmp.Equal(oldTemplate, template, protocmp.Transform()) {
			continue
		}
		if err := validateTableMetadata(template.Engine, template.Table); err != nil {
			return err
		}
	}

	return nil
}

func validateTableMetadata(engine v1pb.Engine, tableMetadata *v1pb.TableMetadata) error {
	tempSchema := &v1pb.SchemaMetadata{
		Name:   "",
		Tables: []*v1pb.TableMetadata{tableMetadata},
	}
	if engine == v1pb.Engine_POSTGRES {
		tempSchema.Name = "temp_schema"
	}
	tempMetadata := &v1pb.DatabaseMetadata{
		Name:    "temp_database",
		Schemas: []*v1pb.SchemaMetadata{tempSchema},
	}
	tempStoreSchemaMetadata, err := convertV1DatabaseMetadata(tempMetadata)
	if err != nil {
		return err
	}
	if err := checkDatabaseMetadata(storepb.Engine(engine), tempStoreSchemaMetadata); err != nil {
		return errors.Wrap(err, "failed to check database metadata")
	}
	if _, err := schema.GetDatabaseDefinition(storepb.Engine(engine), schema.GetDefinitionContext{}, tempStoreSchemaMetadata); err != nil {
		return errors.Wrap(err, "failed to transform database metadata to schema string")
	}
	return nil
}

func settingInWhitelist(name api.SettingName) bool {
	for _, whitelist := range whitelistSettings {
		if name == whitelist {
			return true
		}
	}
	return false
}

func validateApprovalTemplate(template *v1pb.ApprovalTemplate) error {
	if template.Flow == nil {
		return errors.Errorf("approval template cannot be nil")
	}
	if len(template.Flow.Steps) == 0 {
		return errors.Errorf("approval template cannot have 0 step")
	}
	for _, step := range template.Flow.Steps {
		if step.Type != v1pb.ApprovalStep_ANY {
			return errors.Errorf("invalid approval step type: %v", step.Type)
		}
		if len(step.Nodes) != 1 {
			return errors.Errorf("expect 1 node in approval step, got: %v", len(step.Nodes))
		}
	}
	return nil
}

func (s *SettingService) sendTestEmail(ctx context.Context, value *v1pb.SMTPMailDeliverySettingValue) error {
	if value.Password == nil {
		return status.Errorf(codes.InvalidArgument, "password is required when sending test email")
	}
	if value.To == "" {
		return status.Errorf(codes.InvalidArgument, "to is required when sending test email")
	}
	if value.From == "" {
		return status.Errorf(codes.InvalidArgument, "from is required when sending test email")
	}

	consoleRedirectURL := "www.bytebase.com"
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get workspace profile setting: %v", err)
	}
	if setting.ExternalUrl != "" {
		consoleRedirectURL = setting.ExternalUrl
	}

	email := mail.NewEmailMsg()

	logoFull, err := testEmailFs.ReadFile("mail_templates/testmail/statics/logo-full.png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read logo-full.png: %v", err)
	}
	banner, err := testEmailFs.ReadFile("mail_templates/testmail/statics/banner.png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read banner.png: %v", err)
	}
	mailHTMLBody, err := testEmailFs.ReadFile("mail_templates/testmail/template.html")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read template.html: %v", err)
	}
	logoFullReader := bytes.NewReader(logoFull)
	bannerReader := bytes.NewReader(banner)
	logoFullFileName, err := email.Attach(logoFullReader, "logo-full.png", "image/png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach logo-full.png: %v", err)
	}
	bannerFileName, err := email.Attach(bannerReader, "banner.png", "image/png")
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach banner.png: %v", err)
	}

	polishHTMLBody := prepareTestMailContent(string(mailHTMLBody), consoleRedirectURL, logoFullFileName, bannerFileName)
	email.SetFrom(fmt.Sprintf("Bytebase <%s>", value.From)).AddTo(value.To).SetSubject("Bytebase mail server test").SetBody(polishHTMLBody)
	client := mail.NewSMTPClient(value.Server, int(value.Port))
	client.SetAuthType(convertToMailSMTPAuthType(value.Authentication)).
		SetAuthCredentials(value.Username, *value.Password).
		SetEncryptionType(convertToMailSMTPEncryptionType(value.Encryption))

	if err := client.SendMail(email); err != nil {
		return status.Errorf(codes.Internal, "failed to send test email: %v", err)
	}
	return nil
}

func prepareTestMailContent(htmlTemplate, consoleRedirectURL, logoContentID, bannerContentID string) string {
	testEmailContent := strings.ReplaceAll(htmlTemplate, "{{BYTEBASE_LOGO_URL}}", fmt.Sprintf("cid:%s", logoContentID))
	testEmailContent = strings.ReplaceAll(testEmailContent, "{{BYTEBASE_BANNER_URL}}", fmt.Sprintf("cid:%s", bannerContentID))
	testEmailContent = strings.ReplaceAll(testEmailContent, "{{BYTEBASE_CONSOLE_REDIRECT_URL}}", consoleRedirectURL)
	return testEmailContent
}

func convertToMailSMTPAuthType(authType v1pb.SMTPMailDeliverySettingValue_Authentication) mail.SMTPAuthType {
	switch authType {
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_NONE:
		return mail.SMTPAuthTypeNone
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_PLAIN:
		return mail.SMTPAuthTypePlain
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_LOGIN:
		return mail.SMTPAuthTypeLogin
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_CRAM_MD5:
		return mail.SMTPAuthTypeCRAMMD5
	}
	return mail.SMTPAuthTypeNone
}

func convertToStorePbSMTPAuthType(authType v1pb.SMTPMailDeliverySettingValue_Authentication) storepb.SMTPMailDeliverySetting_Authentication {
	switch authType {
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_NONE:
		return storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_PLAIN:
		return storepb.SMTPMailDeliverySetting_AUTHENTICATION_PLAIN
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_LOGIN:
		return storepb.SMTPMailDeliverySetting_AUTHENTICATION_LOGIN
	case v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_CRAM_MD5:
		return storepb.SMTPMailDeliverySetting_AUTHENTICATION_CRAM_MD5
	}
	return storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE
}

func convertToSMTPAuthType(authType storepb.SMTPMailDeliverySetting_Authentication) v1pb.SMTPMailDeliverySettingValue_Authentication {
	switch authType {
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_NONE:
		return v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_NONE
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_PLAIN:
		return v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_PLAIN
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_LOGIN:
		return v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_LOGIN
	case storepb.SMTPMailDeliverySetting_AUTHENTICATION_CRAM_MD5:
		return v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_CRAM_MD5
	}
	return v1pb.SMTPMailDeliverySettingValue_AUTHENTICATION_UNSPECIFIED
}

func convertToMailSMTPEncryptionType(encryptionType v1pb.SMTPMailDeliverySettingValue_Encryption) mail.SMTPEncryptionType {
	switch encryptionType {
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_NONE:
		return mail.SMTPEncryptionTypeNone
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_STARTTLS:
		return mail.SMTPEncryptionTypeSTARTTLS
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_SSL_TLS:
		return mail.SMTPEncryptionTypeSSLTLS
	}
	return mail.SMTPEncryptionTypeNone
}

func convertToStorePbSMTPEncryptionType(encryptionType v1pb.SMTPMailDeliverySettingValue_Encryption) storepb.SMTPMailDeliverySetting_Encryption {
	switch encryptionType {
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_NONE:
		return storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_STARTTLS:
		return storepb.SMTPMailDeliverySetting_ENCRYPTION_STARTTLS
	case v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_SSL_TLS:
		return storepb.SMTPMailDeliverySetting_ENCRYPTION_SSL_TLS
	}
	return storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE
}

func convertToSMTPEncryptionType(encryptionType storepb.SMTPMailDeliverySetting_Encryption) v1pb.SMTPMailDeliverySettingValue_Encryption {
	switch encryptionType {
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_NONE:
		return v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_NONE
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_STARTTLS:
		return v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_STARTTLS
	case storepb.SMTPMailDeliverySetting_ENCRYPTION_SSL_TLS:
		return v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_SSL_TLS
	}
	return v1pb.SMTPMailDeliverySettingValue_ENCRYPTION_UNSPECIFIED
}

// stripSensitiveData strips the sensitive data like password from the setting.value.
func stripSensitiveData(setting *v1pb.Setting) (*v1pb.Setting, error) {
	settingName, err := common.GetSettingName(setting.Name)
	if err != nil {
		return nil, err
	}
	apiSettingName := api.SettingName(settingName)
	switch apiSettingName {
	case api.SettingWorkspaceMailDelivery:
		mailDeliveryValue, ok := setting.Value.Value.(*v1pb.Value_SmtpMailDeliverySettingValue)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "invalid setting value type: %T", setting.Value.Value)
		}
		mailDeliveryValue.SmtpMailDeliverySettingValue.Password = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Ca = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Cert = nil
		mailDeliveryValue.SmtpMailDeliverySettingValue.Key = nil
		setting.Value.Value = mailDeliveryValue
	default:
	}
	return setting, nil
}

func convertSchemaTemplateSetting(template *storepb.SchemaTemplateSetting) (*v1pb.SchemaTemplateSetting, error) {
	v1Setting := new(v1pb.SchemaTemplateSetting)
	for _, v := range template.ColumnTypes {
		v1Setting.ColumnTypes = append(v1Setting.ColumnTypes, &v1pb.SchemaTemplateSetting_ColumnType{
			Engine:  convertToEngine(v.Engine),
			Enabled: v.Enabled,
			Types:   v.Types,
		})
	}
	for _, v := range template.FieldTemplates {
		if v == nil {
			continue
		}
		t := &v1pb.SchemaTemplateSetting_FieldTemplate{
			Id:       v.Id,
			Engine:   convertToEngine(v.Engine),
			Category: v.Category,
		}
		if v.Column != nil {
			t.Column = convertStoreColumnMetadata(v.Column)
		}
		if v.Catalog != nil {
			t.Catalog = convertColumnCatalog(v.Catalog)
		}
		v1Setting.FieldTemplates = append(v1Setting.FieldTemplates, t)
	}
	for _, v := range template.TableTemplates {
		if v == nil {
			continue
		}
		t := &v1pb.SchemaTemplateSetting_TableTemplate{
			Id:       v.Id,
			Engine:   convertToEngine(v.Engine),
			Category: v.Category,
		}
		if v.Table != nil {
			t.Table = convertStoreTableMetadata(v.Table)
		}
		if v.Catalog != nil {
			t.Catalog = convertTableCatalog(v.Catalog)
		}
		v1Setting.TableTemplates = append(v1Setting.TableTemplates, t)
	}

	return v1Setting, nil
}

func convertV1SchemaTemplateSetting(template *v1pb.SchemaTemplateSetting) (*storepb.SchemaTemplateSetting, error) {
	v1Setting := new(storepb.SchemaTemplateSetting)
	for _, v := range template.ColumnTypes {
		v1Setting.ColumnTypes = append(v1Setting.ColumnTypes, &storepb.SchemaTemplateSetting_ColumnType{
			Engine:  convertEngine(v.Engine),
			Enabled: v.Enabled,
			Types:   v.Types,
		})
	}
	for _, v := range template.FieldTemplates {
		if v == nil {
			continue
		}
		t := &storepb.SchemaTemplateSetting_FieldTemplate{
			Id:       v.Id,
			Engine:   convertEngine(v.Engine),
			Category: v.Category,
		}
		if v.Column != nil {
			t.Column = convertV1ColumnMetadata(v.Column)
		}
		if v.Catalog != nil {
			t.Catalog = convertV1ColumnCatalog(v.Catalog)
		}
		v1Setting.FieldTemplates = append(v1Setting.FieldTemplates, t)
	}
	for _, v := range template.TableTemplates {
		if v == nil {
			continue
		}
		t := &storepb.SchemaTemplateSetting_TableTemplate{
			Id:       v.Id,
			Engine:   convertEngine(v.Engine),
			Category: v.Category,
		}
		if v.Table != nil {
			t.Table = convertV1TableMetadata(v.Table)
		}
		if v.Catalog != nil {
			t.Catalog = convertV1TableCatalog(v.Catalog)
		}
		v1Setting.TableTemplates = append(v1Setting.TableTemplates, t)
	}

	return v1Setting, nil
}

var domainRegexp = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
var disallowedDomains = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"outlook.com":    true,
	"hotmail.com":    true,
	"live.com":       true,
	"msn.com":        true,
	"yahoo.com":      true,
	"ymail.com":      true,
	"rocketmail.com": true,
	"icloud.com":     true,
	"me.com":         true,
	"mac.com":        true,
	"aol.com":        true,
	"zoho.com":       true,
	"protonmail.com": true,
	"gmx.com":        true,
	"gmx.net":        true,
	"mail.com":       true,
	"yandex.com":     true,
	"yandex.ru":      true,
	"fastmail.com":   true,
	"fastmail.fm":    true,
	"tutanota.com":   true,
	"163.com":        true,
	"126.com":        true,
	"sohu.com":       true,
	"qq.com":         true,
	"sina.com":       true,
	"sina.cn":        true,
	"aliyun.com":     true,
	"aliyun.cn":      true,
	"tom.com":        true,
	"21cn.com":       true,
	"yeah.net":       true,
}

func validateDomains(domains []string) error {
	for _, domain := range domains {
		if !domainRegexp.MatchString(domain) {
			return errors.Errorf("invalid domain %q", domain)
		}
		if disallowedDomains[domain] {
			return errors.Errorf("domain %q is not allowed", domain)
		}
	}
	return nil
}
