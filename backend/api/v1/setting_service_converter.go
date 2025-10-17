package v1

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

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

	return &storepb.ApprovalFlow{
		Roles: v1Flow.Roles,
	}
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
