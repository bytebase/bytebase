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
	"github.com/bytebase/bytebase/backend/common"
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
	var workspaceID string
	if !s.profile.SaaS {
		ws, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		workspaceID = ws
	}
	info, err := s.getServerInfo(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// GetWorkspaceActuatorInfo gets workspace-scoped actuator info. Requires authentication.
// Workspace validation is handled by the ACL layer (resource_reference on name field).
func (s *ActuatorService) GetWorkspaceActuatorInfo(
	ctx context.Context,
	_ *connect.Request[v1pb.GetWorkspaceActuatorInfoRequest],
) (*connect.Response[v1pb.ActuatorInfo], error) {
	info, err := s.getServerInfo(ctx, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(info), nil
}

// DeleteCache deletes the cache.
func (s *ActuatorService) DeleteCache(
	_ context.Context,
	_ *connect.Request[v1pb.DeleteCacheRequest],
) (*connect.Response[emptypb.Empty], error) {
	s.store.DeleteCache()
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetResourcePackage gets the theme resources.
// Serves both /v1/actuator/resources and /v1/{name=workspaces/*}/actuator/resources.
func (s *ActuatorService) GetResourcePackage(
	ctx context.Context,
	req *connect.Request[v1pb.GetResourcePackageRequest],
) (*connect.Response[v1pb.ResourcePackage], error) {
	var workspaceID string
	if !s.profile.SaaS {
		ws, err := s.store.GetWorkspaceID(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		workspaceID = ws
	} else if req.Msg.Name != "" {
		reqWorkspaceID, err := common.GetWorkspaceID(req.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		workspaceID = reqWorkspaceID
	}

	pkg := &v1pb.ResourcePackage{}

	if workspaceID != "" {
		workspaceProfileSetting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace profile setting"))
		}
		pkg.Logo = []byte(workspaceProfileSetting.BrandingLogo)
	}

	return connect.NewResponse(pkg), nil
}

// SetupSample sets up the sample project and instance.
func (s *ActuatorService) SetupSample(
	ctx context.Context,
	_ *connect.Request[v1pb.SetupSampleRequest],
) (*connect.Response[emptypb.Empty], error) {
	if s.profile.SaaS {
		// skip sample setup in SaaS
		slog.Debug("sample is not available for SaaS")
		return connect.NewResponse(&emptypb.Empty{}), nil
	}
	user, ok := GetUserFromContext(ctx)
	if !ok || user == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
	}

	if s.sampleInstanceManager != nil {
		if err := s.sampleInstanceManager.GenerateOnboardingData(ctx, common.GetWorkspaceIDFromContext(ctx), user, s.schemaSyncer); err != nil {
			// When running inside docker on mac, we sometimes get database does not exist error.
			// This is due to the docker overlay storage incompatibility with mac OS file system.
			// Onboarding error is not critical, so we just emit an error log.
			slog.Error("failed to prepare onboarding data", log.BBError(err))
		}
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ActuatorService) getServerInfo(ctx context.Context, workspaceID string) (*v1pb.ActuatorInfo, error) {
	restriction, err := getAccountRestriction(
		ctx,
		s.store,
		s.licenseService,
		s.profile.SaaS,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}

	serverInfo := v1pb.ActuatorInfo{
		Version:             s.profile.Version,
		GitCommit:           s.profile.GitCommit,
		Saas:                s.profile.SaaS,
		Demo:                s.profile.Demo,
		LastActiveTime:      timestamppb.New(time.Unix(s.profile.LastActiveTS.Load(), 0)),
		Docker:              s.profile.IsDocker,
		ExternalUrlFromFlag: s.profile.ExternalURL != "",
		ReplicaCount:        int32(s.licenseService.CountActiveReplicas(ctx)),
		Restriction:         restriction,
	}

	if workspaceID != "" {
		serverInfo.Workspace = common.FormatWorkspace(workspaceID)

		defaultProjectID, err := s.store.GetDefaultProjectID(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get default project"))
		}
		serverInfo.DefaultProject = common.FormatProject(defaultProjectID)

		usedFeatures, err := s.getUsedFeatures(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get used features"))
		}
		unlicensedFeatures := s.getUnlicensedFeatures(ctx, workspaceID, usedFeatures)
		var unlicensedFeaturesString []string
		for _, f := range unlicensedFeatures {
			unlicensedFeaturesString = append(unlicensedFeaturesString, f.String())
		}
		serverInfo.UnlicensedFeatures = unlicensedFeaturesString

		activeEndUserCount, err := s.store.CountActiveEndUsersPerWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		serverInfo.ActivatedUserCount = int32(activeEndUserCount)

		// Check if sample instances are available
		hasSampleInstances, _ := s.store.HasSampleInstances(ctx, workspaceID)
		serverInfo.EnableSample = hasSampleInstances

		setting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find workspace setting"))
		}
		// Prefer command-line flag over database value for external URL
		externalURL := setting.ExternalUrl
		if s.profile.ExternalURL != "" {
			externalURL = s.profile.ExternalURL
		}
		serverInfo.ExternalUrl = externalURL

		activatedInstanceCount, err := s.store.GetActivatedInstanceCount(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count activated instance"))
		}
		serverInfo.ActivatedInstanceCount = int32(activatedInstanceCount)

		activeInstanceCount, err := s.store.CountActiveInstances(ctx, workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to count total instance"))
		}
		serverInfo.TotalInstanceCount = int32(activeInstanceCount)
	}

	return &serverInfo, nil
}

func (s *ActuatorService) getUsedFeatures(ctx context.Context, workspaceID string) ([]v1pb.PlanFeature, error) {
	var features []v1pb.PlanFeature

	// idp
	idps, err := s.store.ListIdentityProviders(ctx, &store.FindIdentityProviderMessage{Workspace: &workspaceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list identity providers")
	}
	// TODO(d): use fine-grained feature control for SSO.
	if len(idps) > 0 {
		features = append(features, v1pb.PlanFeature_FEATURE_ENTERPRISE_SSO)
	}

	setting, err := s.store.GetWorkspaceProfileSetting(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get workspace general setting")
	}
	if setting.DisallowSignup {
		features = append(features, v1pb.PlanFeature_FEATURE_DISALLOW_SELF_SERVICE_SIGNUP)
	}
	if setting.Require_2Fa {
		features = append(features, v1pb.PlanFeature_FEATURE_TWO_FA)
	}
	if setting.GetRefreshTokenDuration().GetSeconds() > 0 && float64(setting.GetRefreshTokenDuration().GetSeconds()) != auth.DefaultRefreshTokenDuration.Seconds() {
		features = append(features, v1pb.PlanFeature_FEATURE_TOKEN_DURATION_CONTROL)
	}
	if setting.GetAnnouncement().GetText() != "" {
		features = append(features, v1pb.PlanFeature_FEATURE_DASHBOARD_ANNOUNCEMENT)
	}
	if setting.Watermark {
		features = append(features, v1pb.PlanFeature_FEATURE_WATERMARK)
	}
	if setting.BrandingLogo != "" {
		features = append(features, v1pb.PlanFeature_FEATURE_CUSTOM_LOGO)
	}

	// environment tier
	environments, err := s.store.GetEnvironment(ctx, workspaceID)
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

func (s *ActuatorService) getUnlicensedFeatures(ctx context.Context, workspaceID string, features []v1pb.PlanFeature) []v1pb.PlanFeature {
	var unlicensedFeatures []v1pb.PlanFeature
	for _, feature := range features {
		if err := s.licenseService.IsFeatureEnabled(ctx, workspaceID, feature); err != nil {
			unlicensedFeatures = append(unlicensedFeatures, feature)
		}
	}
	return unlicensedFeatures
}

// convertToV1PasswordRestriction converts store PasswordRestriction to v1 PasswordRestriction.
func convertToV1PasswordRestriction(pr *storepb.WorkspaceProfileSetting_PasswordRestriction) *v1pb.WorkspaceProfileSetting_PasswordRestriction {
	if pr == nil {
		return nil
	}
	return &v1pb.WorkspaceProfileSetting_PasswordRestriction{
		MinLength:                         pr.MinLength,
		RequireNumber:                     pr.RequireNumber,
		RequireLetter:                     pr.RequireLetter,
		RequireUppercaseLetter:            pr.RequireUppercaseLetter,
		RequireSpecialCharacter:           pr.RequireSpecialCharacter,
		RequireResetPasswordForFirstLogin: pr.RequireResetPasswordForFirstLogin,
		PasswordRotation:                  pr.PasswordRotation,
	}
}
