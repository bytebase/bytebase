package v1

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
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
	user, ok := GetUserFromContext(ctx)
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
			case "value.workspace_profile_setting_value.inactive_session_timeout":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.InactiveSessionTimeout = payload.InactiveSessionTimeout
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
			case "value.workspace_profile_setting_value.enable_audit_log_stdout":
				if payload.EnableAuditLogStdout {
					// Require TEAM or ENTERPRISE license
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.EnableAuditLogStdout = payload.EnableAuditLogStdout
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
				Source:    storepb.WorkspaceApprovalSetting_Rule_Source(rule.Source),
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
		payload, err := convertAppIMSetting(request.Msg.Setting.Value.GetAppImSettingValue())
		if err != nil {
			return nil, err
		}

		// Helper function to find or create an IM setting entry by type
		findIMSetting := func(imType storepb.ProjectWebhook_Type) *storepb.AppIMSetting_IMSetting {
			for _, s := range payload.Settings {
				if s.Type == imType {
					return s
				}
			}
			return nil
		}

		for _, path := range request.Msg.GetUpdateMask().GetPaths() {
			switch path {
			case "value.app_im_setting_value.slack":
				slackSetting := findIMSetting(storepb.ProjectWebhook_SLACK)
				if slackSetting == nil || slackSetting.GetSlack() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found slack setting"))
				}
				if err := slack.ValidateToken(ctx, slackSetting.GetSlack().GetToken()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}

			case "value.app_im_setting_value.feishu":
				feishuSetting := findIMSetting(storepb.ProjectWebhook_FEISHU)
				if feishuSetting == nil || feishuSetting.GetFeishu() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found feishu setting"))
				}
				if err := feishu.Validate(ctx, feishuSetting.GetFeishu().GetAppId(), feishuSetting.GetFeishu().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}

			case "value.app_im_setting_value.wecom":
				wecomSetting := findIMSetting(storepb.ProjectWebhook_WECOM)
				if wecomSetting == nil || wecomSetting.GetWecom() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found wecom setting"))
				}
				if err := wecom.Validate(ctx, wecomSetting.GetWecom().GetCorpId(), wecomSetting.GetWecom().GetAgentId(), wecomSetting.GetWecom().GetSecret()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}

			case "value.app_im_setting_value.lark":
				larkSetting := findIMSetting(storepb.ProjectWebhook_LARK)
				if larkSetting == nil || larkSetting.GetLark() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found lark setting"))
				}
				if err := lark.Validate(ctx, larkSetting.GetLark().GetAppId(), larkSetting.GetLark().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}

			case "value.app_im_setting_value.dingtalk":
				dingtalkSetting := findIMSetting(storepb.ProjectWebhook_DINGTALK)
				if dingtalkSetting == nil || dingtalkSetting.GetDingtalk() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found dingtalk setting"))
				}
				if err := dingtalk.Validate(ctx, dingtalkSetting.GetDingtalk().GetClientId(), dingtalkSetting.GetDingtalk().GetClientSecret(), dingtalkSetting.GetDingtalk().GetRobotCode(), user.Phone); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("validation failed, error: %v", err))
				}

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

		bytes, err := protojson.Marshal(payload)
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

	// Dynamically update audit logger runtime flag if enable_audit_log_stdout was changed
	if apiSettingName == storepb.SettingName_WORKSPACE_PROFILE {
		for _, path := range request.Msg.UpdateMask.Paths {
			if path == "value.workspace_profile_setting_value.enable_audit_log_stdout" {
				if workspaceValue := settingMessage.GetValue().GetWorkspaceProfileSettingValue(); workspaceValue != nil {
					s.profile.RuntimeEnableAuditLogStdout.Store(workspaceValue.EnableAuditLogStdout)
					if workspaceValue.EnableAuditLogStdout {
						slog.Info("audit logging to stdout enabled")
					} else {
						slog.Info("audit logging to stdout disabled")
					}
				}
				break
			}
		}
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

var domainRegexp = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)
var disallowedDomains = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"outlook.com":    true,
	"hotmail.com":    true,
	"live.com":       true,
	"msn.com":        true,
	"icloud.com":     true,
	"me.com":         true,
	"mac.com":        true,
	"yahoo.com":      true,
	"ymail.com":      true,
	"rocketmail.com": true,
	"aol.com":        true,
	"aim.com":        true,
	"protonmail.com": true,
	"pm.me":          true,
	"zoho.com":       true,
	"mail.com":       true,
	"gmx.com":        true,
	"gmx.net":        true,
	"163.com":        true,
	"126.com":        true,
	"qq.com":         true,
	"yeah.net":       true,
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
		if ok && oldTemplate.Equal(template) {
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
		if ok && oldTemplate.Equal(template) {
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

func validateApprovalTemplate(template *v1pb.ApprovalTemplate) error {
	if template.Flow == nil {
		return errors.Errorf("approval template cannot be nil")
	}
	if len(template.Flow.Roles) == 0 {
		return errors.Errorf("approval template cannot have 0 role")
	}
	return nil
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
