// Package server implements the API server for Bytebase.
package server

import (
	"context"
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
	"github.com/pkg/errors"
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
	s3bb "github.com/bytebase/bytebase/plugin/storage/s3"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/store"
)

// openAPIPrefix is the API prefix for Bytebase OpenAPI.
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

	profile         Profile
	e               *echo.Echo
	pgInstance      *postgres.Instance
	metaDB          *store.MetadataDB
	store           *store.Store
	startedTs       int64
	secret          string
	workspaceID     string
	errorRecordRing api.ErrorRecordRing

	s3Client *s3bb.Client

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
		profile:         prof,
		startedTs:       time.Now().Unix(),
		errorRecordRing: api.NewErrorRecordRing(),
	}

	// Display config
	log.Info("-----Config BEGIN-----")
	log.Info(fmt.Sprintf("mode=%s", prof.Mode))
	log.Info(fmt.Sprintf("externalURL=%s", prof.ExternalURL))
	log.Info(fmt.Sprintf("demoDataDir=%s", prof.DemoDataDir))
	log.Info(fmt.Sprintf("readonly=%t", prof.Readonly))
	log.Info(fmt.Sprintf("demo=%t", prof.Demo))
	log.Info(fmt.Sprintf("debug=%t", prof.Debug))
	log.Info(fmt.Sprintf("dataDir=%s", prof.DataDir))
	log.Info(fmt.Sprintf("backupStorageBackend=%s", prof.BackupStorageBackend))
	log.Info(fmt.Sprintf("backupBucket=%s", prof.BackupBucket))
	log.Info(fmt.Sprintf("backupRegion=%s", prof.BackupRegion))
	log.Info(fmt.Sprintf("backupCredentialFile=%s", prof.BackupCredentialFile))
	log.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var err error

	resourceDir := common.GetResourceDir(prof.DataDir)
	// Install mysqlutil
	if err := mysqlutil.Install(resourceDir); err != nil {
		return nil, errors.Wrap(err, "cannot install mysqlbinlog binary")
	}

	// Install Postgres.
	var pgDataDir string
	if prof.useEmbedDB() {
		pgDataDir = common.GetPostgresDataDir(prof.DataDir)
	}
	log.Info("-----Embedded Postgres Config BEGIN-----")
	log.Info(fmt.Sprintf("datastorePort=%d", prof.DatastorePort))
	log.Info(fmt.Sprintf("resourceDir=%s", resourceDir))
	log.Info(fmt.Sprintf("pgdataDir=%s", pgDataDir))
	log.Info("-----Embedded Postgres Config END-----")
	log.Info("Preparing embedded PostgreSQL instance...")
	// Installs the Postgres binary and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	log.Info(fmt.Sprintf("Installing Postgres OS %q Arch %q", runtime.GOOS, runtime.GOARCH))
	pgInstance, err := postgres.Install(resourceDir, pgDataDir, prof.PgUser)
	if err != nil {
		return nil, err
	}
	s.pgInstance = pgInstance

	// New MetadataDB instance.
	if prof.useEmbedDB() {
		s.metaDB = store.NewMetadataDBWithEmbedPg(pgInstance, prof.PgUser, prof.DemoDataDir, prof.Mode)
	} else {
		s.metaDB = store.NewMetadataDBWithExternalPg(pgInstance, prof.PgURL, prof.DemoDataDir, prof.Mode)
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot create MetadataDB instance")
	}

	// New store.DB instance that represents the db connection.
	storeDB, err := s.metaDB.Connect(prof.DatastorePort, prof.Readonly, prof.Version)
	if err != nil {
		return nil, errors.Wrap(err, "cannot new db")
	}

	// Open the database that stores bytebase's own metadata connection.
	if err = storeDB.Open(ctx); err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, errors.Wrap(err, "cannot open db")
	}

	cacheService := NewCacheService()
	storeInstance := store.New(storeDB, cacheService)
	s.store = storeInstance

	// Backfill activity.
	if err := storeInstance.BackfillSQLEditorActivity(ctx); err != nil {
		return nil, errors.Wrap(err, "cannot backfill SQL editor activities")
	}

	config, err := getInitSetting(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}
	s.secret = config.secret
	s.workspaceID = config.workspaceID

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

	if prof.BackupBucket != "" {
		credentials, err := s3bb.GetCredentialsFromFile(ctx, prof.BackupCredentialFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get credentials from file")
		}
		s3Client, err := s3bb.NewClient(ctx, prof.BackupRegion, prof.BackupBucket, credentials)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create AWS S3 client")
		}
		s.s3Client = s3Client
	}

	if !prof.Readonly {
		// Task scheduler
		taskScheduler := NewTaskScheduler(s)

		taskScheduler.Register(api.TaskGeneral, NewDefaultTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseCreate, NewDatabaseCreateTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseSchemaBaseline, NewSchemaBaselineTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseSchemaUpdate, NewSchemaUpdateTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseSchemaUpdateSDL, NewSchemaUpdateSDLTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseDataUpdate, NewDataUpdateTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseBackup, NewDatabaseBackupTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostSync, NewSchemaUpdateGhostSyncTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostCutover, NewSchemaUpdateGhostCutoverTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseRestorePITRRestore, NewPITRRestoreTaskExecutor)

		taskScheduler.Register(api.TaskDatabaseRestorePITRCutover, NewPITRCutoverTaskExecutor)

		s.TaskScheduler = taskScheduler

		// Task check scheduler
		taskCheckScheduler := NewTaskCheckScheduler(s)

		statementSimpleExecutor := NewTaskCheckStatementAdvisorSimpleExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementFakeAdvise, statementSimpleExecutor)
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementSyntax, statementSimpleExecutor)

		statementCompositeExecutor := NewTaskCheckStatementAdvisorCompositeExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementAdvise, statementCompositeExecutor)

		statementTypeExecutor := NewTaskCheckStatementTypeExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseStatementType, statementTypeExecutor)

		databaseConnectExecutor := NewTaskCheckDatabaseConnectExecutor()
		taskCheckScheduler.Register(api.TaskCheckDatabaseConnect, databaseConnectExecutor)

		migrationSchemaExecutor := NewTaskCheckMigrationSchemaExecutor()
		taskCheckScheduler.Register(api.TaskCheckInstanceMigrationSchema, migrationSchemaExecutor)

		ghostSyncExecutor := NewTaskCheckGhostSyncExecutor()
		taskCheckScheduler.Register(api.TaskCheckGhostSync, ghostSyncExecutor)

		checkLGTMExecutor := NewTaskCheckLGTMExecutor()
		taskCheckScheduler.Register(api.TaskCheckIssueLGTM, checkLGTMExecutor)

		pitrMySQLExecutor := NewTaskCheckPITRMySQLExecutor()
		taskCheckScheduler.Register(api.TaskCheckPITRMySQL, pitrMySQLExecutor)

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
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		errorRecorderMiddleware(err, s, c, e)
	}

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			if s.profile.Mode == common.ReleaseModeProd && !s.profile.Debug {
				return true
			}
			return !common.HasPrefixes(c.Path(), "/api", "/hook")
		},
		Format: `{"time":"${time_rfc3339}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"}` + "\n",
	}))
	e.Use(recoverMiddleware)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	webhookGroup := e.Group("/hook")
	s.registerWebhookRoutes(webhookGroup)

	openAPIGroup := e.Group(openAPIPrefix)
	openAPIGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return openAPIMetricMiddleware(s, next)
	})
	s.registerOpenAPIRoutes(openAPIGroup)

	apiGroup := e.Group("/api")
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
	s.registerAnomalyRoutes(apiGroup)

	// Register healthz endpoint.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!\n")
	})
	// Register pprof endpoints.
	pprof.Register(e)
	// Register prometheus metrics endpoint.
	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	s.ActivityManager = NewActivityManager(s, storeInstance)
	s.LicenseService, err = enterpriseService.NewLicenseService(prof.Mode, s.store)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create license service")
	}

	s.initSubscription(ctx)

	serverStarted = true
	return s, nil
}

