package v1

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ActuatorService implements the actuator service.
type ActuatorService struct {
	v1pb.UnimplementedActuatorServiceServer
	store           *store.Store
	profile         *config.Profile
	errorRecordRing *api.ErrorRecordRing
	licenseService  enterprise.LicenseService
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(store *store.Store, profile *config.Profile, errorRecordRing *api.ErrorRecordRing, licenseService enterprise.LicenseService) *ActuatorService {
	return &ActuatorService{
		store:           store,
		profile:         profile,
		errorRecordRing: errorRecordRing,
		licenseService:  licenseService,
	}
}

// GetActuatorInfo gets the actuator info.
func (s *ActuatorService) GetActuatorInfo(ctx context.Context, _ *v1pb.GetActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	return s.getServerInfo(ctx)
}

// UpdateActuatorInfo updates the actuator info.
func (s *ActuatorService) UpdateActuatorInfo(ctx context.Context, request *v1pb.UpdateActuatorInfoRequest) (*v1pb.ActuatorInfo, error) {
	for _, path := range request.UpdateMask.Paths {
		if path == "debug" {
			level := slog.LevelInfo
			if request.Actuator.Debug {
				level = slog.LevelDebug
			}
			log.LogLevel.Set(level)
		}
	}

	return s.getServerInfo(ctx)
}

// ListDebugLog lists the debug log.
func (s *ActuatorService) ListDebugLog(_ context.Context, _ *v1pb.ListDebugLogRequest) (*v1pb.ListDebugLogResponse, error) {
	resp := &v1pb.ListDebugLogResponse{}

	s.errorRecordRing.RWMutex.RLock()
	defer s.errorRecordRing.RWMutex.RUnlock()

	s.errorRecordRing.Ring.Do(func(p any) {
		if p == nil {
			return
		}
		errRecord, ok := p.(*v1pb.DebugLog)
		if !ok {
			return
		}
		resp.Logs = append(resp.Logs, errRecord)
	})

	return resp, nil
}

// DeleteCache deletes the cache.
func (s *ActuatorService) DeleteCache(_ context.Context, _ *v1pb.DeleteCacheRequest) (*emptypb.Empty, error) {
	s.store.DeleteCache()
	return &emptypb.Empty{}, nil
}

// GetResourcePackage gets the theme resources.
func (s *ActuatorService) GetResourcePackage(ctx context.Context, _ *v1pb.GetResourcePackageRequest) (*v1pb.ResourcePackage, error) {
	settingName := api.SettingBrandingLogo
	brandingSetting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace branding: %v", err)
	}
	if brandingSetting == nil {
		return nil, errors.Errorf("cannot find setting %v", settingName)
	}

	return &v1pb.ResourcePackage{
		Logo: []byte(brandingSetting.Value),
	}, nil
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	count, err := s.store.CountUsers(ctx, api.EndUser)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting: %v", err)
	}

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	usedFeatures, err := s.getUsedFeatures(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get used features, error: %v", err)
	}
	unlicensedFeatures := s.getUnlicensedFeatures(usedFeatures)
	var unlicensedFeaturesString []string
	for _, f := range unlicensedFeatures {
		unlicensedFeaturesString = append(unlicensedFeaturesString, f.String())
	}

	serverInfo := v1pb.ActuatorInfo{
		Version:            s.profile.Version,
		GitCommit:          s.profile.GitCommit,
		Readonly:           s.profile.Readonly,
		Saas:               s.profile.SaaS,
		DemoName:           s.profile.DemoName,
		NeedAdminSetup:     count == 0,
		ExternalUrl:        setting.ExternalUrl,
		DisallowSignup:     setting.DisallowSignup,
		Require_2Fa:        setting.Require_2Fa,
		LastActiveTime:     timestamppb.New(time.Unix(s.profile.LastActiveTs, 0)),
		WorkspaceId:        workspaceID,
		GitopsWebhookUrl:   setting.GitopsWebhookUrl,
		Debug:              slog.Default().Enabled(ctx, slog.LevelDebug),
		Lsp:                s.profile.Lsp,
		PreUpdateBackup:    s.profile.PreUpdateBackup,
		IamGuard:           s.profile.DevelopmentIAM,
		UnlicensedFeatures: unlicensedFeaturesString,
	}

	return &serverInfo, nil
}

func (s *ActuatorService) getUsedFeatures(ctx context.Context) ([]api.FeatureType, error) {
	var features []api.FeatureType

	// idp
	idps, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list identity providers")
	}
	if len(idps) > 0 {
		features = append(features, api.FeatureSSO)
	}

	// setting
	settingName := api.SettingBrandingLogo
	brandingLogo, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &settingName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get branding logo setting")
	}
	if brandingLogo != nil && brandingLogo.Value != "" {
		features = append(features, api.FeatureBranding)
	}

	settingName = api.SettingWatermark
	watermark, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &settingName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get watermark setting")
	}
	if watermark != nil && watermark.Value == "1" {
		features = append(features, api.FeatureWatermark)
	}

	settingName = api.SettingPluginOpenAIKey
	openAIKey, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{Name: &settingName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get openai key setting")
	}
	if openAIKey != nil && openAIKey.Value != "" {
		features = append(features, api.FeaturePluginOpenAI)
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace general setting")
	}
	if setting.DisallowSignup && !s.profile.SaaS {
		features = append(features, api.FeatureDisallowSignup)
	}
	if setting.Require_2Fa {
		features = append(features, api.Feature2FA)
	}
	if setting.GetTokenDuration().GetSeconds() > 0 && float64(setting.GetTokenDuration().GetSeconds()) != auth.DefaultTokenDuration.Seconds() {
		features = append(features, api.FeatureSecureToken)
	}
	if setting.GetAnnouncement().GetText() != "" {
		features = append(features, api.FeatureAnnouncement)
	}

	// environment tier
	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list environments")
	}
	for _, env := range environments {
		if env.Protected {
			features = append(features, api.FeatureEnvironmentTierPolicy)
			break
		}
	}

	// database group
	databaseGroups, err := s.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list database groups")
	}
	if len(databaseGroups) > 0 {
		features = append(features, api.FeatureDatabaseGrouping)
	}
	return features, nil
}

func (s *ActuatorService) getUnlicensedFeatures(features []api.FeatureType) []api.FeatureType {
	var unlicensedFeatures []api.FeatureType
	for _, feature := range features {
		if err := s.licenseService.IsFeatureEnabled(feature); err != nil {
			unlicensedFeatures = append(unlicensedFeatures, feature)
		}
	}
	return unlicensedFeatures
}
