package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/metric"
	metriccollector "github.com/bytebase/bytebase/backend/metric/collector"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) initializeSetting(ctx context.Context) error {
	// secretLength is the length for the secret used to sign the JWT auto token.
	const secretLength = 32

	// initial branding
	_, firstTimeOnboarding, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_BRANDING_LOGO,
		Value: "",
	})
	if err != nil {
		return err
	}

	// initial JWT token
	secret, err := common.RandomString(secretLength)
	if err != nil {
		return errors.Wrap(err, "failed to generate random JWT secret")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_AUTH_SECRET,
		Value: secret,
	}); err != nil {
		return err
	}

	// initial workspace
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_WORKSPACE_ID,
		Value: uuid.New().String(),
	}); err != nil {
		return err
	}

	// Init SCIM config
	scimToken, err := common.RandomString(secretLength)
	if err != nil {
		return errors.Wrap(err, "failed to generate random SCIM secret")
	}
	scimSettingValue, err := protojson.Marshal(&storepb.SCIMSetting{
		Token: scimToken,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial scim setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_SCIM,
		Value: string(scimSettingValue),
	}); err != nil {
		return err
	}

	// Init password validation
	passwordSettingValue, err := protojson.Marshal(&storepb.PasswordRestrictionSetting{
		MinLength:                         8,
		RequireNumber:                     false,
		RequireLetter:                     false,
		RequireUppercaseLetter:            false,
		RequireSpecialCharacter:           false,
		RequireResetPasswordForFirstLogin: false,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial password validation setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_PASSWORD_RESTRICTION,
		Value: string(passwordSettingValue),
	}); err != nil {
		return err
	}

	// initial license
	if _, _, err = s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_ENTERPRISE_LICENSE,
		Value: "",
	}); err != nil {
		return err
	}

	// initial IM app
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_APP_IM,
		Value: "{}",
	}); err != nil {
		return err
	}

	// initial watermark setting
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_WATERMARK,
		Value: "0",
	}); err != nil {
		return err
	}

	// initial schema template setting
	schemaTemplateSettingValue, err := protojson.Marshal(&storepb.SchemaTemplateSetting{})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial schema template setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_SCHEMA_TEMPLATE,
		Value: string(schemaTemplateSettingValue),
	}); err != nil {
		return err
	}

	// initial data classification setting
	initialDataClassification := &storepb.DataClassificationSetting{
		Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
			{
				Id:    "default-sensitive-classification",
				Title: "默认敏感数据分级",
				Levels: []*storepb.DataClassificationSetting_DataClassificationConfig_Level{
					{
						Id:          "high",
						Title:       "高敏感",
						Description: "包含用户核心隐私数据，如密码、银行卡号、身份证号等",
					},
					{
						Id:          "medium",
						Title:       "中敏感",
						Description: "包含用户个人可识别信息，如手机号、邮箱、地址等",
					},
					{
						Id:          "low",
						Title:       "低敏感",
						Description: "包含用户基础信息，如用户名、头像URL等",
					},
				},
				Classification: map[string]*storepb.DataClassificationSetting_DataClassificationConfig_DataClassification{
					"password": {
						Id:          "password",
						Title:       "密码",
						Description: "用户登录密码或支付密码",
						LevelId:     "high",
					},
					"phone": {
						Id:          "phone",
						Title:       "手机号",
						Description: "用户手机号码",
						LevelId:     "medium",
					},
					"email": {
						Id:          "email",
						Title:       "邮箱",
						Description: "用户电子邮箱",
						LevelId:     "medium",
					},
					"id_card": {
						Id:          "id_card",
						Title:       "身份证号",
						Description: "居民身份证号码",
						LevelId:     "high",
					},
					"bank_card": {
						Id:          "bank_card",
						Title:       "银行卡号",
						Description: "银行借记卡/信用卡号码",
						LevelId:     "high",
					},
					"address": {
						Id:          "address",
						Title:       "地址",
						Description: "用户居住或工作地址",
						LevelId:     "low",
					},
					"username": {
						Id:          "username",
						Title:       "用户名",
						Description: "用户账号名称",
						LevelId:     "low",
					},
				},
			},
		},
	}
	dataClassificationSettingValue, err := protojson.Marshal(initialDataClassification)
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial data classification setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_DATA_CLASSIFICATION,
		Value: string(dataClassificationSettingValue),
	}); err != nil {
		return err
	}

	// initial workspace approval setting
	approvalSettingValue, err := protojson.Marshal(&storepb.WorkspaceApprovalSetting{})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial workspace approval setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name: storepb.SettingName_WORKSPACE_APPROVAL,
		// Value is ""
		Value: string(approvalSettingValue),
	}); err != nil {
		return err
	}

	// initial workspace profile setting
	workspaceProfileSetting, err := s.store.GetSettingV2(ctx, storepb.SettingName_WORKSPACE_PROFILE)
	if err != nil {
		return err
	}

	workspaceProfilePayload := &storepb.WorkspaceProfileSetting{
		ExternalUrl:            s.profile.ExternalURL,
		EnableMetricCollection: true, // Default to enabled for new installations
	}
	if workspaceProfileSetting != nil {
		workspaceProfilePayload = new(storepb.WorkspaceProfileSetting)
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(workspaceProfileSetting.Value), workspaceProfilePayload); err != nil {
			return err
		}
		if s.profile.ExternalURL != "" {
			workspaceProfilePayload.ExternalUrl = s.profile.ExternalURL
		}
	}

	bytes, err := protojson.Marshal(workspaceProfilePayload)
	if err != nil {
		return err
	}

	if _, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  storepb.SettingName_WORKSPACE_PROFILE,
		Value: string(bytes),
	}); err != nil {
		return err
	}

	if firstTimeOnboarding {
		// Only grant workspace member role to allUsers at the first time.
		if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Member: common.AllUsers,
			Roles: []string{
				common.FormatRole(common.WorkspaceMember),
			},
		}); err != nil {
			return err
		}
	}

	// Init workspace environment setting
	environmentSettingValue, err := protojson.Marshal(&storepb.EnvironmentSetting{
		Environments: []*storepb.EnvironmentSetting_Environment{
			{
				Title: "Test",
				Id:    "test",
			},
			{
				Title: "Prod",
				Id:    "prod",
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal initial environment setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  storepb.SettingName_ENVIRONMENT,
		Value: string(environmentSettingValue),
	}); err != nil {
		return err
	}

	return nil
}

// initMetricReporter will initial the metric scheduler.
func (s *Server) initMetricReporter() {
	metricReporter := metricreport.NewReporter(s.store, s.licenseService, s.profile)
	metricReporter.Register(metric.InstanceCountMetricName, metriccollector.NewInstanceCountCollector(s.store))
	metricReporter.Register(metric.IssueCountMetricName, metriccollector.NewIssueCountCollector(s.store))
	metricReporter.Register(metric.ProjectCountMetricName, metriccollector.NewProjectCountCollector(s.store))
	metricReporter.Register(metric.UserCountMetricName, metriccollector.NewUserCountCollector(s.store))
	s.metricReporter = metricReporter
}
