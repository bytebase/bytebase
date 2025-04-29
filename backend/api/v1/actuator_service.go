package v1

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ActuatorService implements the actuator service.
type ActuatorService struct {
	v1pb.UnimplementedActuatorServiceServer
	store          *store.Store
	profile        *config.Profile
	licenseService enterprise.LicenseService
	schemaSyncer   *schemasync.Syncer
}

// NewActuatorService creates a new ActuatorService.
func NewActuatorService(
	store *store.Store,
	profile *config.Profile,
	schemaSyncer *schemasync.Syncer,
	licenseService enterprise.LicenseService,
) *ActuatorService {
	return &ActuatorService{
		store:          store,
		profile:        profile,
		licenseService: licenseService,
		schemaSyncer:   schemaSyncer,
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
	brandingSetting, err := s.store.GetSettingV2(ctx, base.SettingBrandingLogo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace branding: %v", err)
	}
	if brandingSetting == nil {
		return nil, errors.Errorf("cannot find setting %v", base.SettingBrandingLogo)
	}

	return &v1pb.ResourcePackage{
		Logo: []byte(brandingSetting.Value),
	}, nil
}

// SetupSample set up the sample project and instance.
func (s *ActuatorService) SetupSample(ctx context.Context, _ *v1pb.SetupSampleRequest) (*emptypb.Empty, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok || user == nil {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	if s.profile.SampleDatabasePort != 0 {
		if err := s.generateOnboardingData(ctx, user); err != nil {
			// When running inside docker on mac, we sometimes get database does not exist error.
			// This is due to the docker overlay storage incompatibility with mac OS file system.
			// Onboarding error is not critical, so we just emit an error log.
			slog.Error("failed to prepare onboarding data", log.BBError(err))
		}
	}
	return &emptypb.Empty{}, nil
}

// generateOnboardingData generates onboarding data including project and instance.
func (s *ActuatorService) generateOnboardingData(ctx context.Context, user *store.UserMessage) error {
	userID := user.ID
	setting := &storepb.Project{
		AllowModifyStatement: true,
		AutoResolveIssue:     true,
	}

	projectID := "project-sample"
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding project %v", projectID)
	}
	if project == nil {
		sampleProject, err := s.store.CreateProjectV2(ctx, &store.ProjectMessage{
			ResourceID: "project-sample",
			Title:      "Sample Project",
			Setting:    setting,
		}, userID)
		if err != nil {
			return errors.Wrapf(err, "failed to create onboarding project")
		}
		project = sampleProject
	}

	instanceMessages := []*store.InstanceMessage{
		{
			ResourceID:    "test-sample-instance",
			EnvironmentID: base.DefaultTestEnvironmentID,
			Metadata: &storepb.Instance{
				Title: "Test Sample Instance",
				DataSources: []*storepb.DataSource{
					{
						Port:     strconv.Itoa(s.profile.SampleDatabasePort),
						Database: postgres.SampleDatabaseTest,
					},
				},
			},
		},
		{
			ResourceID:    "prod-sample-instance",
			EnvironmentID: base.DefaultProdEnvironmentID,
			Metadata: &storepb.Instance{
				Title: "Prod Sample Instance",
				DataSources: []*storepb.DataSource{
					{
						Port:     strconv.Itoa(s.profile.SampleDatabasePort + 1),
						Database: postgres.SampleDatabaseProd,
					},
				},
			},
		},
	}
	for _, instanceMessage := range instanceMessages {
		if err := s.generateInstance(ctx, project.ResourceID, instanceMessage); err != nil {
			slog.Error("failed to prepare onboarding instance", log.BBError(err), slog.String("instance", instanceMessage.ResourceID))
		}
	}

	return nil
}

func (s *ActuatorService) generateInstance(
	ctx context.Context,
	projectID string,
	instanceMessage *store.InstanceMessage,
) error {
	// Generate Sample Instance
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceMessage.ResourceID,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding instance %v", instanceMessage.ResourceID)
	}
	if instance == nil {
		sampleInstance, err := s.store.CreateInstanceV2(ctx, &store.InstanceMessage{
			ResourceID:    instanceMessage.ResourceID,
			EnvironmentID: instanceMessage.EnvironmentID,
			Metadata: &storepb.Instance{
				Title:        instanceMessage.Metadata.Title,
				Engine:       storepb.Engine_POSTGRES,
				ExternalLink: "",
				Activation:   false,
				DataSources: []*storepb.DataSource{
					{
						Id:       "admin",
						Type:     storepb.DataSourceType_ADMIN,
						Username: postgres.SampleUser,
						Password: "",
						Host:     common.GetPostgresSocketDir(),
						Port:     instanceMessage.Metadata.DataSources[0].Port,
						Database: instanceMessage.Metadata.DataSources[0].Database,
					},
				},
			},
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create onboarding instance %v", instanceMessage.ResourceID)
		}
		instance = sampleInstance
	}

	// Sync the instance schema so we can transfer the sample database later.
	if _, _, _, err := s.schemaSyncer.SyncInstance(ctx, instance); err != nil {
		return errors.Wrapf(err, "failed to sync onboarding instance %v", instance.ResourceID)
	}

	dbName := instanceMessage.Metadata.DataSources[0].Database
	// Transfer sample database to the just created project.
	transferDatabaseMessage := &store.UpdateDatabaseMessage{
		InstanceID:   instance.ResourceID,
		DatabaseName: dbName,
		ProjectID:    &projectID,
	}
	_, err = s.store.UpdateDatabase(ctx, transferDatabaseMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to transfer sample database %v", dbName)
	}

	testDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instance.ResourceID,
		DatabaseName:    &dbName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to find onboarding database %v", dbName)
	}
	if testDatabase == nil {
		return errors.Errorf("database %q not found", dbName)
	}

	// Need to sync database schema so we can configure sensitive data policy and create the schema
	// update issue later.
	if err := s.schemaSyncer.SyncDatabaseSchema(ctx, testDatabase); err != nil {
		return errors.Wrapf(err, "failed to sync sample database schema %v", dbName)
	}
	return nil
}

