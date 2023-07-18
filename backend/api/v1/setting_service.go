package v1

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
	"github.com/bytebase/bytebase/backend/plugin/mail"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/edit"
)

// SettingService implements the setting service.
type SettingService struct {
	v1pb.UnimplementedSettingServiceServer
	store          *store.Store
	profile        *config.Profile
	licenseService enterpriseAPI.LicenseService
	stateCfg       *state.State
	feishuProvider *feishu.Provider
}

// NewSettingService creates a new setting service.
func NewSettingService(
	store *store.Store,
	profile *config.Profile,
	licenseService enterpriseAPI.LicenseService,
	stateCfg *state.State,
	feishuProvider *feishu.Provider,
) *SettingService {
	return &SettingService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		feishuProvider: feishuProvider,
	}
}

// Some settings contain secret info so we only return settings that are needed by the client.
var whitelistSettings = []api.SettingName{
	api.SettingBrandingLogo,
	api.SettingAppIM,
	api.SettingWatermark,
	api.SettingPluginOpenAIKey,
	api.SettingPluginOpenAIEndpoint,
	api.SettingWorkspaceApproval,
	api.SettingWorkspaceMailDelivery,
	api.SettingWorkspaceProfile,
	api.SettingWorkspaceExternalApproval,
	api.SettingEnterpriseTrial,
	api.SettingSchemaTemplate,
}

