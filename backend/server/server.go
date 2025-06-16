// Package server implements the API server for Bytebase.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path"
	"sync"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	"github.com/bytebase/bytebase/backend/api/auth"
	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/demo"
	"github.com/bytebase/bytebase/backend/enterprise"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/monitor"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// webhookAPIPrefix is the API prefix for Bytebase webhook.
	webhookAPIPrefix = "/hook"
	scimAPIPrefix    = "/scim"
	// lspAPI is the API for Bytebase Language Server Protocol.
	lspAPI                 = "/lsp"
	maxStacksize           = 1024 * 10240
	gracefulShutdownPeriod = 10 * time.Second
)

// Server is the Bytebase server.
type Server struct {
	// Asynchronous runners.
	taskSchedulerV2    *taskrun.SchedulerV2
	planCheckScheduler *plancheck.Scheduler
	metricReporter     *metricreport.Reporter
	schemaSyncer       *schemasync.Syncer
	approvalRunner     *approval.Runner
	runnerWG           sync.WaitGroup

	webhookManager *webhook.Manager
	iamManager     *iam.Manager

	licenseService *enterprise.LicenseService

	profile    *config.Profile
	echoServer *echo.Echo
	grpcServer *grpc.Server
	muxServer  cmux.CMux
	lspServer  *lsp.Server
	store      *store.Store
	dbFactory  *dbfactory.DBFactory
	startedTS  int64

	// PG server stoppers.
	stopper []func()

	// stateCfg is the shared in-momory state within the server.
	stateCfg *state.State

	// boot specifies that whether the server boot correctly
	cancel context.CancelFunc
}

