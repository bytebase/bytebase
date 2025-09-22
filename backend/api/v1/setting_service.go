package v1

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	"github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	"github.com/bytebase/bytebase/backend/plugin/webhook/lark"
	"github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	"github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
	"github.com/bytebase/bytebase/backend/store"
)

// SettingService implements the setting service.
type SettingService struct {
	v1connect.UnimplementedSettingServiceHandler
	store          *store.Store
	profile        *config.Profile
	licenseService *enterprise.LicenseService
	stateCfg       *state.State
}

// NewSettingService creates a new setting service.
func NewSettingService(
	store *store.Store,
	profile *config.Profile,
	licenseService *enterprise.LicenseService,
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
var whitelistSettings = []storepb.SettingName{
	storepb.SettingName_BRANDING_LOGO,
	storepb.SettingName_WORKSPACE_ID,
	storepb.SettingName_APP_IM,
	storepb.SettingName_WATERMARK,
	storepb.SettingName_AI,
	storepb.SettingName_WORKSPACE_APPROVAL,
	storepb.SettingName_WORKSPACE_PROFILE,
	storepb.SettingName_WORKSPACE_EXTERNAL_APPROVAL,
	storepb.SettingName_SCHEMA_TEMPLATE,
	storepb.SettingName_DATA_CLASSIFICATION,
	storepb.SettingName_SEMANTIC_TYPES,
	storepb.SettingName_SCIM,
	storepb.SettingName_PASSWORD_RESTRICTION,
	storepb.SettingName_ENVIRONMENT,
}

// ListSettings lists all settings.
func (s *SettingService) ListSettings(ctx context.Context, _ *connect.Request[v1pb.ListSettingsRequest]) (*connect.Response[v1pb.ListSettingsResponse], error) {
	settings, err := s.store.ListSettingV2(ctx, &store.FindSettingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list settings: %v", err))
	}

	response := &v1pb.ListSettingsResponse{}
	for _, setting := range settings {
		if !settingInWhitelist(setting.Name) {
			continue
		}
		settingMessage, err := convertToSettingMessage(setting, s.profile)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting message: %v", err))
		}
		response.Settings = append(response.Settings, settingMessage)
	}
	return connect.NewResponse(response), nil
}

// GetSetting gets the setting by name.
func (s *SettingService) GetSetting(ctx context.Context, request *connect.Request[v1pb.GetSettingRequest]) (*connect.Response[v1pb.Setting], error) {
	settingName, err := common.GetSettingName(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting name is invalid: %v", err))
	}
	if settingName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting name is empty"))
	}
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid setting name: %v", err))
	}
	if !settingInWhitelist(apiSettingName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting is not available"))
	}

	setting, err := s.store.GetSettingV2(ctx, apiSettingName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get setting: %v", err))
	}
	if setting == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("setting %s not found", settingName))
	}
	// Only return whitelisted setting.
	settingMessage, err := convertToSettingMessage(setting, s.profile)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting message: %v", err))
	}
	return connect.NewResponse(settingMessage), nil
}