var (
	//go:embed mail_templates/testmail/template.html
	//go:embed mail_templates/testmail/statics/logo-full.png
	//go:embed mail_templates/testmail/statics/banner.png
	testEmailFs embed.FS
)

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
	settingName, err := getSettingName(request.Name)
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
func (s *SettingService) SetSetting(ctx context.Context, request *v1pb.SetSettingRequest) (*v1pb.Setting, error) {
	settingName, err := getSettingName(request.Setting.Name)
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
	var storeSettingValue string
	switch apiSettingName {
	case api.SettingWorkspaceProfile:
		payload := new(storepb.WorkspaceProfileSetting)
		if err := convertV1PbToStorePb(request.Setting.Value.GetWorkspaceProfileSettingValue(), payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		if payload.ExternalUrl != "" {
			externalURL, err := common.NormalizeExternalURL(payload.ExternalUrl)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid external url: %v", err)
			}
			payload.ExternalUrl = externalURL
		}
		if payload.GitopsWebhookUrl != "" {
			gitopsWebhookURL, err := common.NormalizeExternalURL(payload.GitopsWebhookUrl)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid GitOps webhook URL: %v", err)
			}
			payload.GitopsWebhookUrl = gitopsWebhookURL
		}
		bytes, err := protojson.Marshal(payload)
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
			email, err := getUserEmail(rule.Template.Creator)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("failed to get creator: %v", err))
			}
			if email == api.SystemBotEmail {
				creatorID = api.SystemBotID
			} else {
				creator, err := s.store.GetUser(ctx, &store.FindUserMessage{
					Email: &email,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
				}
				if creator == nil {
					return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("creator %s not found", rule.Template.Creator))
				}
				creatorID = creator.ID
			}

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
			if err := protojson.Unmarshal([]byte(oldStoreSetting.Value), oldValue); err != nil {
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
		settingValue := request.Setting.Value.GetAppImSettingValue()
		imType, err := convertToIMType(settingValue.ImType)
		if err != nil {
			return nil, err
		}
		payload := &api.SettingAppIMValue{
			IMType:    imType,
			AppID:     settingValue.AppId,
			AppSecret: settingValue.AppSecret,
			ExternalApproval: api.ExternalApproval{
				Enabled:              settingValue.ExternalApproval.Enabled,
				ApprovalDefinitionID: settingValue.ExternalApproval.ApprovalDefinitionId,
			},
		}
		if payload.IMType != api.IMTypeFeishu {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("unknown IM Type %s", payload.IMType))
		}
		if payload.ExternalApproval.Enabled {
			if err := s.licenseService.IsFeatureEnabled(api.FeatureIMApproval); err != nil {
				return nil, status.Errorf(codes.PermissionDenied, err.Error())
			}

			if payload.AppID == "" || payload.AppSecret == "" {
				return nil, status.Errorf(codes.InvalidArgument, "application ID and secret cannot be empty")
			}

			p := s.feishuProvider
			// clear token cache so that we won't use the previous token.
			p.ClearTokenCache()

			// check bot info
			if _, err := p.GetBotID(ctx, feishu.TokenCtx{
				AppID:     payload.AppID,
				AppSecret: payload.AppSecret,
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get bot id. Hint: check if bot is enabled")
			}

			// create approval definition
			approvalDefinitionID, err := p.CreateApprovalDefinition(ctx, feishu.TokenCtx{
				AppID:     payload.AppID,
				AppSecret: payload.AppSecret,
			}, "")
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create approval definition: %v", err)
			}
			payload.ExternalApproval.ApprovalDefinitionID = approvalDefinitionID
		}

		s, err := json.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal approval setting: %v", err)
		}
		storeSettingValue = string(s)
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
	case api.SettingEnterpriseTrial:
		return nil, status.Errorf(codes.InvalidArgument, "cannot set setting %s", settingName)
	case api.SettingSchemaTemplate:
		oldSetting, err := s.store.GetSchemaTemplateSetting(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get schema template setting: %v", err)
		}
		oldTemplateMap := map[string]*v1pb.SchemaTemplateSetting_FieldTemplate{}
		for _, template := range convertToSchemaTemplateSetting(oldSetting).FieldTemplates {
			oldTemplateMap[template.Id] = template
		}

		schemaTemplateSetting := request.Setting.Value.GetSchemaTemplateSettingValue()
		if schemaTemplateSetting == nil {
			return nil, status.Errorf(codes.InvalidArgument, "value cannot be nil when setting schema template setting")
		}

		// validate the changed template
		for _, template := range schemaTemplateSetting.FieldTemplates {
			oldTemplate, ok := oldTemplateMap[template.Id]
			if ok && cmp.Equal(oldTemplate, template, protocmp.Transform()) {
				continue
			}
			engineType := parser.EngineType(template.Engine.String())
			var defaultVal string
			if template.Column.Default != nil {
				defaultVal = template.Column.Default.Value
			}
			validateResultList, err := edit.ValidateDatabaseEdit(engineType, &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "validation",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     template.Column.Name,
								Type:     template.Column.Type,
								Default:  &defaultVal,
								Nullable: template.Column.Nullable,
								Comment:  template.Column.Comment,
							},
						},
					},
				},
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to validate template, error: %v", err)
			}
			if len(validateResultList) != 0 {
				return nil, status.Errorf(codes.InvalidArgument, validateResultList[0].Message)
			}
		}

		payload := new(storepb.SchemaTemplateSetting)
		if err := convertV1PbToStorePb(schemaTemplateSetting, payload); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", apiSettingName, err)
		}
		bytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal external approval setting, error: %v", err)
		}
		storeSettingValue = string(bytes)
	default:
		storeSettingValue = request.Setting.Value.GetStringValue()
	}
	setting, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  apiSettingName,
		Value: storeSettingValue,
	}, ctx.Value(common.PrincipalIDContextKey).(int))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set setting: %v", err)
	}

	settingMessage, err := s.convertToSettingMessage(ctx, setting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert setting message: %v", err)
	}
	return settingMessage, nil
}

func convertV1PbToStorePb(inputPB, outputPB protoreflect.ProtoMessage) error {
	bytes, err := protojson.Marshal(inputPB)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal setting: %v", err)
	}
	if err := protojson.Unmarshal(bytes, outputPB); err != nil {
		return status.Errorf(codes.Internal, "failed to unmarshal setting: %v", err)
	}
	return nil
}