// NewServer creates a server.
func NewServer(ctx context.Context, profile *config.Profile) (*Server, error) {
	s := &Server{
		profile:   profile,
		startedTS: time.Now().Unix(),
	}

	// Display config
	slog.Info("-----Config BEGIN-----")
	slog.Info(fmt.Sprintf("mode=%s", profile.Mode))
	slog.Info(fmt.Sprintf("dataDir=%s", profile.DataDir))
	slog.Info(fmt.Sprintf("demo=%v", profile.Demo))
	slog.Info(fmt.Sprintf("instanceRunUUID=%s", profile.DeployID))
	slog.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var pgURL string
	if profile.UseEmbedDB() {
		pgDataDir := path.Join(profile.DataDir, "pgdata")
		if profile.Demo {
			pgDataDir = path.Join(profile.DataDir, "pgdata-demo")
		}

		stopper, err := postgres.StartMetadataInstance(ctx, pgDataDir, profile.DatastorePort, profile.Mode)
		if err != nil {
			return nil, err
		}
		s.stopper = append(s.stopper, stopper)
		pgURL = fmt.Sprintf("host=%s port=%d user=bb database=bb", common.GetPostgresSocketDir(), profile.DatastorePort)
	} else {
		pgURL = profile.PgURL
	}

	// Start Postgres sample servers. It is used for onboarding users without requiring them to
	// configure an external instance.
	if profile.SampleDatabasePort != 0 {
		// Only create batch sample databases in demo mode. For normal mode, user starts from the free version
		// and batch databases are useless because batch requires enterprise license.
		stopper := postgres.StartAllSampleInstances(ctx, profile.DataDir, profile.SampleDatabasePort)
		s.stopper = append(s.stopper, stopper...)
	}

	if profile.Demo {
		if err := demo.LoadDemoData(ctx, pgURL); err != nil {
			return nil, errors.Wrapf(err, "failed to load demo data")
		}
	}
	if err := migrator.MigrateSchema(ctx, pgURL); err != nil {
		return nil, errors.Wrapf(err, "failed to migrate schema")
	}

	// Connect to the instance that stores bytebase's own metadata.
	stores, err := store.New(ctx, pgURL, !profile.HA)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to new store")
	}
	s.store = stores
	sheetManager := sheet.NewManager(stores)

	s.stateCfg, err = state.New()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create state config")
	}

	if err := s.store.BackfillIssueTSVector(ctx); err != nil {
		slog.Warn("failed to backfill issue ts vector", log.BBError(err))
	}

	s.licenseService, err = enterprise.NewLicenseService(profile.Mode, stores)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create license service")
	}
	// Cache the license.
	s.licenseService.LoadSubscription(ctx)

	if err := s.initializeSetting(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}
	secret, err := s.store.GetSecret(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get secret")
	}
	s.iamManager, err = iam.NewManager(stores, s.licenseService)
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, errors.Wrapf(err, "failed to reload iam cache")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create iam manager")
	}
	s.webhookManager = webhook.NewManager(stores, s.iamManager)
	s.dbFactory = dbfactory.New(s.store, s.licenseService)

	// Configure echo server.
	s.echoServer = echo.New()

	// Note: the gateway response modifier takes the token duration on server startup. If the value is changed,
	// the user has to restart the server to take the latest value.
	gatewayModifier := auth.GatewayResponseModifier{Store: s.store}
	mux := grpcruntime.NewServeMux(
		grpcruntime.WithMarshalerOption(grpcruntime.MIMEWildcard, &grpcruntime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{},
			//nolint:forbidigo
			UnmarshalOptions: protojson.UnmarshalOptions{},
		}),
		grpcruntime.WithForwardResponseOption(gatewayModifier.Modify),
		grpcruntime.WithRoutingErrorHandler(func(ctx context.Context, sm *grpcruntime.ServeMux, m grpcruntime.Marshaler, w http.ResponseWriter, r *http.Request, httpStatus int) {
			if httpStatus != http.StatusNotFound {
				grpcruntime.DefaultRoutingErrorHandler(ctx, sm, m, w, r, httpStatus)
				return
			}

			err := &grpcruntime.HTTPStatusError{
				HTTPStatus: httpStatus,
				Err:        status.Errorf(codes.NotFound, "Routing error. Please check the request URI %v", r.RequestURI),
			}

			grpcruntime.DefaultHTTPErrorHandler(ctx, sm, m, w, r, err)
		}),
	)

	s.metricReporter = metricreport.NewReporter(s.store, s.licenseService, s.profile)
	s.schemaSyncer = schemasync.NewSyncer(stores, s.dbFactory, s.profile, s.stateCfg)
	s.approvalRunner = approval.NewRunner(stores, sheetManager, s.dbFactory, s.stateCfg, s.webhookManager, s.licenseService)

	s.taskSchedulerV2 = taskrun.NewSchedulerV2(stores, s.stateCfg, s.webhookManager, profile, s.licenseService)
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_CREATE, taskrun.NewDatabaseCreateExecutor(stores, s.dbFactory, s.schemaSyncer, s.stateCfg, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_SCHEMA_UPDATE, taskrun.NewSchemaUpdateExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_DATA_UPDATE, taskrun.NewDataUpdateExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_EXPORT, taskrun.NewDataExportExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_SCHEMA_UPDATE_GHOST, taskrun.NewSchemaUpdateGhostExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, s.profile))

	s.planCheckScheduler = plancheck.NewScheduler(stores, s.licenseService, s.stateCfg)
	databaseConnectExecutor := plancheck.NewDatabaseConnectExecutor(stores, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseConnect, databaseConnectExecutor)
	statementAdviseExecutor := plancheck.NewStatementAdviseExecutor(stores, sheetManager, s.dbFactory, s.licenseService)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementAdvise, statementAdviseExecutor)
	ghostSyncExecutor := plancheck.NewGhostSyncExecutor(stores, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseGhostSync, ghostSyncExecutor)
	statementReportExecutor := plancheck.NewStatementReportExecutor(stores, sheetManager, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementSummaryReport, statementReportExecutor)

	// Metric reporter
	s.initMetricReporter()

	// Setup the gRPC and grpc-gateway.
	authProvider := auth.New(s.store, secret, s.licenseService, s.stateCfg, s.profile)
	auditProvider := apiv1.NewAuditInterceptor(s.store)
	aclProvider := apiv1.NewACLInterceptor(s.store, secret, s.iamManager, s.profile)
	debugProvider := apiv1.NewDebugInterceptor(s.metricReporter)
	onPanic := func(p any) error {
		stack := stacktrace.TakeStacktrace(20 /* n */, 5 /* skip */)
		// keep a multiline stack
		slog.Error("v1 server panic error", log.BBError(errors.Errorf("error: %v\n%s", p, stack)))
		return status.Errorf(codes.Internal, "error: %v\n%s", p, stack)
	}
	recoveryUnaryInterceptor := recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(onPanic))
	recoveryStreamInterceptor := recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(onPanic))
	grpc.EnableTracing = true
	srvMetrics := grpcprom.NewServerMetrics(grpcprom.WithServerHandlingTimeHistogram())
	s.grpcServer = grpc.NewServer(
		// Override the maximum receiving message size to 100M for uploading large sheets.
		grpc.MaxRecvMsgSize(100*1024*1024),
		grpc.InitialWindowSize(100000000),
		grpc.InitialConnWindowSize(100000000),
		grpc.ChainUnaryInterceptor(
			srvMetrics.UnaryServerInterceptor(),
			debugProvider.DebugInterceptor,
			authProvider.AuthenticationInterceptor,
			aclProvider.ACLInterceptor,
			auditProvider.AuditInterceptor,
			recoveryUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			srvMetrics.StreamServerInterceptor(),
			debugProvider.DebugStreamInterceptor,
			authProvider.AuthenticationStreamInterceptor,
			aclProvider.ACLStreamInterceptor,
			auditProvider.AuditStreamInterceptor,
			recoveryStreamInterceptor,
		),
	)
	reflection.Register(s.grpcServer)

	// LSP server.
	s.lspServer = lsp.NewServer(s.store, profile)

	connectHandlers, err := configureGrpcRouters(ctx, mux, s.grpcServer, s.store, sheetManager, s.dbFactory, s.licenseService, s.profile, s.metricReporter, s.stateCfg, s.schemaSyncer, s.webhookManager, s.iamManager, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to configure gRPC routers")
	}
	directorySyncServer := directorysync.NewService(s.store, s.licenseService, s.iamManager)

	// Configure echo server routes.
	configureEchoRouters(s.echoServer, s.grpcServer, s.lspServer, directorySyncServer, mux, profile, connectHandlers)

	// Configure grpc prometheus metrics.
	if err := prometheus.DefaultRegisterer.Register(srvMetrics); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			return nil, errors.Wrapf(err, "failed to register prometheus metrics")
		}
	}
	srvMetrics.InitializeMetrics(s.grpcServer)

	serverStarted = true
	return s, nil
}

