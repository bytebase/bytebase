package v1

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// ActuatorService implements the Connect RPC interface for ActuatorService.
type ActuatorService struct {
	v1connect.UnimplementedActuatorServiceHandler
	store                 *store.Store
	profile               *config.Profile
	licenseService        *enterprise.LicenseService
	schemaSyncer          *schemasync.Syncer
	sampleInstanceManager *sampleinstance.Manager
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(
	store *store.Store,
	profile *config.Profile,
	schemaSyncer *schemasync.Syncer,
	licenseService *enterprise.LicenseService,
	sampleInstanceManager *sampleinstance.Manager,
) *ActuatorService {
	return &ActuatorService{
		store:                 store,
		profile:               profile,
		licenseService:        licenseService,
		schemaSyncer:          schemaSyncer,
		sampleInstanceManager: sampleInstanceManager,
	}
}

// GetActuatorInfo gets the actuator info.
func (s *ActuatorService) GetActuatorInfo(
	ctx context.Context,
	_ *connect.Request[v1pb.GetActuatorInfoRequest],
) (*connect.Response[v1pb.ActuatorInfo], error) {
	info, err := s.getServerInfo(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// UpdateActuatorInfo updates the actuator info.
func (s *ActuatorService) UpdateActuatorInfo(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateActuatorInfoRequest],
) (*connect.Response[v1pb.ActuatorInfo], error) {
	request := req.Msg
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

	info, err := s.getServerInfo(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// DeleteCache deletes the cache.
func (s *ActuatorService) DeleteCache(
	ctx context.Context,
	_ *connect.Request[v1pb.DeleteCacheRequest],
) (*connect.Response[emptypb.Empty], error) {
	s.store.DeleteCache()
	s.licenseService.RefreshCache(ctx)
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetResourcePackage gets the theme resources.
func (s *ActuatorService) GetResourcePackage(
	ctx context.Context,
	_ *connect.Request[v1pb.GetResourcePackageRequest],
) (*connect.Response[v1pb.ResourcePackage], error) {
	brandingSetting, err := s.store.GetSettingV2(ctx, storepb.SettingName_BRANDING_LOGO)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace branding"))
	}
	if brandingSetting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_BRANDING_LOGO)
	}

	pkg := &v1pb.ResourcePackage{
		Logo: []byte(brandingSetting.Value),
	}
	return connect.NewResponse(pkg), nil
}

// SetupSample sets up the sample project and instance.
func (s *ActuatorService) SetupSample(
	ctx context.Context,
	_ *connect.Request[v1pb.SetupSampleRequest],
) (*connect.Response[emptypb.Empty], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}
	if err := s.sampleInstanceManager.GenerateOnboardingData(ctx, user.ID, s.schemaSyncer); err != nil {
		// When running inside docker on mac, we sometimes get database does not exist error.
		// This is due to the docker overlay storage incompatibility with mac OS file system.
		// Onboarding error is not critical, so we just emit an error log.
		slog.Error("failed to prepare onboarding data", log.BBError(err))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	count, err := s.store.CountUsers(ctx, storepb.PrincipalType_END_USER)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
	}

	passwordRestrictionSetting, err := s.store.GetPasswordRestrictionSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find password restriction setting"))
	}
	passwordSetting := convertToPasswordRestrictionSetting(passwordRestrictionSetting)

	workspaceID, err := s.store.GetWorkspaceID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	usedFeatures, err := s.getUsedFeatures(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get used features"))
	}
	unlicensedFeatures := s.getUnlicensedFeatures(usedFeatures)
	var unlicensedFeaturesString []string
	for _, f := range unlicensedFeatures {
		unlicensedFeaturesString = append(unlicensedFeaturesString, f.String())
	}

	// Check if sample instances are available
	hasSampleInstances, _ := s.store.HasSampleInstances(ctx)

	serverInfo := v1pb.ActuatorInfo{
		Version:                s.profile.Version,
		GitCommit:              s.profile.GitCommit,
		Saas:                   s.profile.SaaS,
		Demo:                   s.profile.Demo,
		NeedAdminSetup:         count == 0,
		ExternalUrl:            setting.ExternalUrl,
		DisallowSignup:         setting.DisallowSignup || s.profile.SaaS,
		Require_2Fa:            setting.Require_2Fa,
		LastActiveTime:         timestamppb.New(time.Unix(s.profile.LastActiveTS.Load(), 0)),
		WorkspaceId:            workspaceID,
		Debug:                  s.profile.RuntimeDebug.Load(),
		Docker:                 s.profile.IsDocker,
		UnlicensedFeatures:     unlicensedFeaturesString,
		DisallowPasswordSignin: setting.DisallowPasswordSignin,
		PasswordRestriction:    passwordSetting,
		EnableSample:           hasSampleInstances,
	}

	stats, err := s.store.StatUsers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to stat users"))
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
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count activated instance"))
	}
	serverInfo.ActivatedInstanceCount = int32(activatedInstanceCount)

	totalInstanceCount, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count total instance"))
	}
	serverInfo.TotalInstanceCount = int32(totalInstanceCount)

	return &serverInfo, nil
}

