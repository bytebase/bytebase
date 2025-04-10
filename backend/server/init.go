package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/metric"
	metriccollector "github.com/bytebase/bytebase/backend/metric/collector"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *Server) getInitSetting(ctx context.Context) error {
	// secretLength is the length for the secret used to sign the JWT auto token.
	const secretLength = 32

	// initial branding
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingBrandingLogo,
		Value: "",
	}); err != nil {
		return err
	}

	// initial JWT token
	secret, err := common.RandomString(secretLength)
	if err != nil {
		return errors.Wrap(err, "failed to generate random JWT secret")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingAuthSecret,
		Value: secret,
	}); err != nil {
		return err
	}

	// initial workspace
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingWorkspaceID,
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
		Name:  base.SettingSCIM,
		Value: string(scimSettingValue),
	}); err != nil {
		return err
	}

	// Init password validation
	passwordSettingValue, err := protojson.Marshal(&storepb.PasswordRestrictionSetting{
		MinLength:                         8,
		RequireNumber:                     false,
		RequireLetter:                     true,
		RequireUppercaseLetter:            false,
		RequireSpecialCharacter:           false,
		RequireResetPasswordForFirstLogin: false,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial password validation setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingPasswordRestriction,
		Value: string(passwordSettingValue),
	}); err != nil {
		return err
	}

	// initial license
	if _, _, err = s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingEnterpriseLicense,
		Value: "",
	}); err != nil {
		return err
	}

	// initial IM app
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingAppIM,
		Value: "{}",
	}); err != nil {
		return err
	}

	// initial watermark setting
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingWatermark,
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
		Name:  base.SettingSchemaTemplate,
		Value: string(schemaTemplateSettingValue),
	}); err != nil {
		return err
	}

	// initial data classification setting
	dataClassificationSettingValue, err := protojson.Marshal(&storepb.DataClassificationSetting{})
	if err != nil {
		return errors.Wrap(err, "failed to marshal initial data classification setting")
	}
	if _, _, err := s.store.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:  base.SettingDataClassification,
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
		Name: base.SettingWorkspaceApproval,
		// Value is ""
		Value: string(approvalSettingValue),
	}); err != nil {
		return err
	}

	// initial workspace profile setting
	workspaceProfileSetting, err := s.store.GetSettingV2(ctx, base.SettingWorkspaceProfile)
	if err != nil {
		return err
	}

	workspaceProfilePayload := &storepb.WorkspaceProfileSetting{
		ExternalUrl: s.profile.ExternalURL,
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
		Name:  base.SettingWorkspaceProfile,
		Value: string(bytes),
	}); err != nil {
		return err
	}

	// Init workspace IAM policy
	if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Member: common.FormatUserUID(base.SystemBotID),
		Roles: []string{
			common.FormatRole(base.WorkspaceAdmin.String()),
		},
	}); err != nil {
		return err
	}
	if _, err := s.store.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Member: base.AllUsers,
		Roles: []string{
			common.FormatRole(base.WorkspaceMember.String()),
		},
	}); err != nil {
		return err
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
		Name:  base.SettingEnvironment,
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
