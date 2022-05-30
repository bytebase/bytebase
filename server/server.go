package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	// embed will embeds the acl policy.
	_ "embed"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	scas "github.com/qiangmzsx/string-adapter/v2"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	enterpriseService "github.com/bytebase/bytebase/enterprise/service"
	"github.com/bytebase/bytebase/metric"
	metricCollector "github.com/bytebase/bytebase/metric/collector"
	metricIdentifier "github.com/bytebase/bytebase/metric/identifier"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/store"
)

// Server is the Bytebase server.
type Server struct {
	// Asynchronous runners.
	TaskScheduler      *TaskScheduler
	TaskCheckScheduler *TaskCheckScheduler
	MetricReporter     *MetricReporter
	SchemaSyncer       *SchemaSyncer
	BackupRunner       *BackupRunner
	AnomalyScanner     *AnomalyScanner
	runnerWG           sync.WaitGroup

	ActivityManager *ActivityManager

	LicenseService enterpriseAPI.LicenseService
	subscription   enterpriseAPI.Subscription

	profile   Profile
	e         *echo.Echo
	mysqlutil *mysqlutil.Instance
	metaDB    *store.MetadataDB
	db        *store.DB
	store     *store.Store
	l         *zap.Logger
	lvl       *zap.AtomicLevel
	startedTs int64
	secret    string

	// boot specifies that whether the server boot correctly
	cancel context.CancelFunc
}

//go:embed acl_casbin_model.conf
var casbinModel string

//go:embed acl_casbin_policy_owner.csv
var casbinOwnerPolicy string

//go:embed acl_casbin_policy_dba.csv
var casbinDBAPolicy string

//go:embed acl_casbin_policy_developer.csv
var casbinDeveloperPolicy string

