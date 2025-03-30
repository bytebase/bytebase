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
	store          *store.Store
	profile        *config.Profile
	licenseService enterprise.LicenseService
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(store *store.Store, profile *config.Profile, licenseService enterprise.LicenseService) *ActuatorService {
	return &ActuatorService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
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
			debug := request.GetActuator().GetDebug()

			s.profile.RuntimeDebug.Store(debug)
			level := slog.LevelInfo
			if debug {
				level = slog.LevelDebug
			}
			log.LogLevel.Set(level)
		}
	}

	return s.getServerInfo(ctx)
}

// DeleteCache deletes the cache.
func (s *ActuatorService) DeleteCache(_ context.Context, _ *v1pb.DeleteCacheRequest) (*emptypb.Empty, error) {
	s.store.DeleteCache()
	return &emptypb.Empty{}, nil
}

// GetResourcePackage gets the theme resources.
func (s *ActuatorService) GetResourcePackage(ctx context.Context, _ *v1pb.GetResourcePackageRequest) (*v1pb.ResourcePackage, error) {
	brandingSetting, err := s.store.GetSettingV2(ctx, api.SettingBrandingLogo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace branding: %v", err)
	}
	if brandingSetting == nil {
		return nil, errors.Errorf("cannot find setting %v", api.SettingBrandingLogo)
	}

	return &v1pb.ResourcePackage{
		Logo: []byte(brandingSetting.Value),
	}, nil
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	count, err := s.store.CountUsers(ctx, api.EndUser)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting: %v", err)
	}

	passwordRestrictionSetting, err := s.store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find password restriction setting: %v", err)
	}
	passwordSetting := new(v1pb.PasswordRestrictionSetting)
	if err := convertProtoToProto(passwordRestrictionSetting, passwordSetting); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal password restriction setting with error: %v", err)
	}

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		Version:                s.profile.Version,
		GitCommit:              s.profile.GitCommit,
		Saas:                   s.profile.SaaS,
		Demo:                   s.profile.Demo,
		NeedAdminSetup:         count == 0,
		ExternalUrl:            setting.ExternalUrl,
		DisallowSignup:         setting.DisallowSignup,
		Require_2Fa:            setting.Require_2Fa,
		LastActiveTime:         timestamppb.New(time.Unix(s.profile.LastActiveTS, 0)),
		WorkspaceId:            workspaceID,
		Debug:                  s.profile.RuntimeDebug.Load(),
		Docker:                 s.profile.IsDocker,
		UnlicensedFeatures:     unlicensedFeaturesString,
		DisallowPasswordSignin: setting.DisallowPasswordSignin,
		PasswordRestriction:    passwordSetting,
	}

	stats, err := s.store.StatUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to stat users, error: %v", err)
	}
	for _, stat := range stats {
		serverInfo.UserStats = append(serverInfo.UserStats, &v1pb.ActuatorInfo_StatUser{
			State:    convertDeletedToState(stat.Deleted),
			UserType: convertToV1UserType(stat.Type),
			Count:    int32(stat.Count),
		})
	}

	activatedInstanceCount, err := s.store.GetActivatedInstanceCount(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count activated instance, error: %v", err)
	}
	serverInfo.ActivatedInstanceCount = int32(activatedInstanceCount)

	totalInstanceCount, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count total instance, error: %v", err)
	}
	serverInfo.TotalInstanceCount = int32(totalInstanceCount)

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
	brandingLogo, err := s.store.GetSettingV2(ctx, api.SettingBrandingLogo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get branding logo setting")
	}
	if brandingLogo != nil && brandingLogo.Value != "" {
		features = append(features, api.FeatureBranding)
	}

	watermark, err := s.store.GetSettingV2(ctx, api.SettingWatermark)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get watermark setting")
	}
	if watermark != nil && watermark.Value == "1" {
		features = append(features, api.FeatureWatermark)
	}

	aiSetting, err := s.store.GetAISetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ai setting")
	}
	if aiSetting.ApiKey != "" {
		features = append(features, api.FeatureAIAssistant)
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