// initSubscription will initial the subscription cache in memory.
func (s *Server) initSubscription(ctx context.Context) {
	s.subscription = s.loadSubscription(ctx)
}

// initMetricReporter will initial the metric scheduler.
func (s *Server) initMetricReporter(workspaceID string) {
	enabled := s.profile.Mode == common.ReleaseModeProd && !s.profile.Demo && !s.profile.DisableMetric
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

func getInitSetting(ctx context.Context, store *store.Store) (*config, error) {
	// initial branding
	if _, _, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding logo image in base64 string format.",
	}); err != nil {
		return nil, err
	}

	conf := &config{}

	// initial JWT token
	value, err := common.RandomString(secretLength)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random JWT secret")
	}
	authSetting, _, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingAuthSecret,
		Value:       value,
		Description: "Random string used to sign the JWT auth token.",
	})
	if err != nil {
		return nil, err
	}
	conf.secret = authSetting.Value

	// initial workspace
	workspaceSetting, _, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
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
	if _, _, err = store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingEnterpriseLicense,
		Value:       "",
		Description: "Enterprise license",
	}); err != nil {
		return nil, err
	}

	// initial feishu app
	if _, _, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingAppIM,
		Value:       "",
		Description: "",
	}); err != nil {
		return nil, err
	}

	return conf, nil
}

// Run will run the server.
func (s *Server) Run(ctx context.Context, port int) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	if !s.profile.Readonly {
		if err := s.TaskScheduler.ClearRunningTasks(ctx); err != nil {
			return errors.Wrap(err, "failed to clear existing RUNNING tasks before start the task scheduler")
		}
		// runnerWG waits for all goroutines to complete.
		s.runnerWG.Add(1)
		go s.TaskScheduler.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.TaskCheckScheduler.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.SchemaSyncer.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.BackupRunner.Run(ctx, &s.runnerWG)
		s.runnerWG.Add(1)
		go s.AnomalyScanner.Run(ctx, &s.runnerWG)

		if s.MetricReporter != nil {
			s.runnerWG.Add(1)
			go s.MetricReporter.Run(ctx, &s.runnerWG)
		}
	}

	return s.e.Start(fmt.Sprintf(":%d", port))
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

// GetWorkspaceID returns the workspace id.
func (s *Server) GetWorkspaceID() string {
	return s.workspaceID
}
