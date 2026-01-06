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

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"

	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	"github.com/bytebase/bytebase/backend/api/mcp"
	"github.com/bytebase/bytebase/backend/api/oauth2"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/component/webhook"
	"github.com/bytebase/bytebase/backend/demo"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/approval"
	"github.com/bytebase/bytebase/backend/runner/cleaner"
	"github.com/bytebase/bytebase/backend/runner/monitor"
	"github.com/bytebase/bytebase/backend/runner/notifylistener"
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
	taskScheduler      *taskrun.Scheduler
	planCheckScheduler *plancheck.Scheduler
	schemaSyncer       *schemasync.Syncer
	approvalRunner     *approval.Runner
	notifyListener     *notifylistener.Listener
	dataCleaner        *cleaner.DataCleaner
	runnerWG           sync.WaitGroup

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

	// bus is the message bus for inter-component communication within the server.
	bus *bus.Bus

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
	sheetManager := sheet.NewManager()

	// Initialize sample instance manager and start sample instances if they exist
	s.sampleInstanceManager = sampleinstance.NewManager(stores, profile)
	if err := s.sampleInstanceManager.StartIfExist(ctx); err != nil {
		slog.Warn("failed to start sample instances", log.BBError(err))
	}

	s.bus, err = bus.New()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create message bus")
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

	// Settings are now initialized in the database schema (LATEST.sql)
	systemSetting, err := s.store.GetSystemSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get system setting")
	}
	secret := systemSetting.AuthSecret
	s.iamManager, err = iam.NewManager(stores, s.licenseService)
	if err := s.iamManager.ReloadCache(ctx); err != nil {
		return nil, errors.Wrapf(err, "failed to reload iam cache")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create iam manager")
	}
	s.webhookManager = webhook.NewManager(stores, profile)
	s.dbFactory = dbfactory.New(s.store, s.licenseService)

	// Configure echo server.
	s.echoServer = echo.New()

	s.schemaSyncer = schemasync.NewSyncer(stores, s.dbFactory)
	s.approvalRunner = approval.NewRunner(stores, s.bus, s.webhookManager, s.licenseService)

	s.taskScheduler = taskrun.NewScheduler(stores, s.bus, s.webhookManager, profile)
	s.taskScheduler.Register(storepb.Task_DATABASE_CREATE, taskrun.NewDatabaseCreateExecutor(stores, s.dbFactory, s.schemaSyncer))
	s.taskScheduler.Register(storepb.Task_DATABASE_MIGRATE, taskrun.NewDatabaseMigrateExecutor(stores, s.dbFactory, s.bus, s.schemaSyncer, profile))
	s.taskScheduler.Register(storepb.Task_DATABASE_EXPORT, taskrun.NewDataExportExecutor(stores, s.dbFactory, s.licenseService))

	combinedExecutor := plancheck.NewCombinedExecutor(stores, sheetManager, s.dbFactory)
	s.planCheckScheduler = plancheck.NewScheduler(stores, s.bus, combinedExecutor)
	s.notifyListener = notifylistener.NewListener(stores.GetDB(), s.bus)

	// Data cleaner
	s.dataCleaner = cleaner.NewDataCleaner(stores)

	// LSP server.
	s.lspServer = lsp.NewServer(s.store, profile, secret, s.bus, s.iamManager, s.licenseService)

	directorySyncServer := directorysync.NewService(s.store, s.licenseService, s.iamManager, profile)
	oauth2Service := oauth2.NewService(stores, profile, secret)
	mcpServer, err := mcp.NewServer(stores, profile, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create MCP server")
	}

	if err := configureGrpcRouters(ctx, s.echoServer, s.store, sheetManager, s.dbFactory, s.licenseService, s.profile, s.bus, s.schemaSyncer, s.webhookManager, s.iamManager, secret, s.sampleInstanceManager); err != nil {
		return nil, errors.Wrapf(err, "failed to configure gRPC routers")
	}
	configureEchoRouters(s.echoServer, s.lspServer, directorySyncServer, oauth2Service, mcpServer, profile)

	serverStarted = true
	return s, nil
}

// Run will run the server.
func (s *Server) Run(ctx context.Context, port int) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	// runnerWG waits for all goroutines to complete.
	s.runnerWG.Add(1)
	go s.taskScheduler.Run(ctx, &s.runnerWG)
	s.runnerWG.Add(1)
	go s.schemaSyncer.Run(ctx, &s.runnerWG)
	s.runnerWG.Add(1)
	go s.approvalRunner.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	go s.planCheckScheduler.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	go s.dataCleaner.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	go s.notifyListener.Run(ctx, &s.runnerWG)

	s.runnerWG.Add(1)
	mmm := monitor.NewMemoryMonitor(s.profile)
	go mmm.Run(ctx, &s.runnerWG)

	// Check workspace setting and set audit logger runtime flag
	workspaceProfile, err := s.store.GetWorkspaceProfileSetting(ctx)
	if err == nil && workspaceProfile.GetEnableAuditLogStdout() {
		// Validate license before enabling (prevents usage after license downgrade/expiry)
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_AUDIT_LOG); err != nil {
			slog.Warn("audit logging enabled in workspace settings but license insufficient, keeping disabled",
				log.BBError(err))
		} else {
			s.profile.RuntimeEnableAuditLogStdout.Store(true)
			slog.Info("audit logging to stdout enabled via workspace setting")
		}
	}

	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.echoServer.Listener = listener

	go func() {
		if err := s.echoServer.StartH2CServer(address, &http2.Server{}); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("http server listen error", log.BBError(err))
			}
		}
	}()

	return nil
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Stopping Bytebase...")
	slog.Info("Stopping web server...")

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
