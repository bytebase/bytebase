package v1

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
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

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
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
	api.SettingPluginOpenAIKey,
	api.SettingPluginOpenAIEndpoint,
	api.SettingWorkspaceApproval,
	api.SettingWorkspaceMailDelivery,
	api.SettingWorkspaceProfile,
	api.SettingWorkspaceExternalApproval,
	api.SettingSchemaTemplate,
	api.SettingDataClassification,
	api.SettingSemanticTypes,
	api.SettingMaskingAlgorithm,
}

var preservedMaskingAlgorithmIDMatcher = regexp.MustCompile("^[0]{8}-[0]{4}-[0]{4}-[0]{4}-[0]{9}[0-9a-fA-F]{3}$")

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

	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &apiSettingName,
	})
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

	existedSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &apiSettingName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find setting %s with error: %v", settingName, err)
	}
	if existedSetting == nil && !request.AllowMissing {
		return nil, status.Errorf(codes.NotFound, "setting %s not found", settingName)
	}

	var storeSettingValue string
	switch apiSettingName {
	case api.SettingWorkspaceProfile:
		if request.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
		}
		payload := new(storepb.WorkspaceProfileSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetWorkspaceProfileSettingValue(), payload); err != nil {
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
					return nil, status.Errorf(codes.PermissionDenied, err.Error())
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
					return nil, status.Errorf(codes.PermissionDenied, err.Error())
				}
				oldSetting.Require_2Fa = payload.Require_2Fa
			case "value.workspace_profile_setting_value.outbound_ip_list":
				// We're not support update outbound_ip_list via api.
			case "value.workspace_profile_setting_value.gitops_webhook_url":
				if payload.GitopsWebhookUrl != "" {
					gitopsWebhookURL, err := common.NormalizeExternalURL(payload.GitopsWebhookUrl)
					if err != nil {
						return nil, status.Errorf(codes.InvalidArgument, "invalid GitOps webhook URL: %v", err)
					}
					payload.GitopsWebhookUrl = gitopsWebhookURL
				}
				oldSetting.GitopsWebhookUrl = payload.GitopsWebhookUrl
			case "value.workspace_profile_setting_value.token_duration":
				if err := s.licenseService.IsFeatureEnabled(api.FeatureSecureToken); err != nil {
					return nil, status.Errorf(codes.PermissionDenied, err.Error())
				}
				if payload.TokenDuration != nil && payload.TokenDuration.Seconds > 0 && payload.TokenDuration.AsDuration() < time.Hour {
					return nil, status.Errorf(codes.InvalidArgument, "refresh token duration should be at least one hour")
				}
				oldSetting.TokenDuration = payload.TokenDuration
			case "value.workspace_profile_setting_value.announcement":
				if err := s.licenseService.IsFeatureEnabled(api.FeatureAnnouncement); err != nil {
					return nil, status.Errorf(codes.PermissionDenied, err.Error())
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
				oldSetting.EnforceIdentityDomain = payload.EnforceIdentityDomain
			case "value.workspace_profile_setting_value.database_change_mode":
				oldSetting.DatabaseChangeMode = payload.DatabaseChangeMode
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
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
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
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("failed to get creator: %v", err))
			}
			creator, err := s.store.GetUserByEmail(ctx, email)
			if err != nil {
				return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
			}
			if creator == nil {
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("creator %s not found", rule.Template.Creator))
			}
			creatorID = creator.ID

			flow := new(storepb.ApprovalFlow)
			if err := convertV1PbToStorePb(rule.Template.Flow, flow); err != nil {
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
			oldStoreSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
				Name: &apiSettingName,
			})
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
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case api.SettingPluginAgent:
		payload := new(storepb.AgentPluginSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetAgentPluginSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}

		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)

	case api.SettingAppIM:
		payload := new(storepb.AppIMSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetAppImSettingValue(), payload); err != nil {
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

	case api.SettingWorkspaceExternalApproval:
		oldSetting, err := s.store.GetWorkspaceExternalApprovalSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get workspace external approval setting: %v", err)
		}

		externalApprovalSetting := request.Setting.Value.GetExternalApprovalSettingValue()
		if externalApprovalSetting == nil {
			return nil, status.Errorf(codes.InvalidArgument, "value cannot be nil when setting external approval setting")
		}
		storeValue := convertExternalApprovalSetting(externalApprovalSetting)

		newNode := make(map[string]*storepb.ExternalApprovalSetting_Node)
		for _, node := range storeValue.Nodes {
			newNode[node.Id] = node
		}
		removed := make(map[string]bool)
		for _, node := range oldSetting.Nodes {
			if _, ok := newNode[node.Id]; !ok {
				removed[node.Id] = true
			}
		}
		if len(removed) > 0 {
			externalApprovalType := api.ExternalApprovalTypeRelay
			approvals, err := s.store.ListExternalApprovalV2(
				ctx,
				&store.ListExternalApprovalMessage{
					Type: &externalApprovalType,
				},
			)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to list external approvals: %v", err)
			}
			for _, approval := range approvals {
				payload := &api.ExternalApprovalPayloadRelay{}
				if err := json.Unmarshal([]byte(approval.Payload), payload); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to unmarshal external approval payload: %v", err)
				}
				if removed[payload.ExternalApprovalNodeID] {
					return nil, status.Errorf(codes.InvalidArgument, "cannot remove %s because it is used by the external approval node in issue %d", payload.ExternalApprovalNodeID, approval.IssueUID)
				}
			}
		}

		bytes, err := protojson.Marshal(storeValue)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal external approval setting, error: %v", err)
		}
		storeSettingValue = string(bytes)
	case api.SettingSchemaTemplate:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSchemaTemplate); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		schemaTemplateSetting := request.Setting.Value.GetSchemaTemplateSettingValue()
		if schemaTemplateSetting == nil {
			return nil, status.Errorf(codes.InvalidArgument, "value cannot be nil when setting schema template setting")
		}

		if err := s.validateSchemaTemplate(ctx, schemaTemplateSetting); err != nil {
			return nil, err
		}

		payload, err := convertV1SchemaTemplateSetting(ctx, schemaTemplateSetting)
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
		if err := convertV1PbToStorePb(request.Setting.Value.GetDataClassificationSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		// it's a temporary solution to limit only 1 classification config before we support manage it in the UX.
		if len(payload.Configs) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "only support define 1 classification config for now")
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingSemanticTypes:
		storeSemanticTypeSetting := new(storepb.SemanticTypeSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetSemanticTypeSettingValue(), storeSemanticTypeSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		idMap := make(map[string]struct{})
		for _, tp := range storeSemanticTypeSetting.Types {
			if !isValidUUID(tp.Id) {
				return nil, status.Errorf(codes.InvalidArgument, "invalid semantic type id format: %s", tp.Id)
			}
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
	case api.SettingMaskingAlgorithm:
		idMap := make(map[string]struct{})
		for _, algorithm := range request.Setting.Value.GetMaskingAlgorithmSettingValue().Algorithms {
			if err := validateMaskingAlgorithm(algorithm); err != nil {
				return nil, err
			}
			if _, ok := idMap[algorithm.Id]; ok {
				return nil, status.Errorf(codes.InvalidArgument, "duplicate masking algorithm id: %s", algorithm.Id)
			}
			idMap[algorithm.Id] = struct{}{}
		}
		storeMaskingAlgorithmSetting := new(storepb.MaskingAlgorithmSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetMaskingAlgorithmSettingValue(), storeMaskingAlgorithmSetting); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		bytes, err := protojson.Marshal(storeMaskingAlgorithmSetting)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal setting for %s with error: %v", apiSettingName, err)
		}
		storeSettingValue = string(bytes)
	case api.SettingWatermark:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureWatermark); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	case api.SettingPluginOpenAIKey:
		if err := s.licenseService.IsFeatureEnabled(api.FeaturePluginOpenAI); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		storeSettingValue = request.Setting.Value.GetStringValue()
	default:
		storeSettingValue = request.Setting.Value.GetStringValue()
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	}, principalID)
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
				UpdaterID:                  principalID,
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

