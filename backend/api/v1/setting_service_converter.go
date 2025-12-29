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
		storeValue, ok := setting.Value.(*storepb.AppIMSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		v1Value := convertToAppIMSetting(storeValue)
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_AppIm{
					AppIm: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_WORKSPACE_PROFILE:
		storeValue, ok := setting.Value.(*storepb.WorkspaceProfileSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		v1Value := convertToWorkspaceProfileSetting(storeValue)
		v1Value.DisallowSignup = v1Value.DisallowSignup || profile.SaaS
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceProfile{
					WorkspaceProfile: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_WORKSPACE_APPROVAL:
		storeValue, ok := setting.Value.(*storepb.WorkspaceApprovalSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		v1Value := &v1pb.WorkspaceApprovalSetting{}
		for _, rule := range storeValue.Rules {
			template := convertToApprovalTemplate(rule.Template)
			v1Value.Rules = append(v1Value.Rules, &v1pb.WorkspaceApprovalSetting_Rule{
				Source:    v1pb.WorkspaceApprovalSetting_Rule_Source(rule.Source),
				Condition: rule.Condition,
				Template:  template,
			})
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{
					WorkspaceApproval: v1Value,
				},
			},
		}, nil
	case storepb.SettingName_DATA_CLASSIFICATION:
		storeValue, ok := setting.Value.(*storepb.DataClassificationSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_DataClassification{
					DataClassification: convertToDataClassificationSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_SEMANTIC_TYPES:
		storeValue, ok := setting.Value.(*storepb.SemanticTypeSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_SemanticType{
					SemanticType: convertToSemanticTypeSetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_AI:
		storeValue, ok := setting.Value.(*storepb.AISetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Ai{
					Ai: convertToAISetting(storeValue),
				},
			},
		}, nil
	case storepb.SettingName_ENVIRONMENT:
		storeValue, ok := setting.Value.(*storepb.EnvironmentSetting)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("invalid setting value type for %s", setting.Name))
		}
		v1Value := convertToEnvironmentSetting(storeValue)
		return &v1pb.Setting{
			Name: settingName,
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Environment{
					Environment: v1Value,
				},
			},
		}, nil
	default:
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("unsupported setting %v", setting.Name))
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
	case storepb.SettingName_WORKSPACE_PROFILE:
		return v1pb.Setting_WORKSPACE_PROFILE
	case storepb.SettingName_WORKSPACE_APPROVAL:
		return v1pb.Setting_WORKSPACE_APPROVAL
	case storepb.SettingName_APP_IM:
		return v1pb.Setting_APP_IM
	case storepb.SettingName_AI:
		return v1pb.Setting_AI
	case storepb.SettingName_DATA_CLASSIFICATION:
		return v1pb.Setting_DATA_CLASSIFICATION
	case storepb.SettingName_SEMANTIC_TYPES:
		return v1pb.Setting_SEMANTIC_TYPES
	case storepb.SettingName_ENVIRONMENT:
		return v1pb.Setting_ENVIRONMENT
	case storepb.SettingName_SYSTEM:
		// Backend-only setting, not exposed in v1 API
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
	case v1pb.Setting_WORKSPACE_PROFILE:
		return storepb.SettingName_WORKSPACE_PROFILE
	case v1pb.Setting_WORKSPACE_APPROVAL:
		return storepb.SettingName_WORKSPACE_APPROVAL
	case v1pb.Setting_APP_IM:
		return storepb.SettingName_APP_IM
	case v1pb.Setting_AI:
		return storepb.SettingName_AI
	case v1pb.Setting_DATA_CLASSIFICATION:
		return storepb.SettingName_DATA_CLASSIFICATION
	case v1pb.Setting_SEMANTIC_TYPES:
		return storepb.SettingName_SEMANTIC_TYPES
	case v1pb.Setting_ENVIRONMENT:
		return storepb.SettingName_ENVIRONMENT
	default:
		return storepb.SettingName_SETTING_NAME_UNSPECIFIED
	}
}

func convertToEnvironmentSetting(storeValue *storepb.EnvironmentSetting) *v1pb.EnvironmentSetting {
	if storeValue == nil {
		return nil
	}
	var environments []*v1pb.EnvironmentSetting_Environment

	for _, e := range storeValue.Environments {
		environments = append(environments, convertToEnvironment(e))
	}
	return &v1pb.EnvironmentSetting{
		Environments: environments,
	}
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
		RefreshTokenDuration:   v1Setting.RefreshTokenDuration,
		AccessTokenDuration:    v1Setting.AccessTokenDuration,
		InactiveSessionTimeout: v1Setting.InactiveSessionTimeout,
		MaximumRoleExpiration:  v1Setting.MaximumRoleExpiration,
		Domains:                v1Setting.Domains,
		EnforceIdentityDomain:  v1Setting.EnforceIdentityDomain,
		DatabaseChangeMode:     storepb.WorkspaceProfileSetting_DatabaseChangeMode(v1Setting.DatabaseChangeMode),
		DisallowPasswordSignin: v1Setting.DisallowPasswordSignin,
		EnableMetricCollection: v1Setting.EnableMetricCollection,
		EnableAuditLogStdout:   v1Setting.EnableAuditLogStdout,
		Watermark:              v1Setting.Watermark,
		DirectorySyncToken:     v1Setting.DirectorySyncToken,
		BrandingLogo:           v1Setting.BrandingLogo,
		PasswordRestriction:    convertPasswordRestrictionSetting(v1Setting.PasswordRestriction),
	}

	// Convert announcement if present
	if v1Setting.Announcement != nil {
		storeSetting.Announcement = &storepb.WorkspaceProfileSetting_Announcement{
			Text: v1Setting.Announcement.Text,
			Link: v1Setting.Announcement.Link,
		}
		// Convert alert level
		switch v1Setting.Announcement.Level {
		case v1pb.Announcement_ALERT_LEVEL_UNSPECIFIED:
			storeSetting.Announcement.Level = storepb.WorkspaceProfileSetting_Announcement_ALERT_LEVEL_UNSPECIFIED
		case v1pb.Announcement_INFO:
			storeSetting.Announcement.Level = storepb.WorkspaceProfileSetting_Announcement_INFO
		case v1pb.Announcement_WARNING:
			storeSetting.Announcement.Level = storepb.WorkspaceProfileSetting_Announcement_WARNING
		case v1pb.Announcement_CRITICAL:
			storeSetting.Announcement.Level = storepb.WorkspaceProfileSetting_Announcement_CRITICAL
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
		RefreshTokenDuration:   storeSetting.RefreshTokenDuration,
		AccessTokenDuration:    storeSetting.AccessTokenDuration,
		InactiveSessionTimeout: storeSetting.InactiveSessionTimeout,
		MaximumRoleExpiration:  storeSetting.MaximumRoleExpiration,
		Domains:                storeSetting.Domains,
		EnforceIdentityDomain:  storeSetting.EnforceIdentityDomain,
		DatabaseChangeMode:     v1pb.DatabaseChangeMode(storeSetting.DatabaseChangeMode),
		DisallowPasswordSignin: storeSetting.DisallowPasswordSignin,
		EnableMetricCollection: storeSetting.EnableMetricCollection,
		EnableAuditLogStdout:   storeSetting.EnableAuditLogStdout,
		Watermark:              storeSetting.Watermark,
		DirectorySyncToken:     storeSetting.DirectorySyncToken,
		BrandingLogo:           storeSetting.BrandingLogo,
		PasswordRestriction:    convertToPasswordRestrictionSetting(storeSetting.PasswordRestriction),
	}

	if storeSetting.Announcement != nil {
		v1Setting.Announcement = &v1pb.Announcement{
			Text: storeSetting.Announcement.Text,
			Link: storeSetting.Announcement.Link,
		}
		switch storeSetting.Announcement.Level {
		case storepb.WorkspaceProfileSetting_Announcement_ALERT_LEVEL_UNSPECIFIED:
			v1Setting.Announcement.Level = v1pb.Announcement_ALERT_LEVEL_UNSPECIFIED
		case storepb.WorkspaceProfileSetting_Announcement_INFO:
			v1Setting.Announcement.Level = v1pb.Announcement_INFO
		case storepb.WorkspaceProfileSetting_Announcement_WARNING:
			v1Setting.Announcement.Level = v1pb.Announcement_WARNING
		case storepb.WorkspaceProfileSetting_Announcement_CRITICAL:
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

func convertAppIMSetting(v1Setting *v1pb.AppIMSetting) (*storepb.AppIMSetting, error) {
	if v1Setting == nil {
		return nil, nil
	}

	storeSetting := &storepb.AppIMSetting{}
	findIMType := map[v1pb.WebhookType]bool{}
	for _, setting := range v1Setting.Settings {
		if findIMType[setting.Type] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("duplicate im type %v", setting.Type.String()))
		}
		findIMType[setting.Type] = true

		imSetting := &storepb.AppIMSetting_IMSetting{
			Type: storepb.WebhookType(setting.Type),
		}
		// Handle based on Type field since protobuf-es may serialize oneof incorrectly.
		// The oneof payload type may not match the Type field due to serialization issues.
		switch setting.Type {
		case v1pb.WebhookType_SLACK:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Slack{
				Slack: &storepb.AppIMSetting_Slack{
					Token: setting.GetSlack().GetToken(),
				},
			}
		case v1pb.WebhookType_FEISHU:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Feishu{
				Feishu: &storepb.AppIMSetting_Feishu{
					AppId:     setting.GetFeishu().GetAppId(),
					AppSecret: setting.GetFeishu().GetAppSecret(),
				},
			}
		case v1pb.WebhookType_WECOM:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Wecom{
				Wecom: &storepb.AppIMSetting_Wecom{
					CorpId:  setting.GetWecom().GetCorpId(),
					AgentId: setting.GetWecom().GetAgentId(),
					Secret:  setting.GetWecom().GetSecret(),
				},
			}
		case v1pb.WebhookType_LARK:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Lark{
				Lark: &storepb.AppIMSetting_Lark{
					AppId:     setting.GetLark().GetAppId(),
					AppSecret: setting.GetLark().GetAppSecret(),
				},
			}
		case v1pb.WebhookType_DINGTALK:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Dingtalk{
				Dingtalk: &storepb.AppIMSetting_DingTalk{
					ClientId:     setting.GetDingtalk().GetClientId(),
					ClientSecret: setting.GetDingtalk().GetClientSecret(),
					RobotCode:    setting.GetDingtalk().GetRobotCode(),
				},
			}
		case v1pb.WebhookType_TEAMS:
			imSetting.Payload = &storepb.AppIMSetting_IMSetting_Teams{
				Teams: &storepb.AppIMSetting_Teams{
					TenantId:     setting.GetTeams().GetTenantId(),
					ClientId:     setting.GetTeams().GetClientId(),
					ClientSecret: setting.GetTeams().GetClientSecret(),
				},
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupported im type %v", setting.Type.String()))
		}
		storeSetting.Settings = append(storeSetting.Settings, imSetting)
	}

	return storeSetting, nil
}

func convertToAppIMSetting(storeSetting *storepb.AppIMSetting) *v1pb.AppIMSetting {
	if storeSetting == nil {
		return nil
	}

	v1Setting := &v1pb.AppIMSetting{}
	for _, setting := range storeSetting.Settings {
		imSetting := &v1pb.AppIMSetting_IMSetting{
			Type: v1pb.WebhookType(setting.Type),
		}
		switch setting.Type {
		case storepb.WebhookType_SLACK:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Slack{
				Slack: &v1pb.AppIMSetting_Slack{},
			}
		case storepb.WebhookType_FEISHU:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Feishu{
				Feishu: &v1pb.AppIMSetting_Feishu{},
			}
		case storepb.WebhookType_WECOM:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Wecom{
				Wecom: &v1pb.AppIMSetting_Wecom{},
			}
		case storepb.WebhookType_LARK:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Lark{
				Lark: &v1pb.AppIMSetting_Lark{},
			}
		case storepb.WebhookType_DINGTALK:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Dingtalk{
				Dingtalk: &v1pb.AppIMSetting_DingTalk{},
			}
		case storepb.WebhookType_TEAMS:
			imSetting.Payload = &v1pb.AppIMSetting_IMSetting_Teams{
				Teams: &v1pb.AppIMSetting_Teams{},
			}
		default:
		}
		v1Setting.Settings = append(v1Setting.Settings, imSetting)
	}

	return v1Setting
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
		Id:             c.Id,
		Title:          c.Title,
		Levels:         convertDataClassificationSettingLevels(c.Levels),
		Classification: convertDataClassificationSettingClassification(c.Classification),
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
		Id:             c.Id,
		Title:          c.Title,
		Levels:         convertToDataClassificationSettingLevels(c.Levels),
		Classification: convertToDataClassificationSettingClassification(c.Classification),
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

func convertPasswordRestrictionSetting(v1Setting *v1pb.WorkspaceProfileSetting_PasswordRestriction) *storepb.WorkspaceProfileSetting_PasswordRestriction {
	if v1Setting == nil {
		return nil
	}

	return &storepb.WorkspaceProfileSetting_PasswordRestriction{
		MinLength:                         v1Setting.MinLength,
		RequireNumber:                     v1Setting.RequireNumber,
		RequireLetter:                     v1Setting.RequireLetter,
		RequireUppercaseLetter:            v1Setting.RequireUppercaseLetter,
		RequireSpecialCharacter:           v1Setting.RequireSpecialCharacter,
		RequireResetPasswordForFirstLogin: v1Setting.RequireResetPasswordForFirstLogin,
		PasswordRotation:                  v1Setting.GetPasswordRotation(),
	}
}

func convertToPasswordRestrictionSetting(storeSetting *storepb.WorkspaceProfileSetting_PasswordRestriction) *v1pb.WorkspaceProfileSetting_PasswordRestriction {
	if storeSetting == nil {
		return nil
	}

	return &v1pb.WorkspaceProfileSetting_PasswordRestriction{
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
		// Do not return the API key for security reasons.
		ApiKey:  "",
		Model:   storeSetting.Model,
		Version: storeSetting.Version,
	}
}
