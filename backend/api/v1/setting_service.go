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
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	"github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	"github.com/bytebase/bytebase/backend/plugin/webhook/lark"
	"github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	"github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
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
	storepb.SettingName_SQL_RESULT_SIZE_LIMIT,
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
		settingMessage, err := convertToSettingMessage(setting)
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
	settingMessage, err := convertToSettingMessage(setting)
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
		v1pbSetting, err := convertToSettingMessage(existedSetting)
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
		payload := new(storepb.WorkspaceProfileSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetWorkspaceProfileSettingValue(), payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
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

			flow := new(storepb.ApprovalFlow)
			if err := convertProtoToProto(rule.Template.Flow, flow); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal approval flow with error: %v", err))
			}
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
		payload := new(storepb.AppIMSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetAppImSettingValue(), payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s, error: %v", apiSettingName, err))
		}
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

		payload, err := convertV1SchemaTemplateSetting(schemaTemplateSetting)
		if err != nil {
			return nil, err
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal external approval setting, error: %v", err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_DATA_CLASSIFICATION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_CLASSIFICATION); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := new(storepb.DataClassificationSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetDataClassificationSettingValue(), payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
		// it's a temporary solution to limit only 1 classification config before we support manage it in the UX.
		if len(payload.Configs) != 1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("only support define 1 classification config for now"))
		}
		if len(payload.Configs[0].Classification) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing classification map"))
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_SEMANTIC_TYPES:
		storeSemanticTypeSetting := new(storepb.SemanticTypeSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetSemanticTypeSettingValue(), storeSemanticTypeSetting); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
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
	case storepb.SettingName_SQL_RESULT_SIZE_LIMIT:
		maximumSQLResultSizeSetting := new(storepb.MaximumSQLResultSizeSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetMaximumSqlResultSizeSetting(), maximumSQLResultSizeSetting); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
		if maximumSQLResultSizeSetting.Limit <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid maximum sql result size"))
		}
		bytes, err := protojson.Marshal(maximumSQLResultSizeSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
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
		passwordSetting := new(storepb.PasswordRestrictionSetting)
		if err := convertProtoToProto(request.Msg.Setting.Value.GetPasswordRestrictionSetting(), passwordSetting); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
		if passwordSetting.MinLength < 8 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid password minimum length, should no less than 8"))
		}
		bytes, err := protojson.Marshal(passwordSetting)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting for %s with error: %v", apiSettingName, err))
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_AI:
		aiSetting := &storepb.AISetting{}
		if err := convertProtoToProto(request.Msg.Setting.Value.GetAiSetting(), aiSetting); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", apiSettingName, err))
		}
		if aiSetting.Enabled {
			if aiSetting.Endpoint == "" || aiSetting.Model == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("API endpoint and model are required"))
			}
			if existedSetting != nil {
				existedAISetting, err := convertToSettingMessage(existedSetting)
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
		if serr := validateEnvironments(request.Msg.Setting.Value.GetEnvironmentSetting().GetEnvironments()); serr != nil {
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
				// check if instances are using the environments
				count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{EnvironmentID: &env.Id})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				if count > 0 {
					return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("all instances in the environment %v should be deleted first", env.Id))
				}
				uses, err := s.store.CheckDatabaseUseEnvironment(ctx, env.Id)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				if uses {
					return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("all databases in the environment %v should be deleted first", env.Id))
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

	settingMessage, err := convertToSettingMessage(setting)
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

func convertProtoToProto(inputPB, outputPB protoreflect.ProtoMessage) error {
	bytes, err := protojson.Marshal(inputPB)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to marshal setting: %v", err))
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(bytes, outputPB); err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting: %v", err))
	}
	return nil
}

func convertToSettingMessage(setting *store.SettingMessage) (*v1pb.Setting, error) {
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
	case storepb.SettingName_WORKSPACE_PROFILE:
		v1Value := new(v1pb.WorkspaceProfileSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
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
		value := new(storepb.SchemaTemplateSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
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
	case storepb.SettingName_DATA_CLASSIFICATION:
		v1Value := new(v1pb.DataClassificationSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_DataClassificationSettingValue{
					DataClassificationSettingValue: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_SEMANTIC_TYPES:
		v1Value := new(v1pb.SemanticTypeSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SemanticTypeSettingValue{
					SemanticTypeSettingValue: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_SQL_RESULT_SIZE_LIMIT:
		v1Value := new(v1pb.MaximumSQLResultSizeSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
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
	case storepb.SettingName_SCIM:
		v1Value := new(v1pb.SCIMSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_ScimSetting{
					ScimSetting: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_PASSWORD_RESTRICTION:
		v1Value := new(v1pb.PasswordRestrictionSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_PasswordRestrictionSetting{
					PasswordRestrictionSetting: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_AI:
		v1Value := &v1pb.AISetting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to unmarshal setting value for %s with error: %v", setting.Name, err))
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
	case storepb.SettingName_ENVIRONMENT:
		v1Value, err := convertToEnvironmentSetting(setting.Value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting value for %s with error: %v", setting.Name, err))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_EnvironmentSetting{
					EnvironmentSetting: v1Value,
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
	case storepb.SettingName_SQL_RESULT_SIZE_LIMIT:
		return v1pb.Setting_SQL_RESULT_SIZE_LIMIT
	case storepb.SettingName_SCIM:
		return v1pb.Setting_SCIM
	case storepb.SettingName_PASSWORD_RESTRICTION:
		return v1pb.Setting_PASSWORD_RESTRICTION
	case storepb.SettingName_ENVIRONMENT:
		return v1pb.Setting_ENVIRONMENT
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
	case v1pb.Setting_SQL_RESULT_SIZE_LIMIT:
		return storepb.SettingName_SQL_RESULT_SIZE_LIMIT
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

func validateEnvironments(envs []*v1pb.EnvironmentSetting_Environment) error {
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