// Run will run the server.
func (s *Server) Run(ctx context.Context, port int) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	// runnerWG waits for all goroutines to complete.
	s.runnerWG.Add(1)
	go s.taskSchedulerV2.Run(ctx, &s.runnerWG)
	s.runnerWG.Add(1)
	go s.schemaSyncer.Run(ctx, &s.runnerWG)
	s.runnerWG.Add(1)
	go s.approvalRunner.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	go s.metricReporter.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	go s.planCheckScheduler.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	mmm := monitor.NewMemoryMonitor(s.profile)
	go mmm.Run(ctx, &s.runnerWG)

	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.muxServer = cmux.New(listener)
	grpcListener := s.muxServer.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpListener := s.muxServer.Match(cmux.HTTP1Fast(), cmux.Any())
	s.echoServer.Listener = httpListener

	go func() {
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			slog.Error("grpc server listen error", log.BBError(err))
		}
	}()
	go func() {
		if err := s.echoServer.StartH2CServer(address, &http2.Server{}); err != nil {
			slog.Error("http server listen error", log.BBError(err))
		}
	}()
	go func() {
		if err := s.muxServer.Serve(); err != nil {
			slog.Error("mux server listen error", log.BBError(err))
		}
	}()

	return nil
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Stopping Bytebase...")
	slog.Info("Stopping web server...")

	// Close the metric reporter
	if s.metricReporter != nil {
		s.metricReporter.Close()
	}

	ctx, cancel := context.WithTimeout(ctx, gracefulShutdownPeriod)
	defer cancel()

	// Cancel the worker
	if s.cancel != nil {
		s.cancel()
	}

	// Shutdown echo
	if s.grpcServer != nil {
		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		t := time.NewTimer(gracefulShutdownPeriod)
		select {
		case <-t.C:
			s.grpcServer.Stop()
		case <-stopped:
			t.Stop()
		}
	}
	if s.echoServer != nil {
		if err := s.echoServer.Shutdown(ctx); err != nil {
			s.echoServer.Logger.Fatal(err)
		}
	}
	if s.muxServer != nil {
		s.muxServer.Close()
	}

	// Wait for all runners to exit.
	s.runnerWG.Wait()

	// Close db connection
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			return err
		}
	}

	// Shutdown postgres instances.
	for _, stopper := range s.stopper {
		stopper()
	}

	return nil
}