// NewServer creates a server.
func NewServer(ctx context.Context, prof Profile, logger *zap.Logger, loggerLevel *zap.AtomicLevel) (*Server, error) {
	s := &Server{
		profile:   prof,
		l:         logger,
		lvl:       loggerLevel,
		startedTs: time.Now().Unix(),
	}

	// Display config
	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", prof.Mode)
	fmt.Printf("server=%s:%d\n", prof.BackendHost, prof.BackendPort)
	fmt.Printf("datastore=%s:%d\n", prof.BackendHost, prof.DatastorePort)
	fmt.Printf("frontend=%s:%d\n", prof.FrontendHost, prof.FrontendPort)
	fmt.Printf("demoDataDir=%s\n", prof.DemoDataDir)
	fmt.Printf("readonly=%t\n", prof.Readonly)
	fmt.Printf("demo=%t\n", prof.Demo)
	fmt.Printf("debug=%t\n", prof.Debug)
	fmt.Printf("dataDir=%s\n", prof.DataDir)
	fmt.Println("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var err error

	resourceDir := common.GetResourceDir(prof.DataDir)
	// Install mysqlutil
	mysqlutilIns, err := mysqlutil.Install(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("cannot install mysqlbinlog binary, error: %w", err)
	}
	s.mysqlutil = mysqlutilIns

	// New MetadataDB instance.
	if prof.useEmbedDB() {
		s.metaDB, err = store.NewMetadataDBWithEmbedPg(logger, prof.PgUser, prof.DataDir, prof.DemoDataDir, prof.Mode)
	} else {
		s.metaDB, err = store.NewMetadataDBWithExternalPg(logger, prof.PgURL, prof.DemoDataDir, prof.Mode)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot create MetadataDB instance, error: %w", err)
	}

	// New store.DB instance that represents the db connection.
	storeDB, err := s.metaDB.Connect(prof.DatastorePort, prof.Readonly, prof.Version)
	if err != nil {
		return nil, fmt.Errorf("cannot new db: %w", err)
	}
	s.db = storeDB

	// Open the database that stores bytebase's own metadata connection.
	if err = storeDB.Open(ctx); err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	cacheService := NewCacheService()
	storeInstance := store.New(logger, storeDB, cacheService)
	s.store = storeInstance

	config, err := s.initSetting(ctx, storeInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}
	s.secret = config.secret

	e := echo.New()
	e.Debug = prof.Debug
	e.HideBanner = true
	e.HidePort = true

	// Disallow to be embedded in an iFrame.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XFrameOptions: "DENY",
	}))

	embedFrontend(logger, e)
	s.e = e

	if !prof.Readonly {
		// Task scheduler
		taskScheduler := NewTaskScheduler(logger, s)

		defaultExecutor := NewDefaultTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskGeneral), defaultExecutor)

		createDBExecutor := NewDatabaseCreateTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseCreate), createDBExecutor)

		schemaUpdateExecutor := NewSchemaUpdateTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseSchemaUpdate), schemaUpdateExecutor)

		dataUpdateExecutor := NewDataUpdateTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseDataUpdate), dataUpdateExecutor)

		backupDBExecutor := NewDatabaseBackupTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseBackup), backupDBExecutor)

		restoreDBExecutor := NewDatabaseRestoreTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseRestore), restoreDBExecutor)

		schemaUpdateGhostSyncExecutor := NewSchemaUpdateGhostSyncTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseSchemaUpdateGhostSync), schemaUpdateGhostSyncExecutor)

		schemaUpdateGhostCutoverExecutor := NewSchemaUpdateGhostCutoverTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseSchemaUpdateGhostCutover), schemaUpdateGhostCutoverExecutor)

		schemaUpdateGhostDropOriginalTableExecutor := NewSchemaUpdateGhostDropOriginalTableTaskExecutor(logger)
		taskScheduler.Register(string(api.TaskDatabaseSchemaUpdateGhostDropOriginalTable), schemaUpdateGhostDropOriginalTableExecutor)

		pitrRestoreExecutor := NewPITRRestoreTaskExecutor(logger, s.mysqlutil)
		taskScheduler.Register(string(api.TaskDatabasePITRRestore), pitrRestoreExecutor)

		pitrCutoverExecutor := NewPITRCutoverTaskExecutor(logger, s.mysqlutil)
		taskScheduler.Register(string(api.TaskDatabasePITRCutover), pitrCutoverExecutor)

		s.TaskScheduler = taskScheduler

		// Task check scheduler
		taskCheckScheduler := NewTaskCheckScheduler(logger, s)

		statementSimpleExecutor := NewTaskCheckStatementAdvisorSimpleExecutor(logger)
		taskCheckScheduler.Register(string(api.TaskCheckDatabaseStatementFakeAdvise), statementSimpleExecutor)
		taskCheckScheduler.Register(string(api.TaskCheckDatabaseStatementSyntax), statementSimpleExecutor)

		statementCompositeExecutor := NewTaskCheckStatementAdvisorCompositeExecutor(logger)
		taskCheckScheduler.Register(string(api.TaskCheckDatabaseStatementAdvise), statementCompositeExecutor)

		databaseConnectExecutor := NewTaskCheckDatabaseConnectExecutor(logger)
		taskCheckScheduler.Register(string(api.TaskCheckDatabaseConnect), databaseConnectExecutor)

		migrationSchemaExecutor := NewTaskCheckMigrationSchemaExecutor(logger)
		taskCheckScheduler.Register(string(api.TaskCheckInstanceMigrationSchema), migrationSchemaExecutor)

		timingExecutor := NewTaskCheckTimingExecutor(logger)
		taskCheckScheduler.Register(string(api.TaskCheckGeneralEarliestAllowedTime), timingExecutor)

		s.TaskCheckScheduler = taskCheckScheduler

		// Schema syncer
		s.SchemaSyncer = NewSchemaSyncer(logger, s)

		// Backup runner
		s.BackupRunner = NewBackupRunner(logger, s, prof.BackupRunnerInterval)

		// Anomaly scanner
		s.AnomalyScanner = NewAnomalyScanner(logger, s)

		// Metric reporter
		s.initMetricReporter(config.workspaceID)
	}

	// Middleware
	if prof.Mode == common.ReleaseModeDev || prof.Debug {
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Skipper: func(c echo.Context) bool {
				return !common.HasPrefixes(c.Path(), "/api", "/hook")
			},
			Format: `{"time":"${time_rfc3339}",` +
				`"method":"${method}","uri":"${uri}",` +
				`"status":${status},"error":"${error}"}` + "\n",
		}))
	}
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return recoverMiddleware(logger, next)
	})

	webhookGroup := e.Group("/hook")
	s.registerWebhookRoutes(webhookGroup)

	apiGroup := e.Group("/api")

	apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWTMiddleware(logger, s.store, next, prof.Mode, config.secret)
	})

	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, err
	}
	sa := scas.NewAdapter(strings.Join([]string{casbinOwnerPolicy, casbinDBAPolicy, casbinDeveloperPolicy}, "\n"))
	ce, err := casbin.NewEnforcer(m, sa)
	if err != nil {
		return nil, err
	}
	apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return aclMiddleware(logger, s, ce, next, prof.Readonly)
	})
	s.registerDebugRoutes(apiGroup)
	s.registerSettingRoutes(apiGroup)
	s.registerActuatorRoutes(apiGroup)
	s.registerAuthRoutes(apiGroup)
	s.registerOAuthRoutes(apiGroup)
	s.registerPrincipalRoutes(apiGroup)
	s.registerMemberRoutes(apiGroup)
	s.registerPolicyRoutes(apiGroup)
	s.registerProjectRoutes(apiGroup)
	s.registerProjectWebhookRoutes(apiGroup)
	s.registerProjectMemberRoutes(apiGroup)
	s.registerEnvironmentRoutes(apiGroup)
	s.registerInstanceRoutes(apiGroup)
	s.registerDatabaseRoutes(apiGroup)
	s.registerIssueRoutes(apiGroup)
	s.registerIssueSubscriberRoutes(apiGroup)
	s.registerTaskRoutes(apiGroup)
	s.registerActivityRoutes(apiGroup)
	s.registerInboxRoutes(apiGroup)
	s.registerBookmarkRoutes(apiGroup)
	s.registerSQLRoutes(apiGroup)
	s.registerVCSRoutes(apiGroup)
	s.registerLabelRoutes(apiGroup)
	s.registerSubscriptionRoutes(apiGroup)
	s.registerSheetRoutes(apiGroup)
	s.registerSheetOrganizerRoutes(apiGroup)
	// Register healthz endpoint.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!\n")
	})
	// Register pprof endpoints.
	registerPProfEndpoints(e)

	allRoutes, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		return nil, err
	}

	s.ActivityManager = NewActivityManager(s, storeInstance)
	s.LicenseService, err = enterpriseService.NewLicenseService(logger, prof.Mode, s.store)
	if err != nil {
		return nil, fmt.Errorf("failed to create license service, error: %w", err)
	}

	s.initSubscription()

	logger.Debug(fmt.Sprintf("All registered routes: %v", string(allRoutes)))
	serverStarted = true
	return s, nil
}

