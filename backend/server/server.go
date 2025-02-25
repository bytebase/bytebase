// Package server implements the API server for Bytebase.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	"github.com/bytebase/bytebase/backend/api/auth"
	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/gitops"
	"github.com/bytebase/bytebase/backend/api/lsp"
	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/demo"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	enterprisesvc "github.com/bytebase/bytebase/backend/enterprise/service"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/migrator"
	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/mail"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/slowquerysync"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
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
	slowQuerySyncer    *slowquerysync.Syncer
	mailSender         *mail.SlowQueryWeeklyMailSender
	approvalRunner     *approval.Runner
	runnerWG           sync.WaitGroup

	webhookManager *webhook.Manager
	iamManager     *iam.Manager

	licenseService enterprise.LicenseService

	profile      *config.Profile
	echoServer   *echo.Echo
	grpcServer   *grpc.Server
	muxServer    cmux.CMux
	lspServer    *lsp.Server
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
	startedTs    int64
	secret       string

	// Stubs.
	planService    *apiv1.PlanService
	rolloutService *apiv1.RolloutService
	issueService   *apiv1.IssueService

	// MySQL utility binaries
	mysqlBinDir string
	// MongoDB utility binaries
	mongoBinDir string
	// Postgres utility binaries
	pgBinDir string
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
		startedTs: time.Now().Unix(),
	}

	// Display config
	slog.Info("-----Config BEGIN-----")
	slog.Info(fmt.Sprintf("mode=%s", profile.Mode))
	slog.Info(fmt.Sprintf("dataDir=%s", profile.DataDir))
	slog.Info(fmt.Sprintf("resourceDir=%s", profile.ResourceDir))
	slog.Info(fmt.Sprintf("readonly=%t", profile.Readonly))
	slog.Info(fmt.Sprintf("demoName=%s", profile.DemoName))
	slog.Info(fmt.Sprintf("instanceRunUUID=%s", profile.DeployID))
	slog.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var err error
	if err = os.MkdirAll(profile.ResourceDir, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "failed to create directory: %q", profile.ResourceDir)
	}

	// Install mongoutil.
	s.mongoBinDir, err = mongoutil.Install(profile.ResourceDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot install mongo utility binaries")
	}

	// Installs the Postgres and utility binaries and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	s.pgBinDir, err = postgres.Install(profile.ResourceDir)
	if err != nil {
		return nil, err
	}

	var connCfg dbdriver.ConnectionConfig
	if profile.UseEmbedDB() {
		stopper, err := postgres.StartMetadataInstance(profile.DataDir, profile.ResourceDir, s.pgBinDir, profile.PgUser, profile.DemoName, profile.DatastorePort, profile.Mode)
		if err != nil {
			return nil, err
		}
		s.stopper = append(s.stopper, stopper)
		connCfg = store.GetEmbeddedConnectionConfig(profile.DatastorePort, profile.PgUser)
	} else {
		cfg, err := store.GetConnectionConfig(profile.PgURL)
		if err != nil {
			return nil, err
		}
		connCfg = cfg
	}
	connCfg.ReadOnly = profile.Readonly

	// Start Postgres sample servers. It is used for onboarding users without requiring them to
	// configure an external instance.
	if profile.SampleDatabasePort != 0 {
		// Only create batch sample databases in demo mode. For normal mode, user starts from the free version
		// and batch databases are useless because batch requires enterprise license.
		stopper := postgres.StartAllSampleInstances(ctx, s.pgBinDir, profile.DataDir, profile.SampleDatabasePort, profile.DemoName != "")
		s.stopper = append(s.stopper, stopper...)
	}

	// Connect to the instance that stores bytebase's own metadata.
	storeDB := store.NewDB(connCfg, s.pgBinDir, profile.Readonly, profile.Mode)
	// For embedded database, we will create the database if it does not exist.
	if err := storeDB.Open(ctx, profile.UseEmbedDB() /* createDB */); err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, errors.Wrap(err, "cannot open metadb")
	}
	storeInstance, err := store.New(storeDB, profile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to new store")
	}
	if profile.Readonly {
		slog.Info("Database is opened in readonly mode. Skip migration and demo data setup.")
	} else {
		if err := demo.LoadDemoDataIfNeeded(ctx, storeDB, profile.DemoName); err != nil {
			return nil, errors.Wrapf(err, "failed to load demo data")
		}
		if _, err := migrator.MigrateSchema(ctx, storeDB, storeInstance); err != nil {
			return nil, err
		}
	}
	s.store = storeInstance
	s.sheetManager = sheet.NewManager(storeInstance)

	s.stateCfg, err = state.New()
	if err != nil {
		return nil, err
	}

	if err := s.store.BackfillIssueTsVector(ctx); err != nil {
		slog.Warn("failed to backfill issue ts vector", log.BBError(err))
	}

	s.licenseService, err = enterprisesvc.NewLicenseService(profile.Mode, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create license service")
	}
	// Cache the license.
	s.licenseService.LoadSubscription(ctx)

	secret, err := s.getInitSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}
	s.secret = secret
	s.iamManager, err = iam.NewManager(storeInstance, s.licenseService)
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create iam manager")
	}
	s.webhookManager = webhook.NewManager(storeInstance, s.iamManager)
	s.dbFactory = dbfactory.New(
		s.store,
		s.mysqlBinDir,
		s.mongoBinDir,
		s.pgBinDir,
		profile.DataDir,
		s.secret,
	)

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

	s.metricReporter = metricreport.NewReporter(s.store, s.licenseService, s.profile, false)
	s.schemaSyncer = schemasync.NewSyncer(storeInstance, s.dbFactory, s.stateCfg, profile, s.licenseService)
	if !profile.Readonly {
		s.slowQuerySyncer = slowquerysync.NewSyncer(storeInstance, s.dbFactory, s.stateCfg, profile)
		s.mailSender = mail.NewSender(s.store, s.stateCfg, s.iamManager)
		s.approvalRunner = approval.NewRunner(storeInstance, s.sheetManager, s.dbFactory, s.stateCfg, s.webhookManager, s.licenseService)

		s.taskSchedulerV2 = taskrun.NewSchedulerV2(storeInstance, s.stateCfg, s.webhookManager, profile, s.licenseService)
		s.taskSchedulerV2.Register(api.TaskGeneral, taskrun.NewDefaultExecutor())
		s.taskSchedulerV2.Register(api.TaskDatabaseCreate, taskrun.NewDatabaseCreateExecutor(storeInstance, s.dbFactory, s.schemaSyncer, s.stateCfg, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaBaseline, taskrun.NewSchemaBaselineExecutor(storeInstance, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdate, taskrun.NewSchemaUpdateExecutor(storeInstance, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdateSDL, taskrun.NewSchemaUpdateSDLExecutor(storeInstance, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseDataUpdate, taskrun.NewDataUpdateExecutor(storeInstance, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseDataExport, taskrun.NewDataExportExecutor(storeInstance, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdateGhost, taskrun.NewSchemaUpdateGhostExecutor(storeInstance, s.secret, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, s.profile))

		s.planCheckScheduler = plancheck.NewScheduler(storeInstance, s.licenseService, s.stateCfg)
		databaseConnectExecutor := plancheck.NewDatabaseConnectExecutor(storeInstance, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseConnect, databaseConnectExecutor)
		statementAdviseExecutor := plancheck.NewStatementAdviseExecutor(storeInstance, s.sheetManager, s.dbFactory, s.licenseService)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementAdvise, statementAdviseExecutor)
		ghostSyncExecutor := plancheck.NewGhostSyncExecutor(storeInstance, s.secret, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseGhostSync, ghostSyncExecutor)
		statementReportExecutor := plancheck.NewStatementReportExecutor(storeInstance, s.sheetManager, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementSummaryReport, statementReportExecutor)

		// Metric reporter
		s.initMetricReporter()
	}

	// Setup the gRPC and grpc-gateway.
	authProvider := auth.New(s.store, s.secret, s.licenseService, s.stateCfg, s.profile)
	auditProvider := apiv1.NewAuditInterceptor(s.store)
	aclProvider := apiv1.NewACLInterceptor(s.store, s.secret, s.iamManager, s.profile)
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
	s.grpcServer = grpc.NewServer(
		// Override the maximum receiving message size to 100M for uploading large sheets.
		grpc.MaxRecvMsgSize(100*1024*1024),
		grpc.InitialWindowSize(100000000),
		grpc.InitialConnWindowSize(100000000),
		grpc.ChainUnaryInterceptor(
			debugProvider.DebugInterceptor,
			authProvider.AuthenticationInterceptor,
			aclProvider.ACLInterceptor,
			auditProvider.AuditInterceptor,
			recoveryUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
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

	postCreateUser := func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error {
		if profile.TestOnlySkipOnboardingData {
			return nil
		}
		// Only generate onboarding data after the first enduser signup.
		if firstEndUser {
			if profile.SampleDatabasePort != 0 {
				if err := s.generateOnboardingData(ctx, user); err != nil {
					// When running inside docker on mac, we sometimes get database does not exist error.
					// This is due to the docker overlay storage incompatibility with mac OS file system.
					// Onboarding error is not critical, so we just emit an error log.
					slog.Error("failed to prepare onboarding data", log.BBError(err))
				}
			}
		}
		return nil
	}
	releaseService, planService, rolloutService, issueService, sqlService, err := configureGrpcRouters(ctx, mux, s.grpcServer, s.store, s.sheetManager, s.dbFactory, s.licenseService, s.profile, s.metricReporter, s.stateCfg, s.schemaSyncer, s.webhookManager, s.iamManager, postCreateUser, s.secret)
	if err != nil {
		return nil, err
	}
	s.planService, s.rolloutService, s.issueService = planService, rolloutService, issueService
	// GitOps webhook server.
	gitOpsServer := gitops.NewService(s.store, s.licenseService, releaseService, planService, rolloutService, issueService, sqlService, s.sheetManager, profile)
	directorySyncServer := directorysync.NewService(s.store, s.licenseService, s.iamManager)

	// Configure echo server routes.
	configureEchoRouters(s.echoServer, s.grpcServer, s.lspServer, gitOpsServer, directorySyncServer, mux, profile)

	serverStarted = true
	return s, nil
}

// Run will run the server.
func (s *Server) Run(ctx context.Context, port int) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	if !s.profile.Readonly {
		// runnerWG waits for all goroutines to complete.
		s.runnerWG.Add(1)
		go s.taskSchedulerV2.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.schemaSyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.slowQuerySyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.mailSender.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.approvalRunner.Run(ctx, &s.runnerWG)

		s.runnerWG.Add(1)
		go s.metricReporter.Run(ctx, &s.runnerWG)

		s.runnerWG.Add(1)
		go s.planCheckScheduler.Run(ctx, &s.runnerWG)
	}

	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.muxServer = cmux.New(listener)
	grpcListener := s.muxServer.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := s.muxServer.Match(cmux.HTTP1Fast(), cmux.Any())
	s.echoServer.Listener = httpListener

	go func() {
		if err := s.grpcServer.Serve(grpcListener); err != nil {
			slog.Error("grpc server listen error", log.BBError(err))
		}
	}()
	go func() {
		if err := s.echoServer.Start(address); err != nil {
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
		if err := s.store.Close(ctx); err != nil {
			return err
		}
	}

	// Shutdown postgres instances.
	for _, stopper := range s.stopper {
		stopper()
	}

	return nil
}
