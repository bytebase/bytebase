package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
	metriccollector "github.com/bytebase/bytebase/backend/metric/collector"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (s *Server) getInitSetting(ctx context.Context, datastore *store.Store) (string, string, time.Duration, error) {
	// secretLength is the length for the secret used to sign the JWT auto token.
	const secretLength = 32

	// initial branding
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding slogo image in base64 string format.",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial JWT token
	secret, err := common.RandomString(secretLength)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "failed to generate random JWT secret")
	}
	authSetting, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingAuthSecret,
		Value:       secret,
		Description: "Random string used to sign the JWT auth token.",
	}, api.SystemBotID)
	if err != nil {
		return "", "", 0, err
	}
	// Set secret to the stored secret.
	secret = authSetting.Value

	// initial workspace
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingWorkspaceID,
		Value:       uuid.New().String(),
		Description: "The workspace identifier",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial license
	if _, _, err = datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingEnterpriseLicense,
		Value:       "",
		Description: "Enterprise license",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial feishu app
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingAppIM,
		Value:       "{}",
		Description: "",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial watermark setting
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingWatermark,
		Value:       "0",
		Description: "Display watermark",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial OpenAI key setting
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingPluginOpenAIKey,
		Value:       "",
		Description: "API key to request OpenAI (ChatGPT)",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingPluginOpenAIEndpoint,
		Value:       "",
		Description: "API Endpoint for OpenAI",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial external approval setting
	externalApprovalSettingValue, err := protojson.Marshal(&storepb.ExternalApprovalSetting{})
	if err != nil {
		return "", "", 0, errors.Wrap(err, "failed to marshal initial external approval setting")
	}
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingWorkspaceExternalApproval,
		Value:       string(externalApprovalSettingValue),
		Description: "The external approval setting",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial schema template setting
	schemaTemplateSettingValue, err := protojson.Marshal(&storepb.SchemaTemplateSetting{})
	if err != nil {
		return "", "", 0, errors.Wrap(err, "failed to marshal initial schema template setting")
	}
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingSchemaTemplate,
		Value:       string(schemaTemplateSettingValue),
		Description: "The schema template setting",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial data classification setting
	dataClassificationSettingValue, err := protojson.Marshal(&storepb.DataClassificationSetting{})
	if err != nil {
		return "", "", 0, errors.Wrap(err, "failed to marshal initial data classification setting")
	}
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name:        api.SettingDataClassification,
		Value:       string(dataClassificationSettingValue),
		Description: "The data classification setting",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial workspace approval setting
	approvalSettingValue, err := protojson.Marshal(&storepb.WorkspaceApprovalSetting{})
	if err != nil {
		return "", "", 0, errors.Wrap(err, "failed to marshal initial workspace approval setting")
	}
	if _, _, err := datastore.CreateSettingIfNotExistV2(ctx, &store.SettingMessage{
		Name: api.SettingWorkspaceApproval,
		// Value is ""
		Value:       string(approvalSettingValue),
		Description: "The workspace approval setting",
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// initial workspace profile setting
	settingName := api.SettingWorkspaceProfile
	workspaceProfileSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name:    &settingName,
		Enforce: true,
	})
	if err != nil {
		return "", "", 0, err
	}

	workspaceProfilePayload := &storepb.WorkspaceProfileSetting{
		ExternalUrl: s.profile.ExternalURL,
	}
	if workspaceProfileSetting != nil {
		workspaceProfilePayload = new(storepb.WorkspaceProfileSetting)
		if err := protojson.Unmarshal([]byte(workspaceProfileSetting.Value), workspaceProfilePayload); err != nil {
			return "", "", 0, err
		}
		if s.profile.ExternalURL != "" {
			workspaceProfilePayload.ExternalUrl = s.profile.ExternalURL
		}
	}

	bytes, err := protojson.Marshal(workspaceProfilePayload)
	if err != nil {
		return "", "", 0, err
	}

	if _, err := datastore.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  api.SettingWorkspaceProfile,
		Value: string(bytes),
	}, api.SystemBotID); err != nil {
		return "", "", 0, err
	}

	// Get token duration and external URL.
	workspaceProfileSettingName := api.SettingWorkspaceProfile
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &workspaceProfileSettingName})
	if err != nil {
		return "", "", 0, err
	}
	externalURL := ""
	tokenDuration := auth.DefaultTokenDuration
	if setting != nil {
		settingValue := new(storepb.WorkspaceProfileSetting)
		if err := protojson.Unmarshal([]byte(setting.Value), settingValue); err != nil {
			return "", "", 0, err
		}
		if settingValue.ExternalUrl != "" {
			externalURL = settingValue.ExternalUrl
		}
		if settingValue.TokenDuration != nil && settingValue.TokenDuration.Seconds > 0 {
			tokenDuration = settingValue.TokenDuration.AsDuration()
		}
	}

	return secret, externalURL, tokenDuration, nil
}

// initMetricReporter will initial the metric scheduler.
func (s *Server) initMetricReporter() {
	metricReporter := metricreport.NewReporter(s.store, s.licenseService, &s.profile, s.profile.EnableMetric)
	metricReporter.Register(metric.InstanceCountMetricName, metriccollector.NewInstanceCountCollector(s.store))
	metricReporter.Register(metric.IssueCountMetricName, metriccollector.NewIssueCountCollector(s.store))
	metricReporter.Register(metric.ProjectCountMetricName, metriccollector.NewProjectCountCollector(s.store))
	metricReporter.Register(metric.PolicyCountMetricName, metriccollector.NewPolicyCountCollector(s.store))
	metricReporter.Register(metric.TaskCountMetricName, metriccollector.NewTaskCountCollector(s.store))
	metricReporter.Register(metric.DatabaseCountMetricName, metriccollector.NewDatabaseCountCollector(s.store))
	metricReporter.Register(metric.SheetCountMetricName, metriccollector.NewSheetCountCollector(s.store))
	metricReporter.Register(metric.MemberCountMetricName, metriccollector.NewMemberCountCollector(s.store))
	s.metricReporter = metricReporter
}
