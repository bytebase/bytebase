package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	// embed will embeds the acl policy.
	_ "embed"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	scas "github.com/qiangmzsx/string-adapter/v2"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	_ "github.com/bytebase/bytebase/docs/openapi" // initial the swagger doc
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	enterpriseService "github.com/bytebase/bytebase/enterprise/service"
	"github.com/bytebase/bytebase/metric"
	metricCollector "github.com/bytebase/bytebase/metric/collector"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/store"
)

// openAPIPrefix is the API prefix for Bytebase OpenAPI
const openAPIPrefix = "/v1"

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

	profile       Profile
	e             *echo.Echo
	mysqlutil     mysqlutil.Instance
	pgInstanceDir string
	metaDB        *store.MetadataDB
	store         *store.Store
	startedTs     int64
	secret        string

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

// Use following cmd to generate swagger doc
// swag init -g ./server.go -d ./server --output docs/openapi --parseDependency

// @title Bytebase OpenAPI
// @version 1.0
// @description The OpenAPI for bytebase.
// @termsOfService https://www.bytebase.com/terms

// @contact.name API Support
// @contact.url https://github.com/bytebase/bytebase/
// @contact.email support@bytebase.com

// @license.name MIT
// @license.url https://github.com/bytebase/bytebase/blob/main/LICENSE

// @host localhost:8080
// @BasePath /v1/
// @schemes http

// NewServer creates a server.
func NewServer(ctx context.Context, prof Profile) (*Server, error) {
	s := &Server{
		profile:   prof,
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
	s.mysqlutil = *mysqlutilIns

	// Install Postgres.
	pgDataDir := common.GetPostgresDataDir(prof.DataDir)
	log.Info("-----Embedded Postgres Config BEGIN-----")
	log.Info(fmt.Sprintf("resourceDir=%s\n", resourceDir))
	log.Info(fmt.Sprintf("pgdataDir=%s\n", pgDataDir))
	log.Info("-----Embedded Postgres Config END-----")
	log.Info("Preparing embedded PostgreSQL instance...")
	// Installs the Postgres binary and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	log.Info(fmt.Sprintf("Installing Postgres OS %q Arch %q\n", runtime.GOOS, runtime.GOARCH))
	pgInstance, err := postgres.Install(resourceDir, pgDataDir, prof.PgUser)
	if err != nil {
		return nil, err
	}
	s.pgInstanceDir = pgInstance.BaseDir

	// New MetadataDB instance.
	if prof.useEmbedDB() {
		s.metaDB = store.NewMetadataDBWithEmbedPg(pgInstance, prof.PgUser, prof.DataDir, prof.DemoDataDir, prof.Mode)
	} else {
		s.metaDB = store.NewMetadataDBWithExternalPg(pgInstance, prof.PgURL, prof.DemoDataDir, prof.Mode)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot create MetadataDB instance, error: %w", err)
	}

	// New store.DB instance that represents the db connection.
	storeDB, err := s.metaDB.Connect(prof.DatastorePort, prof.Readonly, prof.Version)
	if err != nil {
		return nil, fmt.Errorf("cannot new db: %w", err)
	}

	// Open the database that stores bytebase's own metadata connection.
	if err = storeDB.Open(ctx); err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	cacheService := NewCacheService()
	storeInstance := store.New(storeDB, cacheService)
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

	embedFrontend(e)
	s.e = e

	if !prof.Readonly {
		// Task scheduler
		taskScheduler := NewTaskScheduler(s)

		defaultExecutor := NewDefaultTaskExecutor()
		taskScheduler.Register(api.TaskGeneral, defaultExecutor)

		createDBExecutor := NewDatabaseCreateTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseCreate, createDBExecutor)

		schemaUpdateExecutor := NewSchemaUpdateTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseSchemaUpdate, schemaUpdateExecutor)

		dataUpdateExecutor := NewDataUpdateTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseDataUpdate, dataUpdateExecutor)

		backupDBExecutor := NewDatabaseBackupTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseBackup, backupDBExecutor)

		restoreDBExecutor := NewDatabaseRestoreTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseRestore, restoreDBExecutor)

		schemaUpdateGhostSyncExecutor := NewSchemaUpdateGhostSyncTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostSync, schemaUpdateGhostSyncExecutor)

		schemaUpdateGhostCutoverExecutor := NewSchemaUpdateGhostCutoverTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostCutover, schemaUpdateGhostCutoverExecutor)

		schemaUpdateGhostDropOriginalTableExecutor := NewSchemaUpdateGhostDropOriginalTableTaskExecutor()
		taskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostDropOriginalTable, schemaUpdateGhostDropOriginalTableExecutor)

		pitrRestoreExecutor := NewPITRRestoreTaskExecutor(s.mysqlutil)
		taskScheduler.Register(api.TaskDatabasePITRRestore, pitrRestoreExecutor)

		pitrCutoverExecutor := NewPITRCutoverTaskExecutor(s.mysqlutil)
		taskScheduler.Register(api.TaskDatabasePITRCutover, pitrCutoverExecutor)

		s.TaskScheduler = taskScheduler

		// Task check scheduler
		taskCheckScheduler := NewTaskCheckScheduler(s)

		statementSimpleExecutor := NewTaskCheckStatementAdvisorSimpleExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementFakeAdvise, statementSimpleExecutor)
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementSyntax, statementSimpleExecutor)

		statementCompositeExecutor := NewTaskCheckStatementAdvisorCompositeExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementAdvise, statementCompositeExecutor)

		databaseConnectExecutor := NewTaskCheckDatabaseConnectExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseConnect, databaseConnectExecutor)

		migrationSchemaExecutor := NewTaskCheckMigrationSchemaExecutor()
		taskCheckScheduler.Register(api.TaskCheckInstanceMigrationSchema, migrationSchemaExecutor)

		ghostSyncExecutor := NewTaskCheckGhostSyncExecutor()
		taskCheckScheduler.Register(api.TaskCheckGhostSync, ghostSyncExecutor)

		timingExecutor := NewTaskCheckTimingExecutor()
		taskCheckScheduler.Register(api.TaskCheckGeneralEarliestAllowedTime, timingExecutor)

		s.TaskCheckScheduler = taskCheckScheduler

		// Schema syncer
		s.SchemaSyncer = NewSchemaSyncer(s)

		// Backup runner
		s.BackupRunner = NewBackupRunner(s, prof.BackupRunnerInterval)

		// Anomaly scanner
		s.AnomalyScanner = NewAnomalyScanner(s)

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
	e.Use(recoverMiddleware)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	webhookGroup := e.Group("/hook")
	s.registerWebhookRoutes(webhookGroup)

	apiGroup := e.Group("/api")
	openAPIGroup := e.Group(openAPIPrefix)
	openAPIGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return openAPIMetricMiddleware(s, next)
	})

	apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWTMiddleware(s.store, next, prof.Mode, config.secret)
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
		return aclMiddleware(s, ce, next, prof.Readonly)
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
	s.registerStageRoutes(apiGroup)
	s.registerActivityRoutes(apiGroup)
	s.registerInboxRoutes(apiGroup)
	s.registerBookmarkRoutes(apiGroup)
	s.registerSQLRoutes(apiGroup)
	s.registerVCSRoutes(apiGroup)
	s.registerLabelRoutes(apiGroup)
	s.registerSubscriptionRoutes(apiGroup)
	s.registerSheetRoutes(apiGroup)
	s.registerSheetOrganizerRoutes(apiGroup)
	s.registerOpenAPIRoutes(openAPIGroup)

	// Register healthz endpoint.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!\n")
	})
	// Register pprof endpoints.
	pprof.Register(e)
	// Register prometheus metrics endpoint.
	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	allRoutes, err := json.MarshalIndent(e.Routes(), "", "  ")
	if err != nil {
		return nil, err
	}

	s.ActivityManager = NewActivityManager(s, storeInstance)
	s.LicenseService, err = enterpriseService.NewLicenseService(prof.Mode, s.store)
	if err != nil {
		return nil, fmt.Errorf("failed to create license service, error: %w", err)
	}

	s.initSubscription()

	log.Debug(fmt.Sprintf("All registered routes: %v", string(allRoutes)))
	serverStarted = true
	return s, nil
}

