package v1

import (
	"context"
	"log/slog"
	"regexp"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto" // Added
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"

	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	"github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	"github.com/bytebase/bytebase/backend/plugin/webhook/lark"
	"github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	"github.com/bytebase/bytebase/backend/plugin/webhook/teams"
	"github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
	"github.com/bytebase/bytebase/backend/store"
)

// SettingService implements the setting service.
type SettingService struct {
	v1connect.UnimplementedSettingServiceHandler
	store          *store.Store
	profile        *config.Profile
	licenseService *enterprise.LicenseService
}

// NewSettingService creates a new setting service.
func NewSettingService(
	store *store.Store,
	profile *config.Profile,
	licenseService *enterprise.LicenseService,
) *SettingService {
	return &SettingService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
	}
}

// ListSettings lists all settings.
func (s *SettingService) ListSettings(ctx context.Context, _ *connect.Request[v1pb.ListSettingsRequest]) (*connect.Response[v1pb.ListSettingsResponse], error) {
	settings, err := s.store.ListSettings(ctx, &store.FindSettingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list settings: %v", err))
	}

	response := &v1pb.ListSettingsResponse{}
	for _, setting := range settings {
		if isSettingDisallowed(setting.Name) {
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
	if isSettingDisallowed(apiSettingName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting is not available"))
	}

	setting, err := s.store.GetSetting(ctx, apiSettingName)
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
	apiSettingName, err := convertStringToSettingName(settingName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid setting name: %v", err))
	}
	if isSettingDisallowed(apiSettingName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("setting is not available"))
	}
	if s.profile.IsFeatureUnavailable(settingName) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
	}
	existedSetting, err := s.store.GetSetting(ctx, apiSettingName)
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

	var storeSettingValue proto.Message
	var resetAuditLogStdout bool
	var resetClassification bool

	switch apiSettingName {
	case storepb.SettingName_WORKSPACE_PROFILE:
		if request.Msg.UpdateMask == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask is required"))
		}
		payload := convertWorkspaceProfileSetting(request.Msg.Setting.Value.GetWorkspaceProfile())
		oldSetting, err := s.store.GetWorkspaceProfileSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to find setting %s with error: %v", apiSettingName, err))
		}

		for _, path := range request.Msg.UpdateMask.Paths {
			switch path {
			case "value.workspace_profile.disallow_signup":
				if s.profile.SaaS {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
				}
				if payload.DisallowSignup {
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.DisallowSignup = payload.DisallowSignup
			case "value.workspace_profile.external_url":
				if s.profile.SaaS {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("feature %s is unavailable in current mode", settingName))
				}
				// Prevent changing external URL via UI when it's set via command-line flag
				if s.profile.ExternalURL != "" {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("external URL is managed via --external-url command-line flag and cannot be changed through the UI"))
				}
				if payload.ExternalUrl != "" {
					externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
					if err != nil {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid external url: %v", err))
					}
					payload.ExternalUrl = externalURL
				}
				oldSetting.ExternalUrl = payload.ExternalUrl
			case "value.workspace_profile.require_2fa":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TWO_FA); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.Require_2Fa = payload.Require_2Fa
			case "value.workspace_profile.access_token_duration":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				if payload.AccessTokenDuration != nil && payload.AccessTokenDuration.Seconds > 0 && payload.AccessTokenDuration.AsDuration() < time.Minute {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("access token duration should be at least one minute"))
				}
				oldSetting.AccessTokenDuration = payload.AccessTokenDuration
			case "value.workspace_profile.refresh_token_duration":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				if payload.RefreshTokenDuration != nil && payload.RefreshTokenDuration.Seconds > 0 && payload.RefreshTokenDuration.AsDuration() < time.Hour {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("refresh token duration should be at least one hour"))
				}
				oldSetting.RefreshTokenDuration = payload.RefreshTokenDuration
			case "value.workspace_profile.inactive_session_timeout":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.InactiveSessionTimeout = payload.InactiveSessionTimeout
			case "value.workspace_profile.announcement":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DASHBOARD_ANNOUNCEMENT); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.Announcement = payload.Announcement
			case "value.workspace_profile.maximum_role_expiration":
				if payload.MaximumRoleExpiration != nil {
					// If the value is less than or equal to 0, we will remove the setting. AKA no limit.
					if payload.MaximumRoleExpiration.Seconds <= 0 {
						payload.MaximumRoleExpiration = nil
					}
				}
				oldSetting.MaximumRoleExpiration = payload.MaximumRoleExpiration
			case "value.workspace_profile.domains":
				if err := validateDomains(payload.Domains); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid domains, error %v", err))
				}
				oldSetting.Domains = payload.Domains
			case "value.workspace_profile.enforce_identity_domain":
				if payload.EnforceIdentityDomain {
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_USER_EMAIL_DOMAIN_RESTRICTION); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.EnforceIdentityDomain = payload.EnforceIdentityDomain
			case "value.workspace_profile.database_change_mode":
				oldSetting.DatabaseChangeMode = payload.DatabaseChangeMode
			case "value.workspace_profile.disallow_password_signin":
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
			case "value.workspace_profile.enable_metric_collection":
				oldSetting.EnableMetricCollection = payload.EnableMetricCollection
			case "value.workspace_profile.enable_audit_log_stdout":
				if payload.EnableAuditLogStdout {
					// Require TEAM or ENTERPRISE license
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				resetAuditLogStdout = true
				oldSetting.EnableAuditLogStdout = payload.EnableAuditLogStdout
			case "value.workspace_profile.watermark":
				if payload.Watermark {
					if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_WATERMARK); err != nil {
						return nil, connect.NewError(connect.CodePermissionDenied, err)
					}
				}
				oldSetting.Watermark = payload.Watermark
			case "value.workspace_profile.directory_sync_token":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DIRECTORY_SYNC); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				// Generate a new token if the payload is empty.
				// This handles both initial setup and token reset (when user explicitly sends empty string).
				if payload.DirectorySyncToken == "" {
					payload.DirectorySyncToken = uuid.New().String()
				}
				oldSetting.DirectorySyncToken = payload.DirectorySyncToken
			case "value.workspace_profile.branding_logo":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_CUSTOM_LOGO); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				oldSetting.BrandingLogo = payload.BrandingLogo
			case "value.workspace_profile.password_restriction":
				if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_PASSWORD_RESTRICTIONS); err != nil {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				if payload.PasswordRestriction != nil && payload.PasswordRestriction.MinLength < 8 {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid password minimum length, should no less than 8"))
				}
				oldSetting.PasswordRestriction = payload.PasswordRestriction
			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %v", path))
			}
		}

		if len(oldSetting.Domains) == 0 && oldSetting.EnforceIdentityDomain {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("identity domain can be enforced only when workspace domains are set"))
		}
		storeSettingValue = oldSetting
	case storepb.SettingName_WORKSPACE_APPROVAL:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_APPROVAL_WORKFLOW); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		payload := &storepb.WorkspaceApprovalSetting{}
		for _, rule := range request.Msg.Setting.Value.GetWorkspaceApproval().Rules {
			// Validate the condition.
			if _, err := common.ConvertUnparsedApproval(rule.Condition); err != nil {
				return nil, err
			}

			// For SOURCE_UNSPECIFIED (fallback) rules, validate that only project_id is used
			if rule.Source == v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED {
				conditionExpr := ""
				if rule.Condition != nil {
					conditionExpr = rule.Condition.Expression
				}
				if err := common.ValidateFallbackApprovalExpr(conditionExpr); err != nil {
					return nil, err
				}
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
		storeSettingValue = payload
	case storepb.SettingName_APP_IM:
		payload, err := convertAppIMSetting(request.Msg.Setting.Value.GetAppIm())
		if err != nil {
			return nil, err
		}

		// Helper function to find or create an IM setting entry by type
		findIMSetting := func(imType storepb.WebhookType) *storepb.AppIMSetting_IMSetting {
			for _, s := range payload.Settings {
				if s.Type == imType {
					return s
				}
			}
			return nil
		}

		for _, path := range request.Msg.GetUpdateMask().GetPaths() {
			switch path {
			case "value.app_im.slack":
				slackSetting := findIMSetting(storepb.WebhookType_SLACK)
				if slackSetting == nil || slackSetting.GetSlack() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found slack setting"))
				}
				if err := slack.ValidateToken(ctx, slackSetting.GetSlack().GetToken()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			case "value.app_im.feishu":
				feishuSetting := findIMSetting(storepb.WebhookType_FEISHU)
				if feishuSetting == nil || feishuSetting.GetFeishu() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found feishu setting"))
				}
				if err := feishu.Validate(ctx, feishuSetting.GetFeishu().GetAppId(), feishuSetting.GetFeishu().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			case "value.app_im.wecom":
				wecomSetting := findIMSetting(storepb.WebhookType_WECOM)
				if wecomSetting == nil || wecomSetting.GetWecom() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found wecom setting"))
				}
				if err := wecom.Validate(ctx, wecomSetting.GetWecom().GetCorpId(), wecomSetting.GetWecom().GetAgentId(), wecomSetting.GetWecom().GetSecret()); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			case "value.app_im.lark":
				larkSetting := findIMSetting(storepb.WebhookType_LARK)
				if larkSetting == nil || larkSetting.GetLark() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found lark setting"))
				}
				if err := lark.Validate(ctx, larkSetting.GetLark().GetAppId(), larkSetting.GetLark().GetAppSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			case "value.app_im.dingtalk":
				dingtalkSetting := findIMSetting(storepb.WebhookType_DINGTALK)
				if dingtalkSetting == nil || dingtalkSetting.GetDingtalk() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found dingtalk setting"))
				}
				if err := dingtalk.Validate(ctx, dingtalkSetting.GetDingtalk().GetClientId(), dingtalkSetting.GetDingtalk().GetClientSecret(), dingtalkSetting.GetDingtalk().GetRobotCode(), user.Phone); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			case "value.app_im_setting_value.teams":
				teamsSetting := findIMSetting(storepb.WebhookType_TEAMS)
				if teamsSetting == nil || teamsSetting.GetTeams() == nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot found teams setting"))
				}
				if err := teams.Validate(ctx, teamsSetting.GetTeams().GetTenantId(), teamsSetting.GetTeams().GetClientId(), teamsSetting.GetTeams().GetClientSecret(), user.Email); err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "validation failed"))
				}

			default:
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %v", path))
			}
		}

		storeSettingValue = payload

	case storepb.SettingName_DATA_CLASSIFICATION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_CLASSIFICATION); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertDataClassificationSetting(request.Msg.Setting.Value.GetDataClassification())
		// it's a temporary solution to limit only 1 classification config before we support manage it in the UX.
		if len(payload.Configs) > 1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("only support define 1 classification config for now"))
		}
		if len(payload.Configs) == 1 && len(payload.Configs[0].Classification) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("missing classification map"))
		}
		resetClassification = true
		storeSettingValue = payload
	case storepb.SettingName_SEMANTIC_TYPES:
		storeSemanticTypeSetting := convertSemanticTypeSetting(request.Msg.Setting.Value.GetSemanticType())
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
		storeSettingValue = storeSemanticTypeSetting
	case storepb.SettingName_AI:
		aiSetting := convertAISetting(request.Msg.Setting.Value.GetAi())
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
					aiSetting.ApiKey = existedAISetting.Value.GetAi().GetApiKey()
				}
			}
			if aiSetting.ApiKey == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("API key is required"))
			}
		}

		storeSettingValue = aiSetting
	case storepb.SettingName_ENVIRONMENT:
		if serr := s.validateEnvironments(request.Msg.Setting.Value.GetEnvironment().GetEnvironments()); serr != nil {
			return nil, serr
		}

		environmentSetting := convertEnvironmentSetting(request.Msg.Setting.Value.GetEnvironment())
		oldEnvironmentSetting, err := s.store.GetEnvironment(ctx)
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
				if _, err := s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
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

		storeSettingValue = environmentSetting
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported setting %v", apiSettingName))
	}

	if request.Msg.ValidateOnly {
		return connect.NewResponse(&v1pb.Setting{
			Name:  request.Msg.Setting.Name,
			Value: request.Msg.Setting.Value,
		}), nil
	}

	setting, err := s.store.UpsertSetting(ctx, &store.SettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to set setting: %v", err))
	}

	// Dynamically update audit logger runtime flag if enable_audit_log_stdout was changed
	if resetAuditLogStdout {
		workspaceProfile, err := s.store.GetWorkspaceProfileSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get workspace setting message: %v", err))
		}
		s.profile.RuntimeEnableAuditLogStdout.Store(workspaceProfile.EnableAuditLogStdout)
	}

	// It's a temporary solution to map the classification to all projects before we support it in the UX.
	if resetClassification {
		classification, err := s.store.GetDataClassificationSetting(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get classification setting message: %v", err))
		}
		var classificationID string
		if len(classification.Configs) > 0 {
			classificationID = classification.Configs[0].Id
		}

		projects, err := s.store.ListProjects(ctx, &store.FindProjectMessage{ShowDeleted: false})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list projects with error: %v", err))
		}

		batchUpdate := []*store.UpdateProjectMessage{}
		for _, project := range projects {
			batchUpdate = append(batchUpdate, &store.UpdateProjectMessage{
				ResourceID:                 project.ResourceID,
				DataClassificationConfigID: &classificationID,
			})
		}
		if _, err = s.store.UpdateProjects(ctx, batchUpdate...); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to patch project classification with error: %v", err))
		}
	}

	settingMessage, err := convertToSettingMessage(setting, s.profile)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to convert setting message: %v", err))
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

func isSettingDisallowed(name storepb.SettingName) bool {
	// Backend-only settings that should never be exposed via the API.
	// SYSTEM: Internal system settings (auth secret, workspace ID, enterprise license)
	return name == storepb.SettingName_SYSTEM
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