func (s *ActuatorService) getServerInfo(ctx context.Context) (*v1pb.ActuatorInfo, error) {
	count, err := s.store.CountUsers(ctx, base.EndUser)
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
		EnableSample:           s.profile.SampleDatabasePort != 0,
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

func (s *ActuatorService) getUsedFeatures(ctx context.Context) ([]base.FeatureType, error) {
	var features []base.FeatureType

	// idp
	idps, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list identity providers")
	}
	if len(idps) > 0 {
		features = append(features, base.FeatureSSO)
	}

	// setting
	brandingLogo, err := s.store.GetSettingV2(ctx, base.SettingBrandingLogo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get branding logo setting")
	}
	if brandingLogo != nil && brandingLogo.Value != "" {
		features = append(features, base.FeatureBranding)
	}

	watermark, err := s.store.GetSettingV2(ctx, base.SettingWatermark)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get watermark setting")
	}
	if watermark != nil && watermark.Value == "1" {
		features = append(features, base.FeatureWatermark)
	}

	aiSetting, err := s.store.GetAISetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ai setting")
	}
	if aiSetting.ApiKey != "" {
		features = append(features, base.FeatureAIAssistant)
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace general setting")
	}
	if setting.DisallowSignup && !s.profile.SaaS {
		features = append(features, base.FeatureDisallowSignup)
	}
	if setting.Require_2Fa {
		features = append(features, base.Feature2FA)
	}
	if setting.GetTokenDuration().GetSeconds() > 0 && float64(setting.GetTokenDuration().GetSeconds()) != auth.DefaultTokenDuration.Seconds() {
		features = append(features, base.FeatureSecureToken)
	}
	if setting.GetAnnouncement().GetText() != "" {
		features = append(features, base.FeatureAnnouncement)
	}

	// environment tier
	environments, err := s.store.GetEnvironmentSetting(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment setting")
	}
	for _, env := range environments.GetEnvironments() {
		if v, ok := env.Tags["protected"]; ok && v == "protected" {
			features = append(features, base.FeatureEnvironmentTierPolicy)
			break
		}
	}

	// database group
	databaseGroups, err := s.store.ListDatabaseGroups(ctx, &store.FindDatabaseGroupMessage{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list database groups")
	}
	if len(databaseGroups) > 0 {
		features = append(features, base.FeatureDatabaseGrouping)
	}
	return features, nil
}

func (s *ActuatorService) getUnlicensedFeatures(features []base.FeatureType) []base.FeatureType {
	var unlicensedFeatures []base.FeatureType
	for _, feature := range features {
		if err := s.licenseService.IsFeatureEnabled(feature); err != nil {
			unlicensedFeatures = append(unlicensedFeatures, feature)
		}
	}
	return unlicensedFeatures
}
