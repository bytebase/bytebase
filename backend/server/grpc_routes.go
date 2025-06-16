package server

import (
	"context"
	"fmt"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	connectcors "connectrpc.com/cors"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/proto/generated-go/v1/v1connect"
)

// withCORS adds CORS support to a Connect HTTP handler following Connect RPC documentation.
func withCORS(h http.Handler) http.Handler {
	middleware := cors.New(cors.Options{
		AllowOriginFunc: func(string) bool {
			return true
		},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   connectcors.ExposedHeaders(),
		AllowCredentials: true,
	})
	return middleware.Handler(h)
}

func configureGrpcRouters(
	ctx context.Context,
	mux *grpcruntime.ServeMux,
	_ *grpc.Server,
	stores *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
	licenseService *enterprise.LicenseService,
	profile *config.Profile,
	metricReporter *metricreport.Reporter,
	stateCfg *state.State,
	schemaSyncer *schemasync.Syncer,
	webhookManager *webhook.Manager,
	iamManager *iam.Manager,
	secret string,
) (map[string]http.Handler, error) {
	// Services that have been migrated to Connect RPC
	actuatorService := apiv1.NewActuatorService(stores, profile, schemaSyncer, licenseService)
	auditLogService := apiv1.NewAuditLogService(stores, iamManager, licenseService)
	celService := apiv1.NewCelService()
	revisionService := apiv1.NewRevisionService(stores)
	riskService := apiv1.NewRiskService(stores, licenseService)
	worksheetService := apiv1.NewWorksheetService(stores, iamManager)

	// Create Connect RPC handlers with CORS support following Connect RPC documentation
	connectHandlers := make(map[string]http.Handler)
	actuatorPath, actuatorHandler := v1connect.NewActuatorServiceHandler(actuatorService)

	// Add CORS support using Connect RPC's recommended approach
	connectHandlers[actuatorPath] = withCORS(actuatorHandler)

	// Register Phase 1 services with Connect RPC
	auditLogPath, auditLogHandler := v1connect.NewAuditLogServiceHandler(auditLogService)
	connectHandlers[auditLogPath] = withCORS(auditLogHandler)

	celPath, celHandler := v1connect.NewCelServiceHandler(celService)
	connectHandlers[celPath] = withCORS(celHandler)

	revisionPath, revisionHandler := v1connect.NewRevisionServiceHandler(revisionService)
	connectHandlers[revisionPath] = withCORS(revisionHandler)

	riskPath, riskHandler := v1connect.NewRiskServiceHandler(riskService)
	connectHandlers[riskPath] = withCORS(riskHandler)

	worksheetPath, worksheetHandler := v1connect.NewWorksheetServiceHandler(worksheetService)
	connectHandlers[worksheetPath] = withCORS(worksheetHandler)

	// UserService has been migrated to Connect RPC
	userService := apiv1.NewUserService(stores, secret, licenseService, metricReporter, profile, stateCfg, iamManager)
	userPath, userHandler := v1connect.NewUserServiceHandler(userService)
	connectHandlers[userPath] = withCORS(userHandler)

	// AuthService has been migrated to Connect RPC
	authService := apiv1.NewAuthService(stores, secret, licenseService, metricReporter, profile, stateCfg, iamManager)
	authPath, authHandler := v1connect.NewAuthServiceHandler(authService)
	connectHandlers[authPath] = withCORS(authHandler)

	// WorkspaceService has been migrated to Connect RPC
	workspaceService := apiv1.NewWorkspaceService(stores, iamManager)
	workspacePath, workspaceHandler := v1connect.NewWorkspaceServiceHandler(workspaceService)
	connectHandlers[workspacePath] = withCORS(workspaceHandler)

	// SettingService has been migrated to Connect RPC
	settingService := apiv1.NewSettingService(stores, profile, licenseService, stateCfg)
	settingPath, settingHandler := v1connect.NewSettingServiceHandler(settingService)
	connectHandlers[settingPath] = withCORS(settingHandler)

	// RoleService has been migrated to Connect RPC
	roleService := apiv1.NewRoleService(stores, iamManager, licenseService)
	rolePath, roleHandler := v1connect.NewRoleServiceHandler(roleService)
	connectHandlers[rolePath] = withCORS(roleHandler)

	// Phase 3 services migrated to Connect RPC
	projectService := apiv1.NewProjectService(stores, profile, iamManager, licenseService)
	projectPath, projectHandler := v1connect.NewProjectServiceHandler(projectService)
	connectHandlers[projectPath] = withCORS(projectHandler)

	instanceService := apiv1.NewInstanceService(stores, licenseService, metricReporter, stateCfg, dbFactory, schemaSyncer, iamManager)
	instancePath, instanceHandler := v1connect.NewInstanceServiceHandler(instanceService)
	connectHandlers[instancePath] = withCORS(instanceHandler)

	databaseService := apiv1.NewDatabaseService(stores, schemaSyncer, licenseService, profile, iamManager)
	databasePath, databaseHandler := v1connect.NewDatabaseServiceHandler(databaseService)
	connectHandlers[databasePath] = withCORS(databaseHandler)

	databaseGroupService := apiv1.NewDatabaseGroupService(stores, profile, iamManager, licenseService)
	databaseGroupPath, databaseGroupHandler := v1connect.NewDatabaseGroupServiceHandler(databaseGroupService)
	connectHandlers[databaseGroupPath] = withCORS(databaseGroupHandler)

	groupService := apiv1.NewGroupService(stores, iamManager, licenseService)
	groupPath, groupHandler := v1connect.NewGroupServiceHandler(groupService)
	connectHandlers[groupPath] = withCORS(groupHandler)

	// Phase 4 services migrated to Connect RPC
	sheetService := apiv1.NewSheetService(stores, sheetManager, licenseService, iamManager, profile)
	sheetPath, sheetHandler := v1connect.NewSheetServiceHandler(sheetService)
	connectHandlers[sheetPath] = withCORS(sheetHandler)

	sqlService := apiv1.NewSQLService(stores, sheetManager, schemaSyncer, dbFactory, licenseService, profile, iamManager)
	sqlPath, sqlHandler := v1connect.NewSQLServiceHandler(sqlService)
	connectHandlers[sqlPath] = withCORS(sqlHandler)

	issueService := apiv1.NewIssueService(stores, webhookManager, stateCfg, licenseService, profile, iamManager, metricReporter)
	issuePath, issueHandler := v1connect.NewIssueServiceHandler(issueService)
	connectHandlers[issuePath] = withCORS(issueHandler)

	rolloutService := apiv1.NewRolloutService(stores, sheetManager, licenseService, dbFactory, stateCfg, webhookManager, profile, iamManager)
	rolloutPath, rolloutHandler := v1connect.NewRolloutServiceHandler(rolloutService)
	connectHandlers[rolloutPath] = withCORS(rolloutHandler)

	planService := apiv1.NewPlanService(stores, sheetManager, licenseService, dbFactory, stateCfg, profile, iamManager)
	planPath, planHandler := v1connect.NewPlanServiceHandler(planService)
	connectHandlers[planPath] = withCORS(planHandler)

	// Phase 5 services migrated to Connect RPC
	subscriptionService := apiv1.NewSubscriptionService(stores, profile, metricReporter, licenseService)
	subscriptionPath, subscriptionHandler := v1connect.NewSubscriptionServiceHandler(subscriptionService)
	connectHandlers[subscriptionPath] = withCORS(subscriptionHandler)

	databaseCatalogService := apiv1.NewDatabaseCatalogService(stores, licenseService)
	databaseCatalogPath, databaseCatalogHandler := v1connect.NewDatabaseCatalogServiceHandler(databaseCatalogService)
	connectHandlers[databaseCatalogPath] = withCORS(databaseCatalogHandler)

	instanceRoleService := apiv1.NewInstanceRoleService(stores, dbFactory)
	instanceRolePath, instanceRoleHandler := v1connect.NewInstanceRoleServiceHandler(instanceRoleService)
	connectHandlers[instanceRolePath] = withCORS(instanceRoleHandler)

	orgPolicyService := apiv1.NewOrgPolicyService(stores, licenseService)
	orgPolicyPath, orgPolicyHandler := v1connect.NewOrgPolicyServiceHandler(orgPolicyService)
	connectHandlers[orgPolicyPath] = withCORS(orgPolicyHandler)

	identityProviderService := apiv1.NewIdentityProviderService(stores, licenseService)
	identityProviderPath, identityProviderHandler := v1connect.NewIdentityProviderServiceHandler(identityProviderService)
	connectHandlers[identityProviderPath] = withCORS(identityProviderHandler)

	releaseService := apiv1.NewReleaseService(stores, sheetManager, schemaSyncer, dbFactory)
	releasePath, releaseHandler := v1connect.NewReleaseServiceHandler(releaseService)
	connectHandlers[releasePath] = withCORS(releaseHandler)

	changelistService := apiv1.NewChangelistService(stores, profile, iamManager)
	changelistPath, changelistHandler := v1connect.NewChangelistServiceHandler(changelistService)
	connectHandlers[changelistPath] = withCORS(changelistHandler)

	reviewConfigService := apiv1.NewReviewConfigService(stores, licenseService)
	reviewConfigPath, reviewConfigHandler := v1connect.NewReviewConfigServiceHandler(reviewConfigService)
	connectHandlers[reviewConfigPath] = withCORS(reviewConfigHandler)

	// REST gateway proxy.
	grpcEndpoint := fmt.Sprintf(":%d", profile.Port)
	grpcConn, err := grpc.NewClient(
		grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024), // Set MaxCallRecvMsgSize to 100M so that users can receive up to 100M via REST calls.
		),
	)
	if err != nil {
		return nil, err
	}

	// Register all services in alphabetical order
	if err := v1pb.RegisterActuatorServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterAuditLogServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterAuthServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterCelServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterChangelistServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseCatalogServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterInstanceRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterInstanceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterIssueServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterOrgPolicyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterPlanServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterProjectServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterReleaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterReviewConfigServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterRevisionServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterRiskServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterRolloutServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSettingServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSQLServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSubscriptionServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterUserServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterWorksheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterWorkspaceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	return connectHandlers, nil
}
