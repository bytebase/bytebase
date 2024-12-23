package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/relay"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func configureGrpcRouters(
	ctx context.Context,
	mux *grpcruntime.ServeMux,
	grpcServer *grpc.Server,
	stores *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	licenseService enterprise.LicenseService,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	stateCfg *state.State,
	schemaSyncer *schemasync.Syncer,
	webhookManager *webhook.Manager,
	iamManager *iam.Manager,
	relayRunner *relay.Runner,
	postCreateUser apiv1.CreateUserFunc,
	secret string) (*apiv1.ReleaseService, *apiv1.PlanService, *apiv1.RolloutService, *apiv1.IssueService, *apiv1.SQLService, error) {
	// Register services.
	authService, err := apiv1.NewAuthService(stores, secret, licenseService, metricReporter, profile, stateCfg, iamManager, postCreateUser)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	v1pb.RegisterAuditLogServiceServer(grpcServer, apiv1.NewAuditLogService(stores, iamManager, licenseService))
	v1pb.RegisterAuthServiceServer(grpcServer, authService)
	v1pb.RegisterActuatorServiceServer(grpcServer, apiv1.NewActuatorService(stores, profile, licenseService))
	v1pb.RegisterSubscriptionServiceServer(grpcServer, apiv1.NewSubscriptionService(
		stores,
		profile,
		metricReporter,
		licenseService))
	v1pb.RegisterEnvironmentServiceServer(grpcServer, apiv1.NewEnvironmentService(stores, licenseService))
	v1pb.RegisterInstanceServiceServer(grpcServer, apiv1.NewInstanceService(
		stores,
		licenseService,
		metricReporter,
		secret,
		stateCfg,
		dbFactory,
		schemaSyncer,
		iamManager))
	v1pb.RegisterProjectServiceServer(grpcServer, apiv1.NewProjectService(stores, profile, iamManager, licenseService))
	v1pb.RegisterDatabaseServiceServer(grpcServer, apiv1.NewDatabaseService(stores, schemaSyncer, licenseService, profile, iamManager))
	v1pb.RegisterDatabaseCatalogServiceServer(grpcServer, apiv1.NewDatabaseCatalogService(stores, licenseService))
	v1pb.RegisterInstanceRoleServiceServer(grpcServer, apiv1.NewInstanceRoleService(stores, dbFactory))
	v1pb.RegisterOrgPolicyServiceServer(grpcServer, apiv1.NewOrgPolicyService(stores, licenseService))
	v1pb.RegisterWorkspaceServiceServer(grpcServer, apiv1.NewWorkspaceService(stores, iamManager))
	v1pb.RegisterIdentityProviderServiceServer(grpcServer, apiv1.NewIdentityProviderService(stores, licenseService))
	v1pb.RegisterSettingServiceServer(grpcServer, apiv1.NewSettingService(stores, profile, licenseService, stateCfg))
	v1pb.RegisterAnomalyServiceServer(grpcServer, apiv1.NewAnomalyService(stores))
	sqlService := apiv1.NewSQLService(stores, sheetManager, schemaSyncer, dbFactory, licenseService, profile, iamManager)
	v1pb.RegisterSQLServiceServer(grpcServer, sqlService)
	v1pb.RegisterVCSProviderServiceServer(grpcServer, apiv1.NewVCSProviderService(stores))
	v1pb.RegisterRiskServiceServer(grpcServer, apiv1.NewRiskService(stores, licenseService))
	releaseService := apiv1.NewReleaseService(stores, sheetManager, schemaSyncer, dbFactory)
	v1pb.RegisterReleaseServiceServer(grpcServer, releaseService)
	planService := apiv1.NewPlanService(stores, sheetManager, licenseService, dbFactory, stateCfg, profile, iamManager)
	v1pb.RegisterPlanServiceServer(grpcServer, planService)
	issueService := apiv1.NewIssueService(stores, webhookManager, relayRunner, stateCfg, licenseService, profile, iamManager, metricReporter)
	v1pb.RegisterIssueServiceServer(grpcServer, issueService)
	rolloutService := apiv1.NewRolloutService(stores, sheetManager, licenseService, dbFactory, stateCfg, webhookManager, profile, iamManager)
	v1pb.RegisterRolloutServiceServer(grpcServer, rolloutService)
	v1pb.RegisterRoleServiceServer(grpcServer, apiv1.NewRoleService(stores, iamManager, licenseService))
	v1pb.RegisterSheetServiceServer(grpcServer, apiv1.NewSheetService(stores, sheetManager, licenseService, iamManager, profile))
	v1pb.RegisterWorksheetServiceServer(grpcServer, apiv1.NewWorksheetService(stores, iamManager))
	v1pb.RegisterBranchServiceServer(grpcServer, apiv1.NewBranchService(stores, licenseService, profile, iamManager))
	v1pb.RegisterCelServiceServer(grpcServer, apiv1.NewCelService())
	v1pb.RegisterDatabaseGroupServiceServer(grpcServer, apiv1.NewDatabaseGroupService(stores, profile, iamManager, licenseService))
	v1pb.RegisterChangelistServiceServer(grpcServer, apiv1.NewChangelistService(stores, profile, iamManager))
	v1pb.RegisterVCSConnectorServiceServer(grpcServer, apiv1.NewVCSConnectorService(stores))
	v1pb.RegisterGroupServiceServer(grpcServer, apiv1.NewGroupService(stores, iamManager))
	v1pb.RegisterReviewConfigServiceServer(grpcServer, apiv1.NewReviewConfigService(stores, licenseService))

	// REST gateway proxy.
	grpcEndpoint := fmt.Sprintf(":%d", profile.Port)
	grpcConn, err := grpc.Dial(
		grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024), // Set MaxCallRecvMsgSize to 100M so that users can receive up to 100M via REST calls.
		),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// Sort by service name, align with api.bytebase.com.
	if err := v1pb.RegisterActuatorServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterAnomalyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterAuditLogServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterAuthServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterBranchServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterCelServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterChangelistServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterDatabaseGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterDatabaseCatalogServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterEnvironmentServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterInstanceRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterInstanceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterIssueServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterOrgPolicyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterPlanServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterProjectServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterReviewConfigServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterRiskServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterRolloutServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterSQLServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterSettingServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterSheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterSubscriptionServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterVCSConnectorServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterVCSProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterWorksheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterWorkspaceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if err := v1pb.RegisterReleaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return releaseService, planService, rolloutService, issueService, sqlService, nil
}