func convertV1PbToStorePb(inputPB, outputPB protoreflect.ProtoMessage) error {
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
				return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
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
	case api.SettingWorkspaceExternalApproval:
		storeValue := new(storepb.ExternalApprovalSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting values for %s with error: %v", setting.Name, err)
		}
		v1Value := convertToExternalApprovalSetting(storeValue)
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_ExternalApprovalSettingValue{
					ExternalApprovalSettingValue: v1Value,
				},
			},
		}, nil
	case api.SettingSchemaTemplate:
		value := new(storepb.SchemaTemplateSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}

		sts, err := convertSchemaTemplateSetting(ctx, value)
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
	case api.SettingMaskingAlgorithm:
		v1Value := new(v1pb.MaskingAlgorithmSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), v1Value); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_MaskingAlgorithmSettingValue{
					MaskingAlgorithmSettingValue: v1Value,
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
	settingName := api.SettingSchemaTemplate
	oldStoreSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get setting %q: %v", settingName, err)
	}
	settingValue := "{}"
	if oldStoreSetting != nil {
		settingValue = oldStoreSetting.Value
	}

	value := new(storepb.SchemaTemplateSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(settingValue), value); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", settingName, err)
	}
	v1Value, err := convertSchemaTemplateSetting(ctx, value)
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
		if err := validateTableMetadata(ctx, template.Engine, tableMetadata); err != nil {
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
		if err := validateTableMetadata(ctx, template.Engine, template.Table); err != nil {
			return err
		}
	}

	return nil
}