// initSubscription will initial the subscription cache in memory.
func (server *Server) initSubscription() {
	server.subscription = server.loadSubscription()
}

// initMetricReporter will initial the metric scheduler.
func (server *Server) initMetricReporter(workspaceID string) {
	enabled := server.profile.Mode == common.ReleaseModeProd && !server.profile.Demo
	if enabled {
		workspace := &api.Workspace{
			ID:      workspaceID,
			Version: server.profile.Version,
		}
		identifier := metricIdentifier.NewIdentifier(server.l, server.store, workspace, &server.subscription)
		metricReporter := NewMetricReporter(server.l, workspaceID, server.profile.MetricConnectionKey, identifier)

		metricReporter.Register(metric.InstanceCountMetricName, metricCollector.NewInstanceCountCollector(server.l, server.store))
		metricReporter.Register(metric.IssueCountMetricName, metricCollector.NewIssueCountCollector(server.l, server.store))
		metricReporter.Register(metric.ProjectCountMetricName, metricCollector.NewProjectCountCollector(server.l, server.store))
		metricReporter.Register(metric.PolicyCountMetricName, metricCollector.NewPolicyCountCollector(server.l, server.store))
		metricReporter.Register(metric.TaskCountMetricName, metricCollector.NewTaskCountCollector(server.l, server.store))
		metricReporter.Register(metric.DatabaseCountMetricName, metricCollector.NewDatabaseCountCollector(server.l, server.store))
		server.MetricReporter = metricReporter
	}
}