// SetSetting set the setting by name.
func (s *SettingService) UpdateSetting(ctx context.Context, request *connect.Request[v1pb.UpdateSettingRequest]) (*connect.Response[v1pb.Setting], error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	settingName, err := common.GetSettingName(request.Msg.Setting.Name)
	if err != nil {
		return nil, err
	}
	if settingName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting name is empty"))
	}
	if s.profile.IsFeatureUnavailable(settingName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
	}
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid setting name: %v", err))
	}
	existedSetting, err := s.store.GetSettingV2(ctx, apiSettingName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find setting %s with error: %v", settingName, err))
	}
	if existedSetting == nil && !request.Msg.AllowMissing {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("setting %s not found", settingName))
	}
	// audit log.
	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok && existedSetting != nil {
		v1pbSetting, err := convertToSettingMessage(existedSetting, s.profile)
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
	case storepb.SettingName_WORKSPACE_PROFILE:
		if request.Msg.UpdateMask == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask is required"))
		}
		payload := convertWorkspaceProfileSetting(request.Msg.Setting.Value.GetWorkspaceProfileSettingValue())
		oldSetting, err := s.store.GetWorkspaceGeneralSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find setting %s with error: %v", apiSettingName, err))
		}

		for _, path := range request.Msg.UpdateMask.Paths {
			switch path {
			case "value.workspace_profile_setting_value.disallow_signup":
				if s.profile.SaaS {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
				}
				if payload.DisallowSignup {
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.DisallowSignup = payload.DisallowSignup
			case "value.workspace_profile_setting_value.external_url":
				if s.profile.SaaS {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
				}
				if payload.ExternalUrl != "" {
					externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
					if err != nil {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid external url: %v", err))
					}
					payload.ExternalUrl = externalURL
				}
				oldSetting.ExternalUrl = payload.ExternalUrl
			case "value.workspace_profile_setting_value.require_2fa":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TWO_FA); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.Require_2Fa = payload.Require_2Fa
			case "value.workspace_profile_setting_value.token_duration":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				if payload.TokenDuration != nil && payload.TokenDuration.Seconds > 0 && payload.TokenDuration.AsDuration() < time.Hour {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("refresh token duration should be at least one hour"))
				}
				oldSetting.TokenDuration = payload.TokenDuration
			case "value.workspace_profile_setting_value.announcement":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DASHBOARD_ANNOUNCEMENT); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
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
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid domains, error %v", err))
				}
				oldSetting.Domains = payload.Domains
			case "value.workspace_profile_setting_value.enforce_identity_domain":
				if payload.EnforceIdentityDomain {
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_USER_EMAIL_DOMAIN_RESTRICTION); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.EnforceIdentityDomain = payload.EnforceIdentityDomain
			case "value.workspace_profile_setting_value.database_change_mode":
				oldSetting.DatabaseChangeMode = payload.DatabaseChangeMode
			case "value.workspace_profile_setting_value.disallow_password_signin":
				if payload.DisallowPasswordSignin {
					// We should still allow users to turn it off.
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_PASSWORD_SIGNIN); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}

					identityProviders, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list identity providers: %v", err))
					}
					if len(identityProviders) == 0 {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("cannot disallow password signin when no identity provider is set"))
					}
				}
				oldSetting.DisallowPasswordSignin = payload.DisallowPasswordSignin
			case "value.workspace_profile_setting_value.enable_metric_collection":
				oldSetting.EnableMetricCollection = payload.EnableMetricCollection
			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %v", path))
			}
		}

		if len(oldSetting.Domains) == 0 && oldSetting.EnforceIdentityDomain {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity domain can be enforced only when workspace domains are set"))
		}
		bytes, err := protojson.Marshal(oldSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_WORKSPACE_APPROVAL:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		payload := &storepb.WorkspaceApprovalSetting{}
		for _, rule := range request.Msg.Setting.Value.GetWorkspaceApprovalSettingValue().Rules {
			// Validate the condition.
			if _, err := common.ConvertUnparsedApproval(rule.Condition); err != nil {
				return nil, err
			}
			if err := validateApprovalTemplate(rule.Template); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid approval template: %v, err: %v", rule.Template, err))
			}

			flow := convertApprovalFlow(rule.Template.Flow)
			payload.Rules = append(payload.Rules, &storepb.WorkspaceApprovalSetting_Rule{
				Condition: rule.Condition,
				Template: &storepb.ApprovalTemplate{
					Flow:        flow,
					Title:       rule.Template.Title,
					Description: rule.Template.Description,
				},
			})
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_BRANDING_LOGO:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_CUSTOM_LOGO); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		storeSettingValue = request.Msg.Setting.Value.GetStringValue()

	case storepb.SettingName_APP_IM:
		payload := convertAppIMSetting(request.Msg.Setting.Value.GetAppImSettingValue())
		setting, err := s.store.GetAppIMSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get old app im setting"))
		}
		if request.Msg.UpdateMask == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask is required"))
		}
		for _, path := range request.Msg.UpdateMask.Paths {
			switch path {
			case "value.app_im_setting_value.slack":
				if err := slack.ValidateToken(ctx, payload.Slack.GetToken()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}
				setting.Slack = payload.Slack

			case "value.app_im_setting_value.feishu":
				if err := feishu.Validate(ctx, payload.GetFeishu().GetAppId(), payload.GetFeishu().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}
				setting.Feishu = payload.Feishu

			case "value.app_im_setting_value.wecom":
				if err := wecom.Validate(ctx, payload.GetWecom().GetCorpId(), payload.GetWecom().GetAgentId(), payload.GetWecom().GetSecret()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}
				setting.Wecom = payload.Wecom

			case "value.app_im_setting_value.lark":
				if err := lark.Validate(ctx, payload.GetLark().GetAppId(), payload.GetLark().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}
				setting.Lark = payload.Lark
			case "value.app_im_setting_value.dingtalk":
				if err := dingtalk.Validate(ctx, payload.GetDingtalk().GetClientId(), payload.GetDingtalk().GetClientSecret(), payload.GetDingtalk().RobotCode, user.Phone); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}
				setting.Dingtalk = payload.Dingtalk

			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %v", path))
			}
		}
		if request.Msg.ValidateOnly {
			return connect.NewResponse(&v1pb.Setting{
				Name: request.Msg.Setting.Name,
				Value: &v1pb.Value{
					Value: &v1pb.Value_AppImSettingValue{
						AppImSettingValue: &v1pb.AppIMSetting{},
					},
				},
			}), nil
		}

		bytes, err := protojson.Marshal(setting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s, error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)

	case storepb.SettingName_SCHEMA_TEMPLATE:
		schemaTemplateSetting := request.Msg.Setting.Value.GetSchemaTemplateSettingValue()
		if schemaTemplateSetting == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("value cannot be nil when setting schema template setting"))
		}

		if err := s.validateSchemaTemplate(ctx, schemaTemplateSetting); err != nil {
			return nil, err
		}

		payload := convertV1SchemaTemplateSetting(schemaTemplateSetting)
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal external approval setting, error: %v", err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_DATA_CLASSIFICATION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_CLASSIFICATION); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertDataClassificationSetting(request.Msg.Setting.Value.GetDataClassificationSettingValue())
		// it's a temporary solution to limit only 1 classification config before we support manage it in the UX.
		if len(payload.Configs) > 1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("only support define 1 classification config for now"))
		}
		if len(payload.Configs) == 1 && len(payload.Configs[0].Classification) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing classification map"))
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_SEMANTIC_TYPES:
		storeSemanticTypeSetting := convertSemanticTypeSetting(request.Msg.Setting.Value.GetSemanticTypeSettingValue())
		idMap := make(map[string]bool)
		for _, tp := range storeSemanticTypeSetting.Types {
			if tp.Title == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("category title cannot be empty: %s", tp.Id))
			}
			if idMap[tp.Id] {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("duplicate semantic type id: %s", tp.Id))
			}
			m, ok := tp.GetAlgorithm().GetMask().(*storepb.Algorithm_InnerOuterMask_)
			if ok && m.InnerOuterMask != nil {
				if m.InnerOuterMask.Type == storepb.Algorithm_InnerOuterMask_MASK_TYPE_UNSPECIFIED {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("inner outer mask type has to be specified"))
				}
			}
			idMap[tp.Id] = true
		}
		bytes, err := protojson.Marshal(storeSemanticTypeSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_WATERMARK:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_WATERMARK); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		storeSettingValue = request.Msg.Setting.Value.GetStringValue()
	case storepb.SettingName_SCIM:
		scimToken, err := common.RandomString(32)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate random SCIM secret with error: %v", err))
		}
		bytes, err := protojson.Marshal(&storepb.SCIMSetting{
			Token: scimToken,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal SCIM setting with error: %v", err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_PASSWORD_RESTRICTION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_PASSWORD_RESTRICTIONS); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		passwordSetting := convertPasswordRestrictionSetting(request.Msg.Setting.Value.GetPasswordRestrictionSetting())
		if passwordSetting.MinLength < 8 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid password minimum length, should no less than 8"))
		}
		bytes, err := protojson.Marshal(passwordSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_AI:
		aiSetting := convertAISetting(request.Msg.Setting.Value.GetAiSetting())
		if aiSetting.Enabled {
			if aiSetting.Endpoint == "" || aiSetting.Model == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("API endpoint and model are required"))
			}
			if existedSetting != nil {
				existedAISetting, err := convertToSettingMessage(existedSetting, s.profile)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal existed ai setting with error: %v", err))
				}
				if aiSetting.ApiKey == "" {
					aiSetting.ApiKey = existedAISetting.Value.GetAiSetting().GetApiKey()
				}
			}
			if aiSetting.ApiKey == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("API key is required"))
			}
		}

		bytes, err := protojson.Marshal(aiSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_ENVIRONMENT:
		if serr := s.validateEnvironments(request.Msg.Setting.Value.GetEnvironmentSetting().GetEnvironments()); serr != nil {
			return nil, serr
		}

		environmentSetting := convertEnvironmentSetting(request.Msg.Setting.Value.GetEnvironmentSetting())
		oldEnvironmentSetting, err := s.store.GetEnvironmentSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get old environment setting with error: %v", err))
		}
		newEnvIDMap := map[string]bool{}
		for _, env := range environmentSetting.Environments {
			newEnvIDMap[env.Id] = true
		}
		for _, env := range oldEnvironmentSetting.Environments {
			if !newEnvIDMap[env.Id] {
				// deleted
				emptyStr := ""
				if _, err := s.store.UpdateInstanceV2(ctx, &store.UpdateInstanceMessage{
					EnvironmentID:       &emptyStr,
					FindByEnvironmentID: &env.Id,
				}); err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to unset environment %v for instances", env.Id))
				}
				if _, err := s.store.BatchUpdateDatabases(ctx, nil, &store.BatchUpdateDatabases{
					EnvironmentID:       &emptyStr,
					FindByEnvironmentID: &env.Id,
				}); err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to unset environment %v for databases", env.Id))
				}
			}
		}

		bytes, err := protojson.Marshal(environmentSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	default:
		storeSettingValue = request.Msg.Setting.Value.GetStringValue()
	}
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to set setting: %v", err))
	}

	settingMessage, err := convertToSettingMessage(setting, s.profile)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting message: %v", err))
	}

	// it's a temporary solution to map the classification to all projects before we support it in the UX.
	if apiSettingName == storepb.SettingName_DATA_CLASSIFICATION && len(settingMessage.Value.GetDataClassificationSettingValue().Configs) == 1 {
		classificationID := settingMessage.Value.GetDataClassificationSettingValue().Configs[0].Id
		projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{ShowDeleted: false})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list projects with error: %v", err))
		}
		for _, project := range projects {
			patch := &store.UpdateProjectMessage{
				ResourceID:                 project.ResourceID,
				DataClassificationConfigID: &classificationID,
			}
			if _, err = s.store.UpdateProjectV2(ctx, patch); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to patch project %s with error: %v", project.Title, err))
			}
		}
	}

	return connect.NewResponse(settingMessage), nil
}