func validateTableMetadata(ctx context.Context, engine v1pb.Engine, tableMetadata *v1pb.TableMetadata) error {
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
	tempStoreSchemaMetadata, _, err := convertV1DatabaseMetadata(ctx, tempMetadata, nil /* optionalStores */)
	if err != nil {
		return err
	}
	if err := checkDatabaseMetadata(storepb.Engine(engine), tempStoreSchemaMetadata); err != nil {
		return errors.Wrap(err, "failed to check database metadata")
	}
	defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(engine), tempStoreSchemaMetadata)
	if _, err := schema.GetDesignSchema(storepb.Engine(engine), defaultSchema, "" /* baseline */, tempStoreSchemaMetadata); err != nil {
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
	workspaceProfileSettingName := api.SettingWorkspaceProfile
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &workspaceProfileSettingName})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get workspace profile setting: %v", err)
	}
	if setting != nil {
		settingValue := new(storepb.WorkspaceProfileSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), settingValue); err != nil {
			return status.Errorf(codes.Internal, "failed to unmarshal setting value: %v", err)
		}
		if settingValue.ExternalUrl != "" {
			consoleRedirectURL = settingValue.ExternalUrl
		}
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

func convertToExternalApprovalSetting(s *storepb.ExternalApprovalSetting) *v1pb.ExternalApprovalSetting {
	return &v1pb.ExternalApprovalSetting{
		Nodes: convertToExternalApprovalSettingNodes(s.Nodes),
	}
}

func convertToExternalApprovalSettingNodes(nodes []*storepb.ExternalApprovalSetting_Node) []*v1pb.ExternalApprovalSetting_Node {
	v1Nodes := make([]*v1pb.ExternalApprovalSetting_Node, len(nodes))
	for i := range nodes {
		v1Nodes[i] = convertToExternalApprovalSettingNode(nodes[i])
	}
	return v1Nodes
}

func convertToExternalApprovalSettingNode(o *storepb.ExternalApprovalSetting_Node) *v1pb.ExternalApprovalSetting_Node {
	return &v1pb.ExternalApprovalSetting_Node{
		Id:       o.Id,
		Title:    o.Title,
		Endpoint: o.Endpoint,
	}
}

func convertExternalApprovalSetting(s *v1pb.ExternalApprovalSetting) *storepb.ExternalApprovalSetting {
	return &storepb.ExternalApprovalSetting{
		Nodes: convertExternalApprovalSettingNodes(s.Nodes),
	}
}

func convertExternalApprovalSettingNodes(nodes []*v1pb.ExternalApprovalSetting_Node) []*storepb.ExternalApprovalSetting_Node {
	storeNodes := make([]*storepb.ExternalApprovalSetting_Node, len(nodes))
	for i := range nodes {
		storeNodes[i] = convertExternalApprovalSettingNode(nodes[i])
	}
	return storeNodes
}