func (server *Server) initSetting(ctx context.Context, store *store.Store) (*config, error) {
	// initial branding
	_, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding logo image in base64 string format.",
	})
	if err != nil {
		return nil, err
	}

	conf := &config{}

	// initial JWT token
	authSetting, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingAuthSecret,
		Value:       common.RandomString(secreatLength),
		Description: "Random string used to sign the JWT auth token.",
	})
	if err != nil {
		return nil, err
	}
	conf.secret = authSetting.Value

	// initial workspace
	workspaceSetting, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingWorkspaceID,
		Value:       uuid.New().String(),
		Description: "The workspace identifier",
	})
	if err != nil {
		return nil, err
	}
	conf.workspaceID = workspaceSetting.Value

	// initial license
	if _, err = store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingEnterpriseLicense,
		Value:       "",
		Description: "Enterprise license",
	}); err != nil {
		return nil, err
	}

	return conf, nil
}

// Run will run the server.
func (server *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	server.cancel = cancel
	if !server.profile.Readonly {
		// runnerWG waits for all goroutines to complete.
		go server.TaskScheduler.Run(ctx, &server.runnerWG)
		server.runnerWG.Add(1)
		go server.TaskCheckScheduler.Run(ctx, &server.runnerWG)
		server.runnerWG.Add(1)
		go server.SchemaSyncer.Run(ctx, &server.runnerWG)
		server.runnerWG.Add(1)
		go server.BackupRunner.Run(ctx, &server.runnerWG)
		server.runnerWG.Add(1)
		go server.AnomalyScanner.Run(ctx, &server.runnerWG)
		server.runnerWG.Add(1)

		if server.MetricReporter != nil {
			go server.MetricReporter.Run(ctx, &server.runnerWG)
			server.runnerWG.Add(1)
		}
	}

	// Sleep for 1 sec to make sure port is released between runs.
	time.Sleep(time.Duration(1) * time.Second)

	return server.e.Start(fmt.Sprintf(":%d", server.profile.BackendPort))
}

// Shutdown will shut down the server.
func (server *Server) Shutdown(ctx context.Context) error {
	server.l.Info("Trying to stop Bytebase ....")
	server.l.Info("Trying to gracefully shutdown server")

	// Close the metric reporter
	if server.MetricReporter != nil {
		server.MetricReporter.Close()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Cancel the worker
	if server.cancel != nil {
		server.cancel()
	}

	// Shutdown echo
	if server.e != nil {
		if err := server.e.Shutdown(ctx); err != nil {
			server.e.Logger.Fatal(err)
		}
	}

	// Wait for all runners to exit.
	server.runnerWG.Wait()

	// Close db connection
	if server.db != nil {
		server.l.Info("Trying to close database connections")
		if err := server.db.Close(); err != nil {
			return err
		}
	}

	// Shutdown postgres server if embed.
	if server.metaDB != nil {
		server.metaDB.Close()
	}
	server.l.Info("Bytebase stopped properly")

	return nil
}

// GetEcho returns the echo server.
func (server *Server) GetEcho() *echo.Echo {
	return server.e
}