func convertToSettingMessage(setting *store.SettingMessage, profile *config.Profile) (*v1pb.Setting, error) {
	settingName := fmt.Sprintf("%s%s", common.SettingNamePrefix, convertStoreSettingNameToV1(setting.Name).String())
	switch setting.Name {
	case storepb.SettingName_APP_IM:
		storeValue := new(storepb.AppIMSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AppImSettingValue{
					AppImSettingValue: &v1pb.AppIMSetting{
						Slack: &v1pb.AppIMSetting_Slack{
							Enabled: storeValue.GetSlack().GetEnabled(),
						},
						Feishu: &v1pb.AppIMSetting_Feishu{
							Enabled: storeValue.GetFeishu().GetEnabled(),
						},
						Wecom: &v1pb.AppIMSetting_Wecom{
							Enabled: storeValue.GetWecom().GetEnabled(),
						},
						Lark: &v1pb.AppIMSetting_Lark{
							Enabled: storeValue.GetLark().GetEnabled(),
						},
						Dingtalk: &v1pb.AppIMSetting_DingTalk{
							Enabled: storeValue.GetDingtalk().GetEnabled(),
						},
					},
				},
			},
		}, nil
	case storepb.SettingName_WORKSPACE_PROFILE:
		storeValue := new(storepb.WorkspaceProfileSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		v1Value := convertToWorkspaceProfileSetting(storeValue)
		v1Value.DisallowSignup = v1Value.DisallowSignup || profile.SaaS
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_WorkspaceProfileSettingValue{
					WorkspaceProfileSettingValue: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_WORKSPACE_APPROVAL:
		storeValue := new(storepb.WorkspaceApprovalSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		v1Value := &v1pb.WorkspaceApprovalSetting{}
		for _, rule := range storeValue.Rules {
			template := convertToApprovalTemplate(rule.Template)
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
	case storepb.SettingName_SCHEMA_TEMPLATE:
		storeValue := new(storepb.SchemaTemplateSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}

		sts := convertToSchemaTemplateSetting(storeValue)
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SchemaTemplateSettingValue{
					SchemaTemplateSettingValue: sts,
				},
			},
		}, nil
	case storepb.SettingName_DATA_CLASSIFICATION:
		storeValue := new(storepb.DataClassificationSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_DataClassificationSettingValue{
					DataClassificationSettingValue: convertToDataClassificationSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_SEMANTIC_TYPES:
		storeValue := new(storepb.SemanticTypeSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SemanticTypeSettingValue{
					SemanticTypeSettingValue: convertToSemanticTypeSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_SCIM:
		storeValue := new(storepb.SCIMSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_ScimSetting{
					ScimSetting: convertToSCIMSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_PASSWORD_RESTRICTION:
		storeValue := new(storepb.PasswordRestrictionSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_PasswordRestrictionSetting{
					PasswordRestrictionSetting: convertToPasswordRestrictionSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_AI:
		storeValue := new(storepb.AISetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		// DO NOT expose the api key.
		storeValue.ApiKey = ""
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AiSetting{
					AiSetting: convertToAISetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_ENVIRONMENT:
		storeValue, err := convertToEnvironmentSetting(setting.Value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_EnvironmentSetting{
					EnvironmentSetting: storeValue,
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
	oldStoreSetting, err := s.store.GetSettingV2(ctx, storepb.SettingName_SCHEMA_TEMPLATE)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to get setting %q: %v", storepb.SettingName_SCHEMA_TEMPLATE, err))
	}
	settingValue := "{}"
	if oldStoreSetting != nil {
		settingValue = oldStoreSetting.Value
	}

	value := new(storepb.SchemaTemplateSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(settingValue), value); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %v with error: %v", storepb.SettingName_SCHEMA_TEMPLATE, err))
	}
	v1Value := convertToSchemaTemplateSetting(value)

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
	tempStoreSchemaMetadata := convertV1DatabaseMetadata(tempMetadata)
	if err := checkDatabaseMetadata(storepb.Engine(engine), tempStoreSchemaMetadata); err != nil {
		return errors.Wrap(err, "failed to check database metadata")
	}
	if _, err := schema.GetDatabaseDefinition(storepb.Engine(engine), schema.GetDefinitionContext{}, tempStoreSchemaMetadata); err != nil {
		return errors.Wrap(err, "failed to transform database metadata to schema string")
	}
	return nil
}

func settingInWhitelist(name storepb.SettingName) bool {
	for _, whitelist := range whitelistSettings {
		if name == whitelist {
			return true
		}
	}
	return false
}

// convertStringToSettingName converts a string to storepb.SettingName.
// It first converts to v1pb.Setting_SettingName, then to storepb.SettingName.
func convertStringToSettingName(name string) (storepb.SettingName, error) {
	// First try to convert string to v1pb.Setting_SettingName
	v1Value, ok := v1pb.Setting_SettingName_value[name]
	if !ok {
		return storepb.SettingName_SETTING_NAME_UNSPECIFIED, errors.Errorf("invalid setting name: %s", name)
	}
	v1SettingName := v1pb.Setting_SettingName(v1Value)

	// Then convert v1pb to storepb
	return convertV1SettingNameToStore(v1SettingName), nil
}

// convertStoreSettingNameToV1 converts storepb.SettingName to v1pb.Setting_SettingName.
func convertStoreSettingNameToV1(storeName storepb.SettingName) v1pb.Setting_SettingName {
	//exhaustive:enforce
	switch storeName {
	case storepb.SettingName_SETTING_NAME_UNSPECIFIED:
		return v1pb.Setting_SETTING_NAME_UNSPECIFIED
	case storepb.SettingName_AUTH_SECRET:
		return v1pb.Setting_AUTH_SECRET
	case storepb.SettingName_BRANDING_LOGO:
		return v1pb.Setting_BRANDING_LOGO
	case storepb.SettingName_WORKSPACE_ID:
		return v1pb.Setting_WORKSPACE_ID
	case storepb.SettingName_WORKSPACE_PROFILE:
		return v1pb.Setting_WORKSPACE_PROFILE
	case storepb.SettingName_WORKSPACE_APPROVAL:
		return v1pb.Setting_WORKSPACE_APPROVAL
	case storepb.SettingName_WORKSPACE_EXTERNAL_APPROVAL:
		return v1pb.Setting_WORKSPACE_EXTERNAL_APPROVAL
	case storepb.SettingName_ENTERPRISE_LICENSE:
		return v1pb.Setting_ENTERPRISE_LICENSE
	case storepb.SettingName_APP_IM:
		return v1pb.Setting_APP_IM
	case storepb.SettingName_WATERMARK:
		return v1pb.Setting_WATERMARK
	case storepb.SettingName_AI:
		return v1pb.Setting_AI
	case storepb.SettingName_SCHEMA_TEMPLATE:
		return v1pb.Setting_SCHEMA_TEMPLATE
	case storepb.SettingName_DATA_CLASSIFICATION:
		return v1pb.Setting_DATA_CLASSIFICATION
	case storepb.SettingName_SEMANTIC_TYPES:
		return v1pb.Setting_SEMANTIC_TYPES
	case storepb.SettingName_SCIM:
		return v1pb.Setting_SCIM
	case storepb.SettingName_PASSWORD_RESTRICTION:
		return v1pb.Setting_PASSWORD_RESTRICTION
	case storepb.SettingName_ENVIRONMENT:
		return v1pb.Setting_ENVIRONMENT
	default:
	}
	return v1pb.Setting_SETTING_NAME_UNSPECIFIED
}

// convertV1SettingNameToStore converts v1pb.Setting_SettingName to storepb.SettingName.
func convertV1SettingNameToStore(v1Name v1pb.Setting_SettingName) storepb.SettingName {
	//exhaustive:enforce
	switch v1Name {
	case v1pb.Setting_SETTING_NAME_UNSPECIFIED:
		return storepb.SettingName_SETTING_NAME_UNSPECIFIED
	case v1pb.Setting_AUTH_SECRET:
		return storepb.SettingName_AUTH_SECRET
	case v1pb.Setting_BRANDING_LOGO:
		return storepb.SettingName_BRANDING_LOGO
	case v1pb.Setting_WORKSPACE_ID:
		return storepb.SettingName_WORKSPACE_ID
	case v1pb.Setting_WORKSPACE_PROFILE:
		return storepb.SettingName_WORKSPACE_PROFILE
	case v1pb.Setting_WORKSPACE_APPROVAL:
		return storepb.SettingName_WORKSPACE_APPROVAL
	case v1pb.Setting_WORKSPACE_EXTERNAL_APPROVAL:
		return storepb.SettingName_WORKSPACE_EXTERNAL_APPROVAL
	case v1pb.Setting_ENTERPRISE_LICENSE:
		return storepb.SettingName_ENTERPRISE_LICENSE
	case v1pb.Setting_APP_IM:
		return storepb.SettingName_APP_IM
	case v1pb.Setting_WATERMARK:
		return storepb.SettingName_WATERMARK
	case v1pb.Setting_AI:
		return storepb.SettingName_AI
	case v1pb.Setting_SCHEMA_TEMPLATE:
		return storepb.SettingName_SCHEMA_TEMPLATE
	case v1pb.Setting_DATA_CLASSIFICATION:
		return storepb.SettingName_DATA_CLASSIFICATION
	case v1pb.Setting_SEMANTIC_TYPES:
		return storepb.SettingName_SEMANTIC_TYPES
	case v1pb.Setting_SCIM:
		return storepb.SettingName_SCIM
	case v1pb.Setting_PASSWORD_RESTRICTION:
		return storepb.SettingName_PASSWORD_RESTRICTION
	case v1pb.Setting_ENVIRONMENT:
		return storepb.SettingName_ENVIRONMENT
	default:
		return storepb.SettingName_SETTING_NAME_UNSPECIFIED
	}
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

func convertToSchemaTemplateSetting(template *storepb.SchemaTemplateSetting) *v1pb.SchemaTemplateSetting {
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

	return v1Setting
}

func convertV1SchemaTemplateSetting(template *v1pb.SchemaTemplateSetting) *storepb.SchemaTemplateSetting {
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

	return v1Setting
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

func (s *SettingService) validateEnvironments(envs []*v1pb.EnvironmentSetting_Environment) error {
	used := map[string]bool{}
	for _, env := range envs {
		if env.Title == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("environment title cannot be empty"))
		}
		if !isValidResourceID(env.Id) {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment ID %v", env.Id))
		}
		if used[env.Id] {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("duplicate environment ID %v", env.Id))
		}
		if v, ok := env.Tags["protected"]; ok && v == "protected" {
			if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_ENVIRONMENT_TIERS); err != nil {
				return connect.NewError(connect.CodePermissionDenied, err)
			}
		}
		used[env.Id] = true
	}
	return nil
}

func convertToEnvironmentSetting(value string) (*v1pb.EnvironmentSetting, error) {
	var setting storepb.EnvironmentSetting
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(value), &setting); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal environment setting")
	}
	var environments []*v1pb.EnvironmentSetting_Environment

	for _, e := range setting.Environments {
		environments = append(environments, convertToEnvironment(e))
	}
	return &v1pb.EnvironmentSetting{
		Environments: environments,
	}, nil
}

func convertToEnvironment(e *storepb.EnvironmentSetting_Environment) *v1pb.EnvironmentSetting_Environment {
	return &v1pb.EnvironmentSetting_Environment{
		Name:  common.FormatEnvironment(e.Id),
		Id:    e.Id,
		Title: e.Title,
		Tags:  e.Tags,
		Color: e.Color,
	}
}

func convertEnvironmentSetting(e *v1pb.EnvironmentSetting) *storepb.EnvironmentSetting {
	var environments []*storepb.EnvironmentSetting_Environment
	for _, env := range e.Environments {
		environments = append(environments, &storepb.EnvironmentSetting_Environment{
			Id:    env.Id,
			Title: env.Title,
			Tags:  env.Tags,
			Color: env.Color,
		})
	}
	return &storepb.EnvironmentSetting{
		Environments: environments,
	}
}

func convertWorkspaceProfileSetting(v1Setting *v1pb.WorkspaceProfileSetting) *storepb.WorkspaceProfileSetting {
	if v1Setting == nil {
		return nil
	}

	storeSetting := &storepb.WorkspaceProfileSetting{
		ExternalUrl:            v1Setting.ExternalUrl,
		DisallowSignup:         v1Setting.DisallowSignup,
		Require_2Fa:            v1Setting.Require_2Fa,
		TokenDuration:          v1Setting.TokenDuration,
		MaximumRoleExpiration:  v1Setting.MaximumRoleExpiration,
		Domains:                v1Setting.Domains,
		EnforceIdentityDomain:  v1Setting.EnforceIdentityDomain,
		DatabaseChangeMode:     storepb.DatabaseChangeMode(v1Setting.DatabaseChangeMode),
		DisallowPasswordSignin: v1Setting.DisallowPasswordSignin,
		EnableMetricCollection: v1Setting.EnableMetricCollection,
	}

	// Convert announcement if present
	if v1Setting.Announcement != nil {
		storeSetting.Announcement = &storepb.Announcement{
			Text: v1Setting.Announcement.Text,
			Link: v1Setting.Announcement.Link,
		}
		// Convert alert level
		switch v1Setting.Announcement.Level {
		case v1pb.Announcement_ALERT_LEVEL_UNSPECIFIED:
			storeSetting.Announcement.Level = storepb.Announcement_ALERT_LEVEL_UNSPECIFIED
		case v1pb.Announcement_INFO:
			storeSetting.Announcement.Level = storepb.Announcement_ALERT_LEVEL_INFO
		case v1pb.Announcement_WARNING:
			storeSetting.Announcement.Level = storepb.Announcement_ALERT_LEVEL_WARNING
		case v1pb.Announcement_CRITICAL:
			storeSetting.Announcement.Level = storepb.Announcement_ALERT_LEVEL_CRITICAL
		default:
		}
	}

	return storeSetting
}

func convertToWorkspaceProfileSetting(storeSetting *storepb.WorkspaceProfileSetting) *v1pb.WorkspaceProfileSetting {
	if storeSetting == nil {
		return nil
	}

	v1Setting := &v1pb.WorkspaceProfileSetting{
		ExternalUrl:            storeSetting.ExternalUrl,
		DisallowSignup:         storeSetting.DisallowSignup,
		Require_2Fa:            storeSetting.Require_2Fa,
		TokenDuration:          storeSetting.TokenDuration,
		MaximumRoleExpiration:  storeSetting.MaximumRoleExpiration,
		Domains:                storeSetting.Domains,
		EnforceIdentityDomain:  storeSetting.EnforceIdentityDomain,
		DatabaseChangeMode:     v1pb.DatabaseChangeMode(storeSetting.DatabaseChangeMode),
		DisallowPasswordSignin: storeSetting.DisallowPasswordSignin,
		EnableMetricCollection: storeSetting.EnableMetricCollection,
	}

	if storeSetting.Announcement != nil {
		v1Setting.Announcement = &v1pb.Announcement{
			Text: storeSetting.Announcement.Text,
			Link: storeSetting.Announcement.Link,
		}
		switch storeSetting.Announcement.Level {
		case storepb.Announcement_ALERT_LEVEL_UNSPECIFIED:
			v1Setting.Announcement.Level = v1pb.Announcement_ALERT_LEVEL_UNSPECIFIED
		case storepb.Announcement_ALERT_LEVEL_INFO:
			v1Setting.Announcement.Level = v1pb.Announcement_INFO
		case storepb.Announcement_ALERT_LEVEL_WARNING:
			v1Setting.Announcement.Level = v1pb.Announcement_WARNING
		case storepb.Announcement_ALERT_LEVEL_CRITICAL:
			v1Setting.Announcement.Level = v1pb.Announcement_CRITICAL
		default:
		}
	}

	return v1Setting
}

func convertApprovalFlow(v1Flow *v1pb.ApprovalFlow) *storepb.ApprovalFlow {
	if v1Flow == nil {
		return nil
	}

	storeFlow := &storepb.ApprovalFlow{}
	for _, step := range v1Flow.Steps {
		storeFlow.Steps = append(storeFlow.Steps, convertApprovalStep(step))
	}
	return storeFlow
}

func convertApprovalStep(v1Step *v1pb.ApprovalStep) *storepb.ApprovalStep {
	if v1Step == nil {
		return nil
	}

	storeStep := &storepb.ApprovalStep{
		Type: storepb.ApprovalStep_Type(v1Step.Type),
	}
	for _, node := range v1Step.Nodes {
		storeStep.Nodes = append(storeStep.Nodes, convertApprovalNode(node))
	}
	return storeStep
}

func convertApprovalNode(v1Node *v1pb.ApprovalNode) *storepb.ApprovalNode {
	if v1Node == nil {
		return nil
	}

	storeNode := &storepb.ApprovalNode{
		Type: storepb.ApprovalNode_Type(v1Node.Type),
		Role: v1Node.Role,
	}

	return storeNode
}

func convertAppIMSetting(v1Setting *v1pb.AppIMSetting) *storepb.AppIMSetting {
	if v1Setting == nil {
		return nil
	}

	storeSetting := &storepb.AppIMSetting{}

	if v1Setting.Slack != nil {
		storeSetting.Slack = &storepb.AppIMSetting_Slack{
			Enabled: v1Setting.Slack.Enabled,
			Token:   v1Setting.Slack.Token,
		}
	}
	if v1Setting.Feishu != nil {
		storeSetting.Feishu = &storepb.AppIMSetting_Feishu{
			Enabled:   v1Setting.Feishu.Enabled,
			AppId:     v1Setting.Feishu.AppId,
			AppSecret: v1Setting.Feishu.AppSecret,
		}
	}
	if v1Setting.Wecom != nil {
		storeSetting.Wecom = &storepb.AppIMSetting_Wecom{
			Enabled: v1Setting.Wecom.Enabled,
			CorpId:  v1Setting.Wecom.CorpId,
			AgentId: v1Setting.Wecom.AgentId,
			Secret:  v1Setting.Wecom.Secret,
		}
	}
	if v1Setting.Lark != nil {
		storeSetting.Lark = &storepb.AppIMSetting_Lark{
			Enabled:   v1Setting.Lark.Enabled,
			AppId:     v1Setting.Lark.AppId,
			AppSecret: v1Setting.Lark.AppSecret,
		}
	}
	if v1Setting.Dingtalk != nil {
		storeSetting.Dingtalk = &storepb.AppIMSetting_DingTalk{
			Enabled:      v1Setting.Dingtalk.Enabled,
			ClientId:     v1Setting.Dingtalk.ClientId,
			ClientSecret: v1Setting.Dingtalk.ClientSecret,
			RobotCode:    v1Setting.Dingtalk.RobotCode,
		}
	}

	return storeSetting
}

func convertDataClassificationSetting(v1Setting *v1pb.DataClassificationSetting) *storepb.DataClassificationSetting {
	if v1Setting == nil {
		return nil
	}

	storeSetting := &storepb.DataClassificationSetting{}
	for _, config := range v1Setting.Configs {
		storeConfig := convertDataClassificationSettingConfig(config)
		storeSetting.Configs = append(storeSetting.Configs, storeConfig)
	}
	return storeSetting
}

func convertDataClassificationSettingConfig(c *v1pb.DataClassificationSetting_DataClassificationConfig) *storepb.DataClassificationSetting_DataClassificationConfig {
	if c == nil {
		return nil
	}

	return &storepb.DataClassificationSetting_DataClassificationConfig{
		Id:                       c.Id,
		Title:                    c.Title,
		Levels:                   convertDataClassificationSettingLevels(c.Levels),
		Classification:           convertDataClassificationSettingClassification(c.Classification),
		ClassificationFromConfig: c.ClassificationFromConfig,
	}
}

func convertDataClassificationSettingLevels(levels []*v1pb.DataClassificationSetting_DataClassificationConfig_Level) []*storepb.DataClassificationSetting_DataClassificationConfig_Level {
	if levels == nil {
		return nil
	}

	storeLevels := make([]*storepb.DataClassificationSetting_DataClassificationConfig_Level, len(levels))
	for i, level := range levels {
		storeLevels[i] = &storepb.DataClassificationSetting_DataClassificationConfig_Level{
			Id:          level.Id,
			Title:       level.Title,
			Description: level.Description,
		}
	}
	return storeLevels
}

func convertDataClassificationSettingClassification(classification map[string]*v1pb.DataClassificationSetting_DataClassificationConfig_DataClassification) map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification {
	if classification == nil {
		return nil
	}

	storeClassification := make(map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification, len(classification))
	for k, v := range classification {
		storeClassification[k] = &storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
			Id:          v.Id,
			Title:       v.Title,
			Description: v.Description,
			LevelId:     v.LevelId,
		}
	}
	return storeClassification
}

func convertToDataClassificationSetting(storeSetting *storepb.DataClassificationSetting) *v1pb.DataClassificationSetting {
	if storeSetting == nil {
		return nil
	}

	v1Setting := &v1pb.DataClassificationSetting{}
	for _, config := range storeSetting.Configs {
		v1Config := convertToDataClassificationSettingConfig(config)
		v1Setting.Configs = append(v1Setting.Configs, v1Config)
	}
	return v1Setting
}

func convertToDataClassificationSettingConfig(c *storepb.DataClassificationSetting_DataClassificationConfig) *v1pb.DataClassificationSetting_DataClassificationConfig {
	if c == nil {
		return nil
	}

	return &v1pb.DataClassificationSetting_DataClassificationConfig{
		Id:                       c.Id,
		Title:                    c.Title,
		Levels:                   convertToDataClassificationSettingLevels(c.Levels),
		Classification:           convertToDataClassificationSettingClassification(c.Classification),
		ClassificationFromConfig: c.ClassificationFromConfig,
	}
}

func convertToDataClassificationSettingLevels(levels []*storepb.DataClassificationSetting_DataClassificationConfig_Level) []*v1pb.DataClassificationSetting_DataClassificationConfig_Level {
	if levels == nil {
		return nil
	}

	v1Levels := make([]*v1pb.DataClassificationSetting_DataClassificationConfig_Level, len(levels))
	for i, level := range levels {
		v1Levels[i] = &v1pb.DataClassificationSetting_DataClassificationConfig_Level{
			Id:          level.Id,
			Title:       level.Title,
			Description: level.Description,
		}
	}
	return v1Levels
}

func convertToDataClassificationSettingClassification(classification map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification) map[string]*v1pb.DataClassificationSetting_DataClassificationConfig_DataClassification {
	if classification == nil {
		return nil
	}

	v1Classification := make(map[string]*v1pb.DataClassificationSetting_DataClassificationConfig_DataClassification, len(classification))
	for k, v := range classification {
		v1Classification[k] = &v1pb.DataClassificationSetting_DataClassificationConfig_DataClassification{
			Id:          v.Id,
			Title:       v.Title,
			Description: v.Description,
			LevelId:     v.LevelId,
		}
	}
	return v1Classification
}

func convertSemanticTypeSetting(v1Setting *v1pb.SemanticTypeSetting) *storepb.SemanticTypeSetting {
	if v1Setting == nil {
		return nil
	}

	storeSetting := &storepb.SemanticTypeSetting{}
	for _, v1Type := range v1Setting.Types {
		storeType := &storepb.SemanticTypeSetting_SemanticType{
			Id:          v1Type.Id,
			Title:       v1Type.Title,
			Description: v1Type.Description,
			Algorithm:   convertAlgorithm(v1Type.Algorithm),
			Icon:        v1Type.Icon,
		}
		storeSetting.Types = append(storeSetting.Types, storeType)
	}
	return storeSetting
}

func convertToSemanticTypeSetting(storeSetting *storepb.SemanticTypeSetting) *v1pb.SemanticTypeSetting {
	if storeSetting == nil {
		return nil
	}

	v1Setting := &v1pb.SemanticTypeSetting{}
	for _, storeType := range storeSetting.Types {
		v1Type := &v1pb.SemanticTypeSetting_SemanticType{
			Id:          storeType.Id,
			Title:       storeType.Title,
			Description: storeType.Description,
			Algorithm:   convertToAlgorithm(storeType.Algorithm),
			Icon:        storeType.Icon,
		}
		v1Setting.Types = append(v1Setting.Types, v1Type)
	}
	return v1Setting
}

func convertAlgorithm(v1Algo *v1pb.Algorithm) *storepb.Algorithm {
	if v1Algo == nil {
		return nil
	}

	storeAlgo := &storepb.Algorithm{}
	switch mask := v1Algo.Mask.(type) {
	case *v1pb.Algorithm_FullMask_:
		storeAlgo.Mask = &storepb.Algorithm_FullMask_{
			FullMask: &storepb.Algorithm_FullMask{
				Substitution: mask.FullMask.Substitution,
			},
		}
	case *v1pb.Algorithm_Md5Mask:
		storeAlgo.Mask = &storepb.Algorithm_Md5Mask{
			Md5Mask: &storepb.Algorithm_MD5Mask{
				Salt: mask.Md5Mask.Salt,
			},
		}
	case *v1pb.Algorithm_RangeMask_:
		storeAlgo.Mask = &storepb.Algorithm_RangeMask_{
			RangeMask: &storepb.Algorithm_RangeMask{
				Slices: convertAlgorithmRangeMaskSlices(mask.RangeMask.Slices),
			},
		}
	case *v1pb.Algorithm_InnerOuterMask_:
		storeAlgo.Mask = &storepb.Algorithm_InnerOuterMask_{
			InnerOuterMask: &storepb.Algorithm_InnerOuterMask{
				PrefixLen:    mask.InnerOuterMask.PrefixLen,
				SuffixLen:    mask.InnerOuterMask.SuffixLen,
				Type:         storepb.Algorithm_InnerOuterMask_MaskType(mask.InnerOuterMask.Type),
				Substitution: mask.InnerOuterMask.Substitution,
			},
		}
	}
	return storeAlgo
}

func convertToAlgorithm(storeAlgo *storepb.Algorithm) *v1pb.Algorithm {
	if storeAlgo == nil {
		return nil
	}

	v1Algo := &v1pb.Algorithm{}
	switch mask := storeAlgo.Mask.(type) {
	case *storepb.Algorithm_FullMask_:
		v1Algo.Mask = &v1pb.Algorithm_FullMask_{
			FullMask: &v1pb.Algorithm_FullMask{
				Substitution: mask.FullMask.Substitution,
			},
		}
	case *storepb.Algorithm_Md5Mask:
		v1Algo.Mask = &v1pb.Algorithm_Md5Mask{
			Md5Mask: &v1pb.Algorithm_MD5Mask{
				Salt: mask.Md5Mask.Salt,
			},
		}
	case *storepb.Algorithm_RangeMask_:
		v1Algo.Mask = &v1pb.Algorithm_RangeMask_{
			RangeMask: &v1pb.Algorithm_RangeMask{
				Slices: convertToAlgorithmRangeMaskSlices(mask.RangeMask.Slices),
			},
		}
	case *storepb.Algorithm_InnerOuterMask_:
		v1Algo.Mask = &v1pb.Algorithm_InnerOuterMask_{
			InnerOuterMask: &v1pb.Algorithm_InnerOuterMask{
				PrefixLen:    mask.InnerOuterMask.PrefixLen,
				SuffixLen:    mask.InnerOuterMask.SuffixLen,
				Type:         v1pb.Algorithm_InnerOuterMask_MaskType(mask.InnerOuterMask.Type),
				Substitution: mask.InnerOuterMask.Substitution,
			},
		}
	}
	return v1Algo
}

func convertAlgorithmRangeMaskSlices(v1Slices []*v1pb.Algorithm_RangeMask_Slice) []*storepb.Algorithm_RangeMask_Slice {
	var storeSlices []*storepb.Algorithm_RangeMask_Slice
	for _, v1Slice := range v1Slices {
		storeSlice := &storepb.Algorithm_RangeMask_Slice{
			Start:        v1Slice.Start,
			End:          v1Slice.End,
			Substitution: v1Slice.Substitution,
		}
		storeSlices = append(storeSlices, storeSlice)
	}
	return storeSlices
}

func convertToAlgorithmRangeMaskSlices(storeSlices []*storepb.Algorithm_RangeMask_Slice) []*v1pb.Algorithm_RangeMask_Slice {
	var v1Slices []*v1pb.Algorithm_RangeMask_Slice
	for _, storeSlice := range storeSlices {
		v1Slice := &v1pb.Algorithm_RangeMask_Slice{
			Start:        storeSlice.Start,
			End:          storeSlice.End,
			Substitution: storeSlice.Substitution,
		}
		v1Slices = append(v1Slices, v1Slice)
	}
	return v1Slices
}

func convertPasswordRestrictionSetting(v1Setting *v1pb.PasswordRestrictionSetting) *storepb.PasswordRestrictionSetting {
	if v1Setting == nil {
		return nil
	}

	return &storepb.PasswordRestrictionSetting{
		MinLength:                         v1Setting.MinLength,
		RequireNumber:                     v1Setting.RequireNumber,
		RequireLetter:                     v1Setting.RequireLetter,
		RequireUppercaseLetter:            v1Setting.RequireUppercaseLetter,
		RequireSpecialCharacter:           v1Setting.RequireSpecialCharacter,
		RequireResetPasswordForFirstLogin: v1Setting.RequireResetPasswordForFirstLogin,
		PasswordRotation:                  v1Setting.GetPasswordRotation(),
	}
}

func convertToPasswordRestrictionSetting(storeSetting *storepb.PasswordRestrictionSetting) *v1pb.PasswordRestrictionSetting {
	if storeSetting == nil {
		return nil
	}

	return &v1pb.PasswordRestrictionSetting{
		MinLength:                         storeSetting.MinLength,
		RequireNumber:                     storeSetting.RequireNumber,
		RequireLetter:                     storeSetting.RequireLetter,
		RequireUppercaseLetter:            storeSetting.RequireUppercaseLetter,
		RequireSpecialCharacter:           storeSetting.RequireSpecialCharacter,
		RequireResetPasswordForFirstLogin: storeSetting.RequireResetPasswordForFirstLogin,
		PasswordRotation:                  storeSetting.GetPasswordRotation(),
	}
}

func convertAISetting(v1Setting *v1pb.AISetting) *storepb.AISetting {
	if v1Setting == nil {
		return nil
	}

	return &storepb.AISetting{
		Enabled:  v1Setting.Enabled,
		Provider: storepb.AISetting_Provider(v1Setting.Provider),
		Endpoint: v1Setting.Endpoint,
		ApiKey:   v1Setting.ApiKey,
		Model:    v1Setting.Model,
		Version:  v1Setting.Version,
	}
}

func convertToAISetting(storeSetting *storepb.AISetting) *v1pb.AISetting {
	if storeSetting == nil {
		return nil
	}

	return &v1pb.AISetting{
		Enabled:  storeSetting.Enabled,
		Provider: v1pb.AISetting_Provider(storeSetting.Provider),
		Endpoint: storeSetting.Endpoint,
		ApiKey:   storeSetting.ApiKey,
		Model:    storeSetting.Model,
		Version:  storeSetting.Version,
	}
}

func convertToSCIMSetting(storeSetting *storepb.SCIMSetting) *v1pb.SCIMSetting {
	if storeSetting == nil {
		return nil
	}

	return &v1pb.SCIMSetting{
		Token: storeSetting.Token,
	}
}