func convertExternalApprovalSettingNode(o *v1pb.ExternalApprovalSetting_Node) *storepb.ExternalApprovalSetting_Node {
	return &storepb.ExternalApprovalSetting_Node{
		Id:       o.Id,
		Title:    o.Title,
		Endpoint: o.Endpoint,
	}
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

func convertSchemaTemplateSetting(ctx context.Context, template *storepb.SchemaTemplateSetting) (*v1pb.SchemaTemplateSetting, error) {
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
		if v.Config != nil {
			t.Config = convertStoreColumnConfig(v.Config)
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
		if v.Config != nil {
			t.Config = convertStoreTableConfig(ctx, v.Config, nil /* optionalStores */)
		}
		v1Setting.TableTemplates = append(v1Setting.TableTemplates, t)
	}

	return v1Setting, nil
}

func convertV1SchemaTemplateSetting(ctx context.Context, template *v1pb.SchemaTemplateSetting) (*storepb.SchemaTemplateSetting, error) {
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
		if v.Config != nil {
			t.Config = convertV1ColumnConfig(v.Config)
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
		if v.Config != nil {
			t.Config = convertV1TableConfig(ctx, v.Config, nil /* optionalStores */)
		}
		v1Setting.TableTemplates = append(v1Setting.TableTemplates, t)
	}

	return v1Setting, nil
}

func validateMaskingAlgorithm(algorithm *v1pb.MaskingAlgorithmSetting_Algorithm) error {
	if !isValidUUID(algorithm.Id) {
		return status.Errorf(codes.InvalidArgument, "invalid masking algorithm id format: %s", algorithm.Id)
	}
	if preservedMaskingAlgorithmIDMatcher.MatchString(algorithm.Id) {
		return status.Errorf(codes.InvalidArgument, "masking algorithm id cannot be preserved id: %s", algorithm.Id)
	}
	if algorithm.Title == "" {
		return status.Errorf(codes.InvalidArgument, "masking algorithm title cannot be empty: %s", algorithm.Id)
	}

	switch algorithm.Category {
	case "MASK":
		if algorithm.Mask == nil {
			return nil
		}
		switch m := algorithm.Mask.(type) {
		case *v1pb.MaskingAlgorithmSetting_Algorithm_FullMask_:
			if err := checkSubstitution(m.FullMask.Substitution); err != nil {
				return err
			}
		case *v1pb.MaskingAlgorithmSetting_Algorithm_RangeMask_:
			for i, slice := range m.RangeMask.Slices {
				if slice.Substitution == "" {
					return status.Errorf(codes.InvalidArgument, "the substitution for slice is required")
				}
				if len(slice.Substitution) > 16 {
					return status.Errorf(codes.InvalidArgument, "the substitution should less than 16 bytes")
				}
				for j := 0; j < i; j++ {
					pre := m.RangeMask.Slices[j]
					if slice.Start >= pre.End || pre.Start >= slice.End {
						continue
					}
					return status.Errorf(codes.InvalidArgument, "the slice range cannot overlap: [%d,%d) and [%d,%d)", pre.Start, pre.End, slice.Start, slice.End)
				}
			}
		case *v1pb.MaskingAlgorithmSetting_Algorithm_InnerOuterMask_:
			if err := checkSubstitution(m.InnerOuterMask.Substitution); err != nil {
				return err
			}
		default:
			return status.Errorf(codes.InvalidArgument, "mismatch masking algorithm category and mask type: %T, %s", algorithm.Mask, algorithm.Category)
		}
	case "HASH":
		if algorithm.Mask == nil {
			return nil
		}
		switch algorithm.Mask.(type) {
		case *v1pb.MaskingAlgorithmSetting_Algorithm_Md5Mask:
		default:
			return status.Errorf(codes.InvalidArgument, "mismatch masking algorithm category and mask type: %T, %s", algorithm.Mask, algorithm.Category)
		}
	default:
		return status.Errorf(codes.InvalidArgument, "invalid masking algorithm category: %s", algorithm.Category)
	}

	return nil
}

func checkSubstitution(substitution string) error {
	if substitution == "" {
		return status.Errorf(codes.InvalidArgument, "the substitution for inner or outer masks is required")
	}
	if len(substitution) > 16 {
		return status.Errorf(codes.InvalidArgument, "the substitution should less than 16 bytes")
	}
	return nil
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
