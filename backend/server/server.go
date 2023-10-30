// Package server implements the API server for Bytebase.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	grpcRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/api/gitops"
	v1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	enterprisesvc "github.com/bytebase/bytebase/backend/enterprise/service"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/migrator"
	dbdriver "github.com/bytebase/bytebase/backend/plugin/db"
	bbs3 "github.com/bytebase/bytebase/backend/plugin/storage/s3"
	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	"github.com/bytebase/bytebase/backend/resources/mysqlutil"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/mail"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/relay"
	"github.com/bytebase/bytebase/backend/runner/rollbackrun"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/slowquerysync"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	_ "github.com/bytebase/bytebase/docs/openapi" // initial the swagger doc
)

const (
	// webhookAPIPrefix is the API prefix for Bytebase webhook.
	webhookAPIPrefix       = "/hook"
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
	backupRunner       *backuprun.Runner
	rollbackRunner     *rollbackrun.Runner
	approvalRunner     *approval.Runner
	relayRunner        *relay.Runner
	runnerWG           sync.WaitGroup

	activityManager *activity.Manager

	licenseService enterprise.LicenseService

	profile         config.Profile
	e               *echo.Echo
	grpcServer      *grpc.Server
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	startedTs       int64
	secret          string
	errorRecordRing api.ErrorRecordRing

	// Stubs.
	rolloutService *v1.RolloutService
	issueService   *v1.IssueService

	// MySQL utility binaries
	mysqlBinDir string
	// MongoDB utility binaries
	mongoBinDir string
	// Postgres utility binaries
	pgBinDir string
	// PG server stoppers.
	stopper []func()

	s3Client *bbs3.Client

	// stateCfg is the shared in-momory state within the server.
	stateCfg *state.State

	// boot specifies that whether the server boot correctly
	cancel context.CancelFunc
}

