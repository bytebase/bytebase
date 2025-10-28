// Package server implements the API server for Bytebase.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"path"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"

	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/demo"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	runnermigrator "github.com/bytebase/bytebase/backend/runner/migrator"
	"github.com/bytebase/bytebase/backend/runner/monitor"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// webhookAPIPrefix is the API prefix for Bytebase webhook.
	webhookAPIPrefix = "/hook"
	scimAPIPrefix    = "/scim"
	// lspAPI is the API for Bytebase Language Server Protocol.
	lspAPI                 = "/lsp"
	gracefulShutdownPeriod = 10 * time.Second
)

// Server is the Bytebase server.
type Server struct {
	// Asynchronous runners.
	taskSchedulerV2      *taskrun.SchedulerV2
	planCheckScheduler   *plancheck.Scheduler
	metricReporter       *metricreport.Reporter
	schemaSyncer         *schemasync.Syncer
	approvalRunner       *approval.Runner
	exportArchiveCleaner *runnermigrator.ExportArchiveCleaner
	runnerWG             sync.WaitGroup

	webhookManager        *webhook.Manager
	iamManager            *iam.Manager
	sampleInstanceManager *sampleinstance.Manager

	licenseService *enterprise.LicenseService

	profile    *config.Profile
	echoServer *echo.Echo
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

	// Connect to the instance that stores bytebase's own metadata.
	stores, err := store.New(ctx, pgURL, !profile.HA)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to new store")
	}

	if profile.Demo {
		if err := demo.LoadDemoData(ctx, stores.GetDB()); err != nil {
			stores.Close()
			return nil, errors.Wrapf(err, "failed to load demo data")
		}
	}
	if err := migrator.MigrateSchema(ctx, stores.GetDB()); err != nil {
		stores.Close()
		return nil, errors.Wrapf(err, "failed to migrate schema")
	}
	s.store = stores
	sheetManager := sheet.NewManager(stores)

	// Initialize sample instance manager and start sample instances if they exist
	s.sampleInstanceManager = sampleinstance.NewManager(stores, profile)
	if err := s.sampleInstanceManager.StartIfExist(ctx); err != nil {
		slog.Warn("failed to start sample instances", log.BBError(err))
	}

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

	s.metricReporter = metricreport.NewReporter(s.store, s.licenseService, s.profile)
	s.schemaSyncer = schemasync.NewSyncer(stores, s.dbFactory, s.profile, s.stateCfg, s.licenseService)
	s.approvalRunner = approval.NewRunner(stores, sheetManager, s.dbFactory, s.stateCfg, s.webhookManager, s.licenseService)

	s.taskSchedulerV2 = taskrun.NewSchedulerV2(stores, s.stateCfg, s.webhookManager, profile, s.licenseService)
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_CREATE, taskrun.NewDatabaseCreateExecutor(stores, s.dbFactory, s.schemaSyncer, s.stateCfg, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_MIGRATE, taskrun.NewDatabaseMigrateExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_EXPORT, taskrun.NewDataExportExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))
	s.taskSchedulerV2.Register(storepb.Task_DATABASE_SDL, taskrun.NewSchemaDeclareExecutor(stores, s.dbFactory, s.licenseService, s.stateCfg, s.schemaSyncer, profile))

	s.planCheckScheduler = plancheck.NewScheduler(stores, s.licenseService, s.stateCfg)
	databaseConnectExecutor := plancheck.NewDatabaseConnectExecutor(stores, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseConnect, databaseConnectExecutor)
	statementAdviseExecutor := plancheck.NewStatementAdviseExecutor(stores, sheetManager, s.dbFactory, s.licenseService)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementAdvise, statementAdviseExecutor)
	ghostSyncExecutor := plancheck.NewGhostSyncExecutor(stores, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseGhostSync, ghostSyncExecutor)
	statementReportExecutor := plancheck.NewStatementReportExecutor(stores, sheetManager, s.dbFactory)
	s.planCheckScheduler.Register(store.PlanCheckDatabaseStatementSummaryReport, statementReportExecutor)

	// Export archive cleaner
	s.exportArchiveCleaner = runnermigrator.NewExportArchiveCleaner(stores)

	// Metric reporter
	s.initMetricReporter()

	// LSP server.
	s.lspServer = lsp.NewServer(s.store, profile)

	directorySyncServer := directorysync.NewService(s.store, s.licenseService, s.iamManager)

	if err := configureGrpcRouters(ctx, s.echoServer, s.store, sheetManager, s.dbFactory, s.licenseService, s.profile, s.metricReporter, s.stateCfg, s.schemaSyncer, s.webhookManager, s.iamManager, secret, s.sampleInstanceManager); err != nil {
		return nil, errors.Wrapf(err, "failed to configure gRPC routers")
	}
	configureEchoRouters(s.echoServer, s.lspServer, directorySyncServer, profile)

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
	go s.exportArchiveCleaner.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	mmm := monitor.NewMemoryMonitor(s.profile)
	go mmm.Run(ctx, &s.runnerWG)

	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.echoServer.Listener = listener

	go func() {
		if err := s.echoServer.StartH2CServer(address, &http2.Server{}); err != nil {
			slog.Error("http server listen error", log.BBError(err))
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
	if s.echoServer != nil {
		if err := s.echoServer.Shutdown(ctx); err != nil {
			s.echoServer.Logger.Fatal(err)
		}
	}

	// Wait for all runners to exit.
	s.runnerWG.Wait()

	// Close db connection
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			return err
		}
	}

	// Shutdown sample instances
	if s.sampleInstanceManager != nil {
		s.sampleInstanceManager.Stop()
	}

	// Shutdown postgres instances.
	for _, stopper := range s.stopper {
		stopper()
	}

	return nil
}