// initSubscription will initial the subscription cache in memory.
func (s *Server) initSubscription() {
	s.subscription = s.loadSubscription()
}

// initMetricReporter will initial the metric scheduler.
func (s *Server) initMetricReporter(workspaceID string) {
	enabled := s.profile.Mode == common.ReleaseModeProd && !s.profile.Demo
	if enabled {
		metricReporter := NewMetricReporter(s, workspaceID)
		metricReporter.Register(metric.InstanceCountMetricName, metricCollector.NewInstanceCountCollector(s.store))
		metricReporter.Register(metric.IssueCountMetricName, metricCollector.NewIssueCountCollector(s.store))
		metricReporter.Register(metric.ProjectCountMetricName, metricCollector.NewProjectCountCollector(s.store))
		metricReporter.Register(metric.PolicyCountMetricName, metricCollector.NewPolicyCountCollector(s.store))
		metricReporter.Register(metric.TaskCountMetricName, metricCollector.NewTaskCountCollector(s.store))
		metricReporter.Register(metric.DatabaseCountMetricName, metricCollector.NewDatabaseCountCollector(s.store))
		metricReporter.Register(metric.SheetCountMetricName, metricCollector.NewSheetCountCollector(s.store))
		metricReporter.Register(metric.MemberCountMetricName, metricCollector.NewMemberCountCollector(s.store))
		s.MetricReporter = metricReporter
	}
}

func (s *Server) initSetting(ctx context.Context, store *store.Store) (*config, error) {
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
		Value:       common.RandomString(secretLength),
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
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	if !s.profile.Readonly {
		// runnerWG waits for all goroutines to complete.
		go s.TaskScheduler.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.TaskCheckScheduler.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.SchemaSyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.BackupRunner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.AnomalyScanner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)

		if s.MetricReporter != nil {
			go s.MetricReporter.Run(ctx, &s.runnerWG)
			s.runnerWG.Add(1)
		}
	}

	// Sleep for 1 sec to make sure port is released between runs.
	time.Sleep(time.Duration(1) * time.Second)

	return s.e.Start(fmt.Sprintf(":%d", s.profile.BackendPort))
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Trying to stop Bytebase ....")
	log.Info("Trying to gracefully shutdown server")

	// Close the metric reporter
	if s.MetricReporter != nil {
		s.MetricReporter.Close()
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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

	// Wait for all runners to exit.
	s.runnerWG.Wait()

	// Close db connection
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			return err
		}
	}

	// Shutdown postgres server if embed.
	if s.metaDB != nil {
		s.metaDB.Close()
	}
	log.Info("Bytebase stopped properly")

	return nil
}

// GetEcho returns the echo server.
func (s *Server) GetEcho() *echo.Echo {
	return s.e
}
