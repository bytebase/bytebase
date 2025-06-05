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

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
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
var whitelistSettings = []storepb.SettingName{
	storepb.SettingName_BRANDING_LOGO,
	storepb.SettingName_WORKSPACE_ID,
	storepb.SettingName_APP_IM,
	storepb.SettingName_WATERMARK,
	storepb.SettingName_AI,
	storepb.SettingName_WORKSPACE_APPROVAL,
	storepb.SettingName_WORKSPACE_MAIL_DELIVERY,
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
		settingMessage, err := convertToSettingMessage(setting)
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
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting name: %v", err)
	}
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
	settingMessage, err := convertToSettingMessage(setting)
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
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting name: %v", err)
	}
	existedSetting, err := s.store.GetSettingV2(ctx, apiSettingName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find setting %s with error: %v", settingName, err)
	}
	if existedSetting == nil && !request.AllowMissing {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
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
				if err := s.licenseService.IsFeatureEnabled(base.FeatureDisallowSignup); err != nil {
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
				if err := s.licenseService.IsFeatureEnabled(base.Feature2FA); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				oldSetting.Require_2Fa = payload.Require_2Fa
			case "value.workspace_profile_setting_value.outbound_ip_list":
				// We're not support update outbound_ip_list via api.
			case "value.workspace_profile_setting_value.token_duration":
				if err := s.licenseService.IsFeatureEnabled(base.FeatureSecureToken); err != nil {
					return nil, status.Error(codes.PermissionDenied, err.Error())
				}
				if payload.TokenDuration != nil && payload.TokenDuration.Seconds > 0 && payload.TokenDuration.AsDuration() < time.Hour {
					return nil, status.Errorf(codes.InvalidArgument, "refresh token duration should be at least one hour")
				}
				oldSetting.TokenDuration = payload.TokenDuration
			case "value.workspace_profile_setting_value.announcement":
				if err := s.licenseService.IsFeatureEnabled(base.FeatureAnnouncement); err != nil {
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
					if err := s.licenseService.IsFeatureEnabled(base.FeatureDomainRestriction); err != nil {
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
	case storepb.SettingName_WORKSPACE_APPROVAL:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureCustomApproval); err != nil {
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
				},
			})
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_WORKSPACE_MAIL_DELIVERY:
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
	case storepb.SettingName_BRANDING_LOGO:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureBranding); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case storepb.SettingName_PLUGIN_AGENT:
		payload := new(storepb.AgentPluginSetting)
		if err := convertProtoToProto(request.Setting.Value.GetAgentPluginSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}

		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)

	case storepb.SettingName_APP_IM:
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

	case storepb.SettingName_SCHEMA_TEMPLATE:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureSchemaTemplate); err != nil {
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
	case storepb.SettingName_DATA_CLASSIFICATION:
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
	case storepb.SettingName_SEMANTIC_TYPES:
		storeSemanticTypeSetting := new(storepb.SemanticTypeSetting)
		if err := convertProtoToProto(request.Setting.Value.GetSemanticTypeSettingValue(), storeSemanticTypeSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		idMap := make(map[string]bool)
		for _, tp := range storeSemanticTypeSetting.Types {
			if tp.Title == "" {
				return nil, status.Errorf(codes.InvalidArgument, "category title cannot be empty: %s", tp.Id)
			}
			if idMap[tp.Id] {
				return nil, status.Errorf(codes.InvalidArgument, "duplicate semantic type id: %s", tp.Id)
			}
			m, ok := tp.GetAlgorithm().GetMask().(*storepb.Algorithm_InnerOuterMask_)
			if ok && m.InnerOuterMask != nil {
				if m.InnerOuterMask.Type == storepb.Algorithm_InnerOuterMask_MASK_TYPE_UNSPECIFIED {
					return nil, status.Errorf(codes.InvalidArgument, "inner outer mask type has to be specified")
				}
			}
			idMap[tp.Id] = true
		}
		bytes, err := protojson.Marshal(storeSemanticTypeSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case storepb.SettingName_WATERMARK:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureWatermark); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case storepb.SettingName_SQL_RESULT_SIZE_LIMIT:
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
	case storepb.SettingName_SCIM:
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
	case storepb.SettingName_PASSWORD_RESTRICTION:
		if err := s.licenseService.IsFeatureEnabled(base.FeaturePasswordRestriction); err != nil {
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
	case storepb.SettingName_AI:
		aiSetting := &storepb.AISetting{}
		if err := convertProtoToProto(request.Setting.Value.GetAiSetting(), aiSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		if aiSetting.Enabled {
			if aiSetting.Endpoint == "" || aiSetting.Model == "" {
				return nil, status.Errorf(codes.InvalidArgument, "API endpoint and model are required")
			}
			if existedSetting != nil {
				existedAISetting, err := convertToSettingMessage(existedSetting)
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
	case storepb.SettingName_ENVIRONMENT:
		if serr := validateEnvironments(request.Setting.Value.GetEnvironmentSetting().GetEnvironments()); serr != nil {
			return nil, serr.Err()
		}

		environmentSetting := convertEnvironmentSetting(request.Setting.Value.GetEnvironmentSetting())
		oldEnvironmentSetting, err := s.store.GetEnvironmentSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get old environment setting with error: %v", err)
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
					return nil, status.Error(codes.Internal, err.Error())
				}
				if count > 0 {
					return nil, status.Errorf(codes.FailedPrecondition, "all instances in the environment %v should be deleted first", env.Id)
				}
				uses, err := s.store.CheckDatabaseUseEnvironment(ctx, env.Id)
				if err != nil {
					return nil, status.Error(codes.Internal, err.Error())
				}
				if uses {
					return nil, status.Errorf(codes.FailedPrecondition, "all databases in the environment %v should be deleted first", env.Id)
				}
			}
		}

		bytes, err := protojson.Marshal(environmentSetting)
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

	settingMessage, err := convertToSettingMessage(setting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
	}

	// it's a temporary solution to map the classification to all projects before we support it in the UX.
	if apiSettingName == storepb.SettingName_DATA_CLASSIFICATION && len(settingMessage.Value.GetDataClassificationSettingValue().Configs) == 1 {
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

func convertToSettingMessage(setting *store.SettingMessage) (*v1pb.Setting, error) {
	settingName := fmt.Sprintf("%s%s", common.SettingNamePrefix, convertStoreSettingNameToV1(setting.Name).String())
	switch setting.Name {
	case storepb.SettingName_WORKSPACE_MAIL_DELIVERY:
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
	case storepb.SettingName_APP_IM:
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
	case storepb.SettingName_PLUGIN_AGENT:
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
	case storepb.SettingName_WORKSPACE_PROFILE:
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
	case storepb.SettingName_WORKSPACE_APPROVAL:
		storeValue := new(storepb.WorkspaceApprovalSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
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
	case storepb.SettingName_DATA_CLASSIFICATION:
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
	case storepb.SettingName_SEMANTIC_TYPES:
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
	case storepb.SettingName_SQL_RESULT_SIZE_LIMIT:
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
	case storepb.SettingName_SCIM:
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
	case storepb.SettingName_PASSWORD_RESTRICTION:
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
	case storepb.SettingName_AI:
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
	case storepb.SettingName_ENVIRONMENT:
		v1Value, err := convertToEnvironmentSetting(setting.Value)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert setting value for %s with error: %v", setting.Name, err)
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
		return status.Errorf(codes.Internal, "failed to get setting %q: %v", storepb.SettingName_SCHEMA_TEMPLATE, err)
	}
	settingValue := "{}"
	if oldStoreSetting != nil {
		settingValue = oldStoreSetting.Value
	}

	value := new(storepb.SchemaTemplateSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(settingValue), value); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal setting value for %v with error: %v", storepb.SettingName_SCHEMA_TEMPLATE, err)
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
	case storepb.SettingName_PLUGIN_AGENT:
		return v1pb.Setting_PLUGIN_AGENT
	case storepb.SettingName_WORKSPACE_MAIL_DELIVERY:
		return v1pb.Setting_WORKSPACE_MAIL_DELIVERY
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
	case v1pb.Setting_PLUGIN_AGENT:
		return storepb.SettingName_PLUGIN_AGENT
	case v1pb.Setting_WORKSPACE_MAIL_DELIVERY:
		return storepb.SettingName_WORKSPACE_MAIL_DELIVERY
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
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, err
	}
	switch apiSettingName {
	case storepb.SettingName_WORKSPACE_MAIL_DELIVERY:
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

func validateEnvironments(envs []*v1pb.EnvironmentSetting_Environment) *status.Status {
	used := map[string]bool{}
	for _, env := range envs {
		if env.Title == "" {
			return status.Newf(codes.InvalidArgument, "environment title cannot be empty")
		}
		if !isValidResourceID(env.Id) {
			return status.Newf(codes.InvalidArgument, "invalid environment ID %v", env.Id)
		}
		if used[env.Id] {
			return status.Newf(codes.InvalidArgument, "duplicate environment ID %v", env.Id)
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
