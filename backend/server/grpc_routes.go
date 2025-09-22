package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"

	"github.com/bytebase/bytebase/backend/api/auth"
	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

func configureGrpcRouters(
	ctx context.Context,
	e *echo.Echo,
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
	sampleInstanceManager *sampleinstance.Manager,
) error {
	// Note: the gateway response modifier takes the token duration on server startup. If the value is changed,
	// the user has to restart the server to take the latest value.
	gatewayModifier := auth.GatewayResponseModifier{Store: stores, LicenseService: licenseService}
	mux := grpcruntime.NewServeMux(
		grpcruntime.WithMarshalerOption(grpcruntime.MIMEWildcard, &grpcruntime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{},
			//nolint:forbidigo
			UnmarshalOptions: protojson.UnmarshalOptions{},
		}),
		grpcruntime.WithForwardResponseOption(gatewayModifier.Modify),
		grpcruntime.WithRoutingErrorHandler(func(ctx context.Context, sm *grpcruntime.ServeMux, m grpcruntime.Marshaler, w http.ResponseWriter, r *http.Request, httpStatus int) {
			err := &grpcruntime.HTTPStatusError{
				HTTPStatus: httpStatus,
				Err:        connect.NewError(connect.CodeNotFound, errors.Errorf("gateway routing error %d: request method %v, URI %v", httpStatus, r.Method, r.RequestURI)),
			}
			grpcruntime.DefaultHTTPErrorHandler(ctx, sm, m, w, r, err)
		}),
	)
	actuatorService := apiv1.NewActuatorService(stores, profile, schemaSyncer, licenseService, sampleInstanceManager)
	auditLogService := apiv1.NewAuditLogService(stores, iamManager, licenseService)
	authService := apiv1.NewAuthService(stores, secret, licenseService, metricReporter, profile, stateCfg, iamManager)
	celService := apiv1.NewCelService()
	changelistService := apiv1.NewChangelistService(stores, profile, iamManager)
	databaseCatalogService := apiv1.NewDatabaseCatalogService(stores, licenseService)
	databaseGroupService := apiv1.NewDatabaseGroupService(stores, profile, iamManager, licenseService)
	databaseService := apiv1.NewDatabaseService(stores, schemaSyncer, licenseService, profile, iamManager)
	groupService := apiv1.NewGroupService(stores, iamManager, licenseService)
	identityProviderService := apiv1.NewIdentityProviderService(stores, licenseService)
	instanceRoleService := apiv1.NewInstanceRoleService(stores, dbFactory)
	instanceService := apiv1.NewInstanceService(stores, licenseService, metricReporter, stateCfg, dbFactory, schemaSyncer, iamManager, sampleInstanceManager)
	issueService := apiv1.NewIssueService(stores, webhookManager, stateCfg, licenseService, profile, iamManager, metricReporter)
	orgPolicyService := apiv1.NewOrgPolicyService(stores, licenseService)
	planService := apiv1.NewPlanService(stores, sheetManager, licenseService, dbFactory, stateCfg, profile, iamManager)
	projectService := apiv1.NewProjectService(stores, profile, iamManager, licenseService)
	releaseService := apiv1.NewReleaseService(stores, sheetManager, schemaSyncer, dbFactory, iamManager)
	reviewConfigService := apiv1.NewReviewConfigService(stores, licenseService)
	revisionService := apiv1.NewRevisionService(stores)
	riskService := apiv1.NewRiskService(stores, iamManager, licenseService)
	roleService := apiv1.NewRoleService(stores, iamManager, licenseService)
	rolloutService := apiv1.NewRolloutService(stores, sheetManager, licenseService, dbFactory, stateCfg, webhookManager, profile, iamManager)
	settingService := apiv1.NewSettingService(stores, profile, licenseService, stateCfg)
	sheetService := apiv1.NewSheetService(stores, sheetManager, licenseService, iamManager, profile)
	sqlService := apiv1.NewSQLService(stores, sheetManager, schemaSyncer, dbFactory, licenseService, profile, iamManager)
	subscriptionService := apiv1.NewSubscriptionService(stores, profile, metricReporter, licenseService)
	userService := apiv1.NewUserService(stores, secret, licenseService, metricReporter, profile, stateCfg, iamManager)
	worksheetService := apiv1.NewWorksheetService(stores, iamManager)
	workspaceService := apiv1.NewWorkspaceService(stores, iamManager)

	onPanic := func(_ context.Context, s connect.Spec, _ http.Header, p any) error {
		stack := stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */)
		// keep a multiline stack
		slog.Error("v1 server panic error", "method", s.Procedure, log.BBError(errors.Errorf("error: %v\n%s", p, stack)))
		return connect.NewError(connect.CodeInternal, errors.Errorf("error: %v\n%s", p, stack))
	}

	handlerOpts := connect.WithHandlerOptions(
		connect.WithInterceptors(
			apiv1.NewDebugInterceptor(metricReporter),
			auth.New(stores, secret, licenseService, stateCfg, profile),
			apiv1.NewACLInterceptor(stores, secret, iamManager, profile),
			apiv1.NewAuditInterceptor(stores),
		),
		connect.WithRecover(onPanic),
	)

	connectHandlers := make(map[string]http.Handler)

	actuatorPath, actuatorHandler := v1connect.NewActuatorServiceHandler(actuatorService, handlerOpts)
	connectHandlers[actuatorPath] = actuatorHandler

	auditLogPath, auditLogHandler := v1connect.NewAuditLogServiceHandler(auditLogService, handlerOpts)
	connectHandlers[auditLogPath] = auditLogHandler

	authPath, authHandler := v1connect.NewAuthServiceHandler(authService, handlerOpts)
	connectHandlers[authPath] = authHandler

	celPath, celHandler := v1connect.NewCelServiceHandler(celService, handlerOpts)
	connectHandlers[celPath] = celHandler

	changelistPath, changelistHandler := v1connect.NewChangelistServiceHandler(changelistService, handlerOpts)
	connectHandlers[changelistPath] = changelistHandler

	databaseCatalogPath, databaseCatalogHandler := v1connect.NewDatabaseCatalogServiceHandler(databaseCatalogService, handlerOpts)
	connectHandlers[databaseCatalogPath] = databaseCatalogHandler

	databaseGroupPath, databaseGroupHandler := v1connect.NewDatabaseGroupServiceHandler(databaseGroupService, handlerOpts)
	connectHandlers[databaseGroupPath] = databaseGroupHandler

	databasePath, databaseHandler := v1connect.NewDatabaseServiceHandler(databaseService, handlerOpts)
	connectHandlers[databasePath] = databaseHandler

	groupPath, groupHandler := v1connect.NewGroupServiceHandler(groupService, handlerOpts)
	connectHandlers[groupPath] = groupHandler

	identityProviderPath, identityProviderHandler := v1connect.NewIdentityProviderServiceHandler(identityProviderService, handlerOpts)
	connectHandlers[identityProviderPath] = identityProviderHandler

	instanceRolePath, instanceRoleHandler := v1connect.NewInstanceRoleServiceHandler(instanceRoleService, handlerOpts)
	connectHandlers[instanceRolePath] = instanceRoleHandler

	instancePath, instanceHandler := v1connect.NewInstanceServiceHandler(instanceService, handlerOpts)
	connectHandlers[instancePath] = instanceHandler

	issuePath, issueHandler := v1connect.NewIssueServiceHandler(issueService, handlerOpts)
	connectHandlers[issuePath] = issueHandler

	orgPolicyPath, orgPolicyHandler := v1connect.NewOrgPolicyServiceHandler(orgPolicyService, handlerOpts)
	connectHandlers[orgPolicyPath] = orgPolicyHandler

	planPath, planHandler := v1connect.NewPlanServiceHandler(planService, handlerOpts)
	connectHandlers[planPath] = planHandler

	projectPath, projectHandler := v1connect.NewProjectServiceHandler(projectService, handlerOpts)
	connectHandlers[projectPath] = projectHandler

	releasePath, releaseHandler := v1connect.NewReleaseServiceHandler(releaseService, handlerOpts)
	connectHandlers[releasePath] = releaseHandler

	reviewConfigPath, reviewConfigHandler := v1connect.NewReviewConfigServiceHandler(reviewConfigService, handlerOpts)
	connectHandlers[reviewConfigPath] = reviewConfigHandler

	revisionPath, revisionHandler := v1connect.NewRevisionServiceHandler(revisionService, handlerOpts)
	connectHandlers[revisionPath] = revisionHandler

	riskPath, riskHandler := v1connect.NewRiskServiceHandler(riskService, handlerOpts)
	connectHandlers[riskPath] = riskHandler

	rolePath, roleHandler := v1connect.NewRoleServiceHandler(roleService, handlerOpts)
	connectHandlers[rolePath] = roleHandler

	rolloutPath, rolloutHandler := v1connect.NewRolloutServiceHandler(rolloutService, handlerOpts)
	connectHandlers[rolloutPath] = rolloutHandler

	settingPath, settingHandler := v1connect.NewSettingServiceHandler(settingService, handlerOpts)
	connectHandlers[settingPath] = settingHandler

	sheetPath, sheetHandler := v1connect.NewSheetServiceHandler(sheetService, handlerOpts)
	connectHandlers[sheetPath] = sheetHandler

	sqlPath, sqlHandler := v1connect.NewSQLServiceHandler(sqlService, handlerOpts)
	connectHandlers[sqlPath] = sqlHandler

	subscriptionPath, subscriptionHandler := v1connect.NewSubscriptionServiceHandler(subscriptionService, handlerOpts)
	connectHandlers[subscriptionPath] = subscriptionHandler

	userPath, userHandler := v1connect.NewUserServiceHandler(userService, handlerOpts)
	connectHandlers[userPath] = userHandler

	worksheetPath, worksheetHandler := v1connect.NewWorksheetServiceHandler(worksheetService, handlerOpts)
	connectHandlers[worksheetPath] = worksheetHandler

	workspacePath, workspaceHandler := v1connect.NewWorkspaceServiceHandler(workspaceService, handlerOpts)
	connectHandlers[workspacePath] = workspaceHandler

	// grpc reflection handlers.
	reflector := grpcreflect.NewStaticReflector(
		v1connect.ActuatorServiceName,
		v1connect.AuditLogServiceName,
		v1connect.AuthServiceName,
		v1connect.CelServiceName,
		v1connect.ChangelistServiceName,
		v1connect.DatabaseCatalogServiceName,
		v1connect.DatabaseGroupServiceName,
		v1connect.DatabaseServiceName,
		v1connect.GroupServiceName,
		v1connect.IdentityProviderServiceName,
		v1connect.InstanceRoleServiceName,
		v1connect.InstanceServiceName,
		v1connect.IssueServiceName,
		v1connect.OrgPolicyServiceName,
		v1connect.PlanServiceName,
		v1connect.ProjectServiceName,
		v1connect.ReleaseServiceName,
		v1connect.ReviewConfigServiceName,
		v1connect.RevisionServiceName,
		v1connect.RiskServiceName,
		v1connect.RoleServiceName,
		v1connect.RolloutServiceName,
		v1connect.SettingServiceName,
		v1connect.SheetServiceName,
		v1connect.SQLServiceName,
		v1connect.SubscriptionServiceName,
		v1connect.UserServiceName,
		v1connect.WorksheetServiceName,
		v1connect.WorkspaceServiceName,
	)
	reflectPath, reflectHandler := grpcreflect.NewHandlerV1(reflector)
	connectHandlers[reflectPath] = reflectHandler

	reflectAlphaPath, reflectAlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	connectHandlers[reflectAlphaPath] = reflectAlphaHandler

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
		return err
	}

	if err := v1pb.RegisterActuatorServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterAuditLogServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterAuthServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterCelServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterChangelistServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterDatabaseCatalogServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterDatabaseGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterGroupServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterInstanceRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterInstanceServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterIssueServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterOrgPolicyServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterPlanServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterProjectServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterReleaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterReviewConfigServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterRevisionServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterRiskServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterRolloutServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterSettingServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterSheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterSQLServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterSubscriptionServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterUserServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterWorksheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	if err := v1pb.RegisterWorkspaceServiceHandler(ctx, mux, grpcConn); err != nil {
		return err
	}
	// Register echo routes for mux and connectHandlers
	e.GET("/v1:adminExecute", echo.WrapHandler(wsproxy.WebsocketProxy(
		mux,
		wsproxy.WithTokenCookieName("access-token"),
		// 100M.
		wsproxy.WithMaxRespBodyBufferSize(100*1024*1024),
	)))
	e.Any("/v1/*", echo.WrapHandler(mux))

	// Register Connect RPC handlers
	for path, handler := range connectHandlers {
		e.Any(path+"*", echo.WrapHandler(handler))
	}

	return nil
}