func (s *ActuatorService) getUsedFeatures(ctx context.Context) ([]v1pb.PlanFeature, error) {
	var features []v1pb.PlanFeature

	// idp
	idps, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list identity providers")
	}
	// TODO(d): use fine-grained feature control for SSO.
	if len(idps) > 0 {
		features = append(features, v1pb.PlanFeature_FEATURE_ENTERPRISE_SSO)
	}

	// setting
	brandingLogo, err := s.store.GetSettingV2(ctx, storepb.SettingName_BRANDING_LOGO)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get branding logo setting")
	}
	if brandingLogo != nil && brandingLogo.Value != "" {
		features = append(features, v1pb.PlanFeature_FEATURE_CUSTOM_LOGO)
	}

	watermark, err := s.store.GetSettingV2(ctx, storepb.SettingName_WATERMARK)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get watermark setting")
	}
	if watermark != nil && watermark.Value == "1" {
		features = append(features, v1pb.PlanFeature_FEATURE_WATERMARK)
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace general setting")
	}
	if setting.DisallowSignup && !s.profile.SaaS {
		features = append(features, v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP)
	}
	if setting.Require_2Fa {
		features = append(features, v1pb.PlanFeature_FEATURE_TWO_FA)
	}
	if setting.GetTokenDuration().GetSeconds() > 0 && float64(setting.GetTokenDuration().GetSeconds()) != auth.DefaultTokenDuration.Seconds() {
		features = append(features, v1pb.PlanFeature_FEATURE_SIGN_IN_FREQUENCY_CONTROL)
	}
	if setting.GetAnnouncement().GetText() != "" {
		features = append(features, v1pb.PlanFeature_FEATURE_DASHBOARD_ANNOUNCEMENT)
	}

	// environment tier
	environments, err := s.store.GetEnvironmentSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment setting")
	}
	for _, env := range environments.GetEnvironments() {
		if v, ok := env.Tags["protected"]; ok && v == "protected" {
			features = append(features, v1pb.PlanFeature_FEATURE_ENVIRONMENT_TIERS)
			break
		}
	}

	// database group
	databaseGroups, err := s.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list database groups")
	}
	if len(databaseGroups) > 0 {
		features = append(features, v1pb.PlanFeature_FEATURE_DATABASE_GROUPS)
	}
	return features, nil
}

func (s *ActuatorService) getUnlicensedFeatures(features []v1pb.PlanFeature) []v1pb.PlanFeature {
	var unlicensedFeatures []v1pb.PlanFeature
	for _, feature := range features {
		if err := s.licenseService.IsFeatureEnabled(feature); err != nil {
			unlicensedFeatures = append(unlicensedFeatures, feature)
		}
	}
	return unlicensedFeatures
}
