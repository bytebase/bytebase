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
	grpcServer *grpc.Server,
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

	// Register services.
	v1pb.RegisterSubscriptionServiceServer(grpcServer, apiv1.NewSubscriptionService(
		stores,
		profile,
		metricReporter,
		licenseService))
	v1pb.RegisterInstanceServiceServer(grpcServer, apiv1.NewInstanceService(
		stores,
		licenseService,
		metricReporter,
		stateCfg,
		dbFactory,
		schemaSyncer,
		iamManager))
	v1pb.RegisterProjectServiceServer(grpcServer, apiv1.NewProjectService(stores, profile, iamManager, licenseService))
	v1pb.RegisterDatabaseServiceServer(grpcServer, apiv1.NewDatabaseService(stores, schemaSyncer, licenseService, profile, iamManager))
	v1pb.RegisterDatabaseCatalogServiceServer(grpcServer, apiv1.NewDatabaseCatalogService(stores, licenseService))
	v1pb.RegisterInstanceRoleServiceServer(grpcServer, apiv1.NewInstanceRoleService(stores, dbFactory))
	v1pb.RegisterOrgPolicyServiceServer(grpcServer, apiv1.NewOrgPolicyService(stores, licenseService))
	v1pb.RegisterIdentityProviderServiceServer(grpcServer, apiv1.NewIdentityProviderService(stores, licenseService))
	// SettingService is now handled by Connect RPC
	sqlService := apiv1.NewSQLService(stores, sheetManager, schemaSyncer, dbFactory, licenseService, profile, iamManager)
	v1pb.RegisterSQLServiceServer(grpcServer, sqlService)
	releaseService := apiv1.NewReleaseService(stores, sheetManager, schemaSyncer, dbFactory)
	v1pb.RegisterReleaseServiceServer(grpcServer, releaseService)
	planService := apiv1.NewPlanService(stores, sheetManager, licenseService, dbFactory, stateCfg, profile, iamManager)
	v1pb.RegisterPlanServiceServer(grpcServer, planService)
	issueService := apiv1.NewIssueService(stores, webhookManager, stateCfg, licenseService, profile, iamManager, metricReporter)
	v1pb.RegisterIssueServiceServer(grpcServer, issueService)
	rolloutService := apiv1.NewRolloutService(stores, sheetManager, licenseService, dbFactory, stateCfg, webhookManager, profile, iamManager)
	v1pb.RegisterRolloutServiceServer(grpcServer, rolloutService)
	// RoleService is now handled by Connect RPC
	v1pb.RegisterSheetServiceServer(grpcServer, apiv1.NewSheetService(stores, sheetManager, licenseService, iamManager, profile))
	v1pb.RegisterDatabaseGroupServiceServer(grpcServer, apiv1.NewDatabaseGroupService(stores, profile, iamManager, licenseService))
	v1pb.RegisterChangelistServiceServer(grpcServer, apiv1.NewChangelistService(stores, profile, iamManager))
	v1pb.RegisterGroupServiceServer(grpcServer, apiv1.NewGroupService(stores, iamManager, licenseService))
	v1pb.RegisterReviewConfigServiceServer(grpcServer, apiv1.NewReviewConfigService(stores, licenseService))

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

	// Sort by service name, align with api.bytebase.com.
	// ActuatorService is now handled by Connect RPC
	// UserService is now handled by Connect RPC
	// AuditLogService is now handled by Connect RPC
	// AuthService is now handled by Connect RPC
	// CelService is now handled by Connect RPC
	if err := v1pb.RegisterChangelistServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	// RevisionService is now handled by Connect RPC
	if err := v1pb.RegisterDatabaseCatalogServiceHandler(ctx, mux, grpcConn); err != nil {
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
	if err := v1pb.RegisterReviewConfigServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	// RiskService is now handled by Connect RPC
	// RoleService is now handled by Connect RPC
	if err := v1pb.RegisterRolloutServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSQLServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	// SettingService is now handled by Connect RPC
	if err := v1pb.RegisterSheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterSubscriptionServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	// WorksheetService is now handled by Connect RPC
	// WorkspaceService is now handled by Connect RPC
	if err := v1pb.RegisterReleaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	return connectHandlers, nil
}