func (s *SettingService) convertToSettingMessage(ctx context.Context, setting *store.SettingMessage) (*v1pb.Setting, error) {
	settingName := fmt.Sprintf("%s%s", settingNamePrefix, setting.Name)
	switch setting.Name {
	case api.SettingWorkspaceMailDelivery:
		storeValue := new(storepb.SMTPMailDeliverySetting)
		if err := protojson.Unmarshal([]byte(setting.Value), storeValue); err != nil {
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
		apiValue := new(api.SettingAppIMValue)
		stringValue := setting.Value
		if stringValue == "" {
			stringValue = "{}"
		}
		if err := json.Unmarshal([]byte(stringValue), apiValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting value for %s with error: %v", setting.Name, err)
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_AppImSettingValue{
					AppImSettingValue: &v1pb.AppIMSetting{
						ImType:    convertV1IMType(apiValue.IMType),
						AppId:     apiValue.AppID,
						AppSecret: apiValue.AppSecret,
						ExternalApproval: &v1pb.AppIMSetting_ExternalApproval{
							Enabled:              apiValue.ExternalApproval.Enabled,
							ApprovalDefinitionId: apiValue.ExternalApproval.ApprovalDefinitionID,
						},
					},
				},
			},
		}, nil
	case api.SettingPluginAgent:
		v1Value := new(v1pb.AgentPluginSetting)
		if err := protojson.Unmarshal([]byte(setting.Value), v1Value); err != nil {
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
		if err := protojson.Unmarshal([]byte(setting.Value), v1Value); err != nil {
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
		if err := protojson.Unmarshal([]byte(setting.Value), storeValue); err != nil {
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
				template.Creator = fmt.Sprintf("%s%s", userNamePrefix, creator.Email)
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
		if err := protojson.Unmarshal([]byte(setting.Value), storeValue); err != nil {
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
		storeValue := new(storepb.SchemaTemplateSetting)
		if err := protojson.Unmarshal([]byte(setting.Value), storeValue); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal setting values for %s with error: %v", setting.Name, err)
		}
		v1Value := convertToSchemaTemplateSetting(storeValue)
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.Value{
				Value: &v1pb.Value_SchemaTemplateSettingValue{
					SchemaTemplateSettingValue: v1Value,
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

func convertToIMType(imType v1pb.AppIMSetting_IMType) (api.IMType, error) {
	var resp api.IMType
	switch imType {
	case v1pb.AppIMSetting_FEISHU:
		resp = api.IMTypeFeishu
	default:
		return resp, status.Errorf(codes.InvalidArgument, "unknown im type %v", imType.String())
	}
	return resp, nil
}

func convertV1IMType(imType api.IMType) v1pb.AppIMSetting_IMType {
	switch imType {
	case api.IMTypeFeishu:
		return v1pb.AppIMSetting_FEISHU
	default:
		return v1pb.AppIMSetting_IM_TYPE_UNSPECIFIED
	}
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
		if err := protojson.Unmarshal([]byte(setting.Value), settingValue); err != nil {
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

func convertToSchemaTemplateSetting(s *storepb.SchemaTemplateSetting) *v1pb.SchemaTemplateSetting {
	v1Templates := []*v1pb.SchemaTemplateSetting_FieldTemplate{}
	for _, template := range s.FieldTemplates {
		v1Templates = append(v1Templates, &v1pb.SchemaTemplateSetting_FieldTemplate{
			Id:     template.Id,
			Engine: v1pb.Engine(template.Engine),
			Column: &v1pb.ColumnMetadata{
				Name:     template.Column.Name,
				Type:     template.Column.Type,
				Default:  template.Column.Default,
				Nullable: template.Column.Nullable,
				Comment:  template.Column.Comment,
			},
		})
	}

	return &v1pb.SchemaTemplateSetting{
		FieldTemplates: v1Templates,
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
	settingName, err := getSettingName(setting.Name)
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