// NewServer creates a server.
func NewServer(ctx context.Context, profile config.Profile) (*Server, error) {
	s := &Server{
		profile:         profile,
		startedTs:       time.Now().Unix(),
		errorRecordRing: api.NewErrorRecordRing(),
	}

	// Display config
	slog.Info("-----Config BEGIN-----")
	slog.Info(fmt.Sprintf("mode=%s", profile.Mode))
	slog.Info(fmt.Sprintf("dataDir=%s", profile.DataDir))
	slog.Info(fmt.Sprintf("resourceDir=%s", profile.ResourceDir))
	slog.Info(fmt.Sprintf("readonly=%t", profile.Readonly))
	slog.Info(fmt.Sprintf("demoName=%s", profile.DemoName))
	slog.Info(fmt.Sprintf("backupStorageBackend=%s", profile.BackupStorageBackend))
	slog.Info(fmt.Sprintf("backupBucket=%s", profile.BackupBucket))
	slog.Info(fmt.Sprintf("backupRegion=%s", profile.BackupRegion))
	slog.Info(fmt.Sprintf("backupCredentialFile=%s", profile.BackupCredentialFile))
	slog.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var err error
	// Install mysqlutil
	s.mysqlBinDir, err = mysqlutil.Install(profile.ResourceDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot install mysql utility binaries")
	}

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
		slog.Info("-----Sample Postgres Instance BEGIN-----")
		for i, v := range []string{"test", "prod"} {
			slog.Info(fmt.Sprintf("Start %q sample database sampleDatabasePort=%d", v, profile.SampleDatabasePort))
			stopper, err := postgres.StartSampleInstance(ctx, s.pgBinDir, profile.DataDir, v, profile.SampleDatabasePort+i, profile.Mode)
			if err != nil {
				slog.Error("failed to init sample instance", log.BBError(err))
				continue
			}
			s.stopper = append(s.stopper, stopper)
		}
		slog.Info("-----Sample Postgres Instance END-----")
	}

	// Connect to the instance that stores bytebase's own metadata.
	storeDB := store.NewDB(connCfg, s.pgBinDir, profile.DemoName, profile.Readonly, profile.Version, profile.Mode)
	if err := storeDB.Open(ctx); err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, errors.Wrap(err, "cannot open metadb")
	}
	storeInstance := store.New(storeDB)
	if profile.Readonly {
		slog.Info("Database is opened in readonly mode. Skip migration and demo data setup.")
	} else {
		if _, err := migrator.MigrateSchema(ctx, storeDB, !profile.UseEmbedDB(), s.pgBinDir, profile.DemoName, profile.Version, profile.Mode); err != nil {
			return nil, err
		}
	}
	s.store = storeInstance

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

	secret, externalURL, tokenDuration, err := s.getInitSetting(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}
	s.secret = secret
	s.activityManager = activity.NewManager(storeInstance)
	s.dbFactory = dbfactory.New(s.mysqlBinDir, s.mongoBinDir, s.pgBinDir, profile.DataDir, s.secret)

	// Configure echo server.
	s.e = echo.New()

	// Note: the gateway response modifier takes the external url on server startup. If the external URL is changed,
	// the user has to restart the server to take the latest value.
	gatewayModifier := auth.GatewayResponseModifier{ExternalURL: externalURL, TokenDuration: tokenDuration}
	mux := grpcRuntime.NewServeMux(grpcRuntime.WithForwardResponseOption(gatewayModifier.Modify))

	if profile.BackupBucket != "" {
		credentials, err := bbs3.GetCredentialsFromFile(ctx, profile.BackupCredentialFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get credentials from file")
		}
		s3Client, err := bbs3.NewClient(ctx, profile.BackupRegion, profile.BackupBucket, credentials)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create AWS S3 client")
		}
		s.s3Client = s3Client
	}

	s.metricReporter = metricreport.NewReporter(s.store, s.licenseService, &s.profile, false)
	s.schemaSyncer = schemasync.NewSyncer(storeInstance, s.dbFactory, s.stateCfg, profile, s.licenseService)
	if !profile.Readonly {
		s.slowQuerySyncer = slowquerysync.NewSyncer(storeInstance, s.dbFactory, s.stateCfg, profile)
		s.backupRunner = backuprun.NewRunner(storeInstance, s.dbFactory, s.s3Client, s.stateCfg, &profile)
		s.rollbackRunner = rollbackrun.NewRunner(&profile, storeInstance, s.dbFactory, s.stateCfg)
		s.mailSender = mail.NewSender(s.store, s.stateCfg)
		s.relayRunner = relay.NewRunner(storeInstance, s.activityManager, s.stateCfg)
		s.approvalRunner = approval.NewRunner(storeInstance, s.dbFactory, s.stateCfg, s.activityManager, s.relayRunner, s.licenseService)

		s.taskSchedulerV2 = taskrun.NewSchedulerV2(storeInstance, s.stateCfg, s.activityManager)
		s.taskSchedulerV2.Register(api.TaskGeneral, taskrun.NewDefaultExecutor())
		s.taskSchedulerV2.Register(api.TaskDatabaseCreate, taskrun.NewDatabaseCreateExecutor(storeInstance, s.dbFactory, s.schemaSyncer, s.stateCfg, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaBaseline, taskrun.NewSchemaBaselineExecutor(storeInstance, s.dbFactory, s.activityManager, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdate, taskrun.NewSchemaUpdateExecutor(storeInstance, s.dbFactory, s.activityManager, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdateSDL, taskrun.NewSchemaUpdateSDLExecutor(storeInstance, s.dbFactory, s.activityManager, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseDataUpdate, taskrun.NewDataUpdateExecutor(storeInstance, s.dbFactory, s.activityManager, s.licenseService, s.stateCfg, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseBackup, taskrun.NewDatabaseBackupExecutor(storeInstance, s.dbFactory, s.s3Client, s.stateCfg, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdateGhostSync, taskrun.NewSchemaUpdateGhostSyncExecutor(storeInstance, s.stateCfg, s.secret))
		s.taskSchedulerV2.Register(api.TaskDatabaseSchemaUpdateGhostCutover, taskrun.NewSchemaUpdateGhostCutoverExecutor(storeInstance, s.dbFactory, s.activityManager, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseRestorePITRRestore, taskrun.NewPITRRestoreExecutor(storeInstance, s.dbFactory, s.s3Client, s.schemaSyncer, s.stateCfg, profile))
		s.taskSchedulerV2.Register(api.TaskDatabaseRestorePITRCutover, taskrun.NewPITRCutoverExecutor(storeInstance, s.dbFactory, s.schemaSyncer, s.stateCfg, s.backupRunner, s.activityManager, profile))

		s.planCheckScheduler = plancheck.NewScheduler(storeInstance, s.licenseService, s.stateCfg)
		databaseConnectExecutor := plancheck.NewDatabaseConnectExecutor(storeInstance, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseConnect, databaseConnectExecutor)
		statementTypeExecutor := plancheck.NewStatementTypeExecutor(storeInstance, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementType, statementTypeExecutor)
		statementAdviseExecutor := plancheck.NewStatementAdviseExecutor(storeInstance, s.dbFactory, s.licenseService)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementAdvise, statementAdviseExecutor)
		ghostSyncExecutor := plancheck.NewGhostSyncExecutor(storeInstance, s.secret)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseGhostSync, ghostSyncExecutor)
		pitrMySQLExecutor := plancheck.NewPITRMySQLExecutor(storeInstance, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabasePITRMySQL, pitrMySQLExecutor)
		statementReportExecutor := plancheck.NewStatementReportExecutor(storeInstance, s.dbFactory)
		s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementSummaryReport, statementReportExecutor)

		// Metric reporter
		s.initMetricReporter()
	}

	// Setup the gRPC and grpc-gateway.
	authProvider := auth.New(s.store, s.secret, tokenDuration, s.licenseService, s.stateCfg, profile.Mode)
	aclProvider := v1.NewACLInterceptor(s.store, s.secret, s.licenseService, profile.Mode)
	debugProvider := v1.NewDebugInterceptor(&s.errorRecordRing, &profile, s.metricReporter)
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
			recoveryUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			debugProvider.DebugStreamInterceptor,
			authProvider.AuthenticationStreamInterceptor,
			aclProvider.ACLStreamInterceptor,
			recoveryStreamInterceptor,
		),
	)
	configureEchoRouters(s.e, s.grpcServer, mux)
	postCreateUser := func(ctx context.Context, user *store.UserMessage, firstEndUser bool) error {
		if profile.TestOnlySkipOnboardingData {
			return nil
		}
		// Only generate onboarding data after the first enduser signup.
		if firstEndUser {
			if profile.SampleDatabasePort != 0 {
				if err := s.generateOnboardingData(ctx, user); err != nil {
					return status.Errorf(codes.Internal, "failed to prepare onboarding data, error: %v", err)
				}
			}
		}
		return nil
	}
	rolloutService, issueService, err := configureGrpcRouters(ctx, mux, s.grpcServer, s.store, s.dbFactory, s.licenseService, &s.profile, s.metricReporter, s.stateCfg, s.schemaSyncer, s.activityManager, s.backupRunner, s.relayRunner, s.planCheckScheduler, postCreateUser, s.secret, &s.errorRecordRing, tokenDuration)
	if err != nil {
		return nil, err
	}
	s.rolloutService, s.issueService = rolloutService, issueService

	webhookGroup := s.e.Group(webhookAPIPrefix)
	gitOpsService := gitops.NewService(s.store, s.dbFactory, s.activityManager, s.stateCfg, s.licenseService, rolloutService, issueService)
	gitOpsService.RegisterWebhookRoutes(webhookGroup)

	reflection.Register(s.grpcServer)

	serverStarted = true
	return s, nil
}

// Run will run the server.
func (s *Server) Run(ctx context.Context, port int) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	if !s.profile.Readonly {
		// runnerWG waits for all goroutines to complete.
		if err := s.taskSchedulerV2.ClearRunningTaskRuns(ctx); err != nil {
			return errors.Wrap(err, "failed to clear existing RUNNING tasks before starting the task scheduler")
		}
		s.runnerWG.Add(1)
		go s.taskSchedulerV2.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.schemaSyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.slowQuerySyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.mailSender.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.backupRunner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.rollbackRunner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.approvalRunner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.relayRunner.Run(ctx, &s.runnerWG)

		s.runnerWG.Add(1)
		go s.metricReporter.Run(ctx, &s.runnerWG)

		s.runnerWG.Add(1)
		go s.planCheckScheduler.Run(ctx, &s.runnerWG)
	}

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port+1))
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(listen); err != nil {
			slog.Error("grpc server listen error", log.BBError(err))
		}
	}()
	return s.e.Start(fmt.Sprintf(":%d", port))
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
	if s.e != nil {
		if err := s.e.Shutdown(ctx); err != nil {
			s.e.Logger.Fatal(err)
		}
	}
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
