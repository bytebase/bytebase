package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/validate"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v5"
	"github.com/pkg/errors"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"

	"github.com/bytebase/bytebase/backend/api/auth"
	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/enterprise"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
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
	bus *bus.Bus,
	schemaSyncer *schemasync.Syncer,
	webhookManager *webhook.Manager,
	iamManager *iam.Manager,
	secret string,
	sampleInstanceManager *sampleinstance.Manager,
) (http.Handler, error) {
	// Note: the gateway response modifier takes the token duration on server startup. If the value is changed,
	// the user has to restart the server to take the latest value.
	gatewayMarshaler := &grpcruntime.HTTPBodyMarshaler{
		Marshaler: newSuggestingMarshaler(&grpcruntime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{},
			//nolint:forbidigo
			UnmarshalOptions: protojson.UnmarshalOptions{},
		}),
	}
	mux := grpcruntime.NewServeMux(
		grpcruntime.WithMarshalerOption(grpcruntime.MIMEWildcard, gatewayMarshaler),
		// pass through request headers that need to be used by connect rpc handlers.
		grpcruntime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			// grpc-gateway hard codes authorization pass-through already, we do it again anyways.
			// https://github.com/grpc-ecosystem/grpc-gateway/blob/2cca0efe61de30f05068b9e3b4eb4801b1b2c1aa/runtime/context.go#L160
			case "authorization", "cookie", "origin":
				return key, true
			default:
				return "", false
			}
		}),
		grpcruntime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "set-cookie":
				return key, true
			default:
				return "", false
			}
		}),
		grpcruntime.WithRoutingErrorHandler(func(ctx context.Context, sm *grpcruntime.ServeMux, m grpcruntime.Marshaler, w http.ResponseWriter, r *http.Request, httpStatus int) {
			err := &grpcruntime.HTTPStatusError{
				HTTPStatus: httpStatus,
				Err:        connect.NewError(connect.CodeNotFound, errors.Errorf("gateway routing error %d: request method %v, URI %v", httpStatus, r.Method, r.RequestURI)),
			}
			grpcruntime.DefaultHTTPErrorHandler(ctx, sm, m, w, r, err)
		}),
	)
	aiService := apiv1.NewAIService(stores)
	accessGrantService := apiv1.NewAccessGrantService(stores, licenseService, webhookManager, bus)
	actuatorService := apiv1.NewActuatorService(stores, profile, schemaSyncer, licenseService, sampleInstanceManager)
	auditLogService := apiv1.NewAuditLogService(stores, licenseService)
	authService := apiv1.NewAuthService(stores, secret, licenseService, profile, iamManager)
	celService := apiv1.NewCelService()
	changelogService := apiv1.NewChangelogService(stores)
	databaseCatalogService := apiv1.NewDatabaseCatalogService(stores)
	databaseGroupService := apiv1.NewDatabaseGroupService(stores, licenseService)
	databaseService := apiv1.NewDatabaseService(stores, schemaSyncer, profile, iamManager, licenseService)
	groupService := apiv1.NewGroupService(stores, iamManager, licenseService)
	identityProviderService := apiv1.NewIdentityProviderService(stores, licenseService, profile)
	instanceRoleService := apiv1.NewInstanceRoleService(stores)
	instanceService := apiv1.NewInstanceService(stores, profile, licenseService, dbFactory, schemaSyncer, sampleInstanceManager)
	issueService := apiv1.NewIssueService(stores, webhookManager, bus, licenseService, iamManager)
	orgPolicyService := apiv1.NewOrgPolicyService(stores, licenseService, iamManager)
	planService := apiv1.NewPlanService(stores, bus, iamManager, webhookManager, licenseService)
	projectService := apiv1.NewProjectService(stores, profile, iamManager)
	queryHistoryService := apiv1.NewQueryHistoryService(stores)
	releaseService := apiv1.NewReleaseService(stores, sheetManager, dbFactory, licenseService)
	reviewConfigService := apiv1.NewReviewConfigService(stores)
	revisionService := apiv1.NewRevisionService(stores)
	roleService := apiv1.NewRoleService(stores, iamManager, licenseService)
	rolloutService := apiv1.NewRolloutService(stores, dbFactory, bus, webhookManager, iamManager)
	settingService := apiv1.NewSettingService(stores, profile, licenseService, iamManager)
	sheetService := apiv1.NewSheetService(stores)
	sqlService := apiv1.NewSQLService(stores, schemaSyncer, dbFactory, licenseService, iamManager, queryHistoryService)
	subscriptionService := apiv1.NewSubscriptionService(profile, stores, licenseService)
	userService := apiv1.NewUserService(stores, licenseService, profile, iamManager)
	serviceAccountService := apiv1.NewServiceAccountService(stores, profile, iamManager)
	workloadIdentityService := apiv1.NewWorkloadIdentityService(stores, profile, iamManager)
	worksheetService := apiv1.NewWorksheetService(stores, iamManager)
	workspaceService := apiv1.NewWorkspaceService(stores, iamManager, profile, licenseService, authService)

	onPanic := func(_ context.Context, s connect.Spec, _ http.Header, p any) error {
		stack := stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */)
		// keep a multiline stack
		slog.Error("v1 server panic error", "method", s.Procedure, log.BBError(errors.Errorf("error: %v\n%s", p, stack)))
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}

	// Create validation interceptor.
	validateInterceptor := validate.NewInterceptor()

	handlerOpts := connect.WithHandlerOptions(
		connect.WithRecover(onPanic),
		connect.WithInterceptors(
			validateInterceptor,
			auth.New(stores, secret, licenseService, bus, profile),
			apiv1.NewACLInterceptor(stores, secret, iamManager, profile),
			apiv1.NewAuditInterceptor(stores, secret, profile),
		),
	)

	// registerServiceHandlers builds one connect handler per v1 service with the
	// given interceptor chain. Called twice: once for the public chain and once
	// for the internal MCP chain — same service instances, different auth.
	registerServiceHandlers := func(opts connect.HandlerOption) map[string]http.Handler {
		handlers := make(map[string]http.Handler)
		add := func(path string, handler http.Handler) {
			handlers[path] = handler
		}
		add(v1connect.NewAIServiceHandler(aiService, opts))
		add(v1connect.NewAccessGrantServiceHandler(accessGrantService, opts))
		add(v1connect.NewActuatorServiceHandler(actuatorService, opts))
		add(v1connect.NewAuditLogServiceHandler(auditLogService, opts))
		add(v1connect.NewAuthServiceHandler(authService, opts))
		add(v1connect.NewCelServiceHandler(celService, opts))
		add(v1connect.NewChangelogServiceHandler(changelogService, opts))
		add(v1connect.NewDatabaseCatalogServiceHandler(databaseCatalogService, opts))
		add(v1connect.NewDatabaseGroupServiceHandler(databaseGroupService, opts))
		add(v1connect.NewDatabaseServiceHandler(databaseService, opts))
		add(v1connect.NewGroupServiceHandler(groupService, opts))
		add(v1connect.NewIdentityProviderServiceHandler(identityProviderService, opts))
		add(v1connect.NewInstanceRoleServiceHandler(instanceRoleService, opts))
		add(v1connect.NewInstanceServiceHandler(instanceService, opts))
		add(v1connect.NewIssueServiceHandler(issueService, opts))
		add(v1connect.NewOrgPolicyServiceHandler(orgPolicyService, opts))
		add(v1connect.NewPlanServiceHandler(planService, opts))
		add(v1connect.NewProjectServiceHandler(projectService, opts))
		add(v1connect.NewQueryHistoryServiceHandler(queryHistoryService, opts))
		add(v1connect.NewReleaseServiceHandler(releaseService, opts))
		add(v1connect.NewReviewConfigServiceHandler(reviewConfigService, opts))
		add(v1connect.NewRevisionServiceHandler(revisionService, opts))
		add(v1connect.NewRoleServiceHandler(roleService, opts))
		add(v1connect.NewRolloutServiceHandler(rolloutService, opts))
		add(v1connect.NewSettingServiceHandler(settingService, opts))
		add(v1connect.NewSheetServiceHandler(sheetService, opts))
		add(v1connect.NewSQLServiceHandler(sqlService, opts))
		add(v1connect.NewSubscriptionServiceHandler(subscriptionService, opts))
		add(v1connect.NewUserServiceHandler(userService, opts))
		add(v1connect.NewServiceAccountServiceHandler(serviceAccountService, opts))
		add(v1connect.NewWorkloadIdentityServiceHandler(workloadIdentityService, opts))
		add(v1connect.NewWorksheetServiceHandler(worksheetService, opts))
		add(v1connect.NewWorkspaceServiceHandler(workspaceService, opts))
		return handlers
	}

	connectHandlers := registerServiceHandlers(handlerOpts)

	// The internal MCP handler chain: the same service handlers behind an auth
	// interceptor that accepts ONLY the delegated credential minted at the /mcp
	// boundary. It is reachable exclusively through the in-memory transport
	// handed to the MCP server — never bound to a listener, so the internal
	// credential never touches a socket. ACL runs exactly as on the public
	// chain: the credential carries identity + grant state, while authorization
	// is re-resolved live per request.
	//
	// Unlike the public chain, audit sits OUTSIDE ACL (first-listed interceptor
	// is outermost, so listing audit before ACL wraps it): an ACL denial must
	// still produce an audit row, because a denied MCP call is exactly the
	// event an operator investigating an agent needs to see. Methods whose
	// annotation opts out of auditing stay unaudited for permitted and denied
	// calls alike (needAudit gates both).
	//
	// The FORBIDDEN gate sits between them — inside audit, so a denial is
	// recorded wherever the method's annotation asks for auditing at all, and
	// outside ACL, because the class is refused whatever the caller's RBAC
	// would have allowed. P1b's full ceiling gate takes this same slot, and
	// brings the typed denial record for the unannotated methods with it.
	internalHandlerOpts := connect.WithHandlerOptions(
		connect.WithRecover(onPanic),
		connect.WithInterceptors(
			validateInterceptor,
			auth.NewInternalMCPAuthInterceptor(stores, secret, profile),
			apiv1.NewAuditInterceptor(stores, secret, profile),
			apiv1.NewInternalMCPForbiddenInterceptor(),
			apiv1.NewACLInterceptor(stores, secret, iamManager, profile),
		),
	)
	internalMCPMux := http.NewServeMux()
	for path, handler := range registerServiceHandlers(internalHandlerOpts) {
		internalMCPMux.Handle(path, handler)
	}

	// grpc reflection handlers.
	reflector := grpcreflect.NewStaticReflector(
		v1connect.AIServiceName,
		v1connect.AccessGrantServiceName,
		v1connect.ActuatorServiceName,
		v1connect.AuditLogServiceName,
		v1connect.AuthServiceName,
		v1connect.CelServiceName,
		v1connect.ChangelogServiceName,
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
		v1connect.QueryHistoryServiceName,
		v1connect.ReleaseServiceName,
		v1connect.ReviewConfigServiceName,
		v1connect.RevisionServiceName,
		v1connect.RoleServiceName,
		v1connect.RolloutServiceName,
		v1connect.SettingServiceName,
		v1connect.ServiceAccountServiceName,
		v1connect.SheetServiceName,
		v1connect.SQLServiceName,
		v1connect.SubscriptionServiceName,
		v1connect.UserServiceName,
		v1connect.WorkloadIdentityServiceName,
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
		return nil, err
	}

	if err := v1pb.RegisterAIServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterAccessGrantServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
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
	if err := v1pb.RegisterChangelogServiceHandler(ctx, mux, grpcConn); err != nil {
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
	if err := v1pb.RegisterQueryHistoryServiceHandler(ctx, mux, grpcConn); err != nil {
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
	if err := v1pb.RegisterServiceAccountServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterWorkloadIdentityServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterWorksheetServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterWorkspaceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
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

	return internalMCPMux, nil
}
