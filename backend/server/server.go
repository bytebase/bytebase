// Package server implements the API server for Bytebase.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	// embed will embeds the acl policy.
	_ "embed"

	"github.com/blang/semver/v4"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	scas "github.com/qiangmzsx/string-adapter/v2"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/api/auth"
	v1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	enterpriseService "github.com/bytebase/bytebase/backend/enterprise/service"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
	metricCollector "github.com/bytebase/bytebase/backend/metric/collector"
	"github.com/bytebase/bytebase/backend/plugin/app/feishu"
	bbs3 "github.com/bytebase/bytebase/backend/plugin/storage/s3"
	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	"github.com/bytebase/bytebase/backend/resources/mysqlutil"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/runner/anomaly"
	"github.com/bytebase/bytebase/backend/runner/apprun"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/runner/metricreport"
	"github.com/bytebase/bytebase/backend/runner/rollbackrun"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/runner/taskcheck"
	"github.com/bytebase/bytebase/backend/runner/taskrun"
	"github.com/bytebase/bytebase/backend/store"
	_ "github.com/bytebase/bytebase/docs/openapi" // initial the swagger doc

	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/clickhouse"
	// Register mysql driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	// Register snowflake driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/snowflake"
	// Register sqlite driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"
	// Register mongodb driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mongodb"
	// Register spanner driver.
	_ "github.com/bytebase/bytebase/backend/plugin/db/spanner"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	// Register postgresql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"

	// Register mysql differ driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/differ/mysql"
	// Register postgres differ driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/differ/pg"
	// Register mysql edit driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/edit/mysql"
	// Register postgres edit driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/edit/pg"
	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/engine/pg"
	// Register mysql transform driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/transform/mysql"
)

const (
	// internalAPIPrefix is the API prefix for Bytebase internal, used by the UX.
	internalAPIPrefix = "/api"
	// webhookAPIPrefix is the API prefix for Bytebase webhook.
	webhookAPIPrefix = "/hook"
	// openAPIPrefix is the API prefix for Bytebase OpenAPI.
	openAPIPrefix = "/v1"
)

// Server is the Bytebase server.
type Server struct {
	// Asynchronous runners.
	TaskScheduler      *taskrun.Scheduler
	TaskCheckScheduler *taskcheck.Scheduler
	MetricReporter     *metricreport.Reporter
	SchemaSyncer       *schemasync.Syncer
	BackupRunner       *backuprun.Runner
	AnomalyScanner     *anomaly.Scanner
	ApplicationRunner  *apprun.Runner
	RollbackRunner     *rollbackrun.Runner
	runnerWG           sync.WaitGroup

	ActivityManager *activity.Manager

	licenseService enterpriseAPI.LicenseService

	// SchemaVersion is the bytebase's schema version
	SchemaVersion *semver.Version

	profile         config.Profile
	e               *echo.Echo
	grpcServer      *grpc.Server
	metaDB          *store.MetadataDB
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	startedTs       int64
	secret          string
	workspaceID     string
	errorRecordRing api.ErrorRecordRing

	// MySQL utility binaries
	mysqlBinDir string
	// MongoDB utility binaries
	mongoBinDir string
	// Postgres utility binaries
	pgBinDir string

	s3Client       *bbs3.Client
	feishuProvider *feishu.Provider

	// stateCfg is the shared in-momory state within the server.
	stateCfg *state.State

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
// swag init -g ./backend/server.go -d ./backend/server --output docs/openapi --parseDependency

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
func NewServer(ctx context.Context, profile config.Profile) (*Server, error) {
	s := &Server{
		profile:         profile,
		startedTs:       time.Now().Unix(),
		errorRecordRing: api.NewErrorRecordRing(),
	}

	resourceDir := common.GetResourceDir(profile.DataDir)
	if profile.ResourceDirOverride != "" {
		resourceDir = profile.ResourceDirOverride
	}

	// Display config
	log.Info("-----Config BEGIN-----")
	log.Info(fmt.Sprintf("mode=%s", profile.Mode))
	log.Info(fmt.Sprintf("dataDir=%s", profile.DataDir))
	log.Info(fmt.Sprintf("resourceDir=%s", resourceDir))
	log.Info(fmt.Sprintf("externalURL=%s", profile.ExternalURL))
	log.Info(fmt.Sprintf("readonly=%t", profile.Readonly))
	log.Info(fmt.Sprintf("debug=%t", profile.Debug))
	log.Info(fmt.Sprintf("demoName=%s", profile.DemoName))
	log.Info(fmt.Sprintf("backupStorageBackend=%s", profile.BackupStorageBackend))
	log.Info(fmt.Sprintf("backupBucket=%s", profile.BackupBucket))
	log.Info(fmt.Sprintf("backupRegion=%s", profile.BackupRegion))
	log.Info(fmt.Sprintf("backupCredentialFile=%s", profile.BackupCredentialFile))
	log.Info("-----Config END-------")

	serverStarted := false
	defer func() {
		if !serverStarted {
			_ = s.Shutdown(ctx)
		}
	}()

	var err error
	// Install mysqlutil
	s.mysqlBinDir, err = mysqlutil.Install(resourceDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot install mysql utility binaries")
	}

	s.mongoBinDir, err = mongoutil.Install(resourceDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot install mongo utility binaries")
	}

	// Installs the Postgres and utility binaries and creates the 'activeProfile.pgUser' user/database
	// to store Bytebase's own metadata.
	s.pgBinDir, err = postgres.Install(resourceDir)
	if err != nil {
		return nil, err
	}

	// New MetadataDB instance.
	if profile.UseEmbedDB() {
		pgDataDir := common.GetPostgresDataDir(profile.DataDir)
		log.Info("-----Embedded Postgres Config BEGIN-----")
		log.Info(fmt.Sprintf("datastorePort=%d", profile.DatastorePort))
		log.Info(fmt.Sprintf("pgDataDir=%s", pgDataDir))
		log.Info("-----Embedded Postgres Config END-----")
		if err := postgres.InitDB(s.pgBinDir, pgDataDir, profile.PgUser); err != nil {
			return nil, err
		}
		s.metaDB = store.NewMetadataDBWithEmbedPg(profile.PgUser, pgDataDir, s.pgBinDir, profile.DemoName, profile.Mode)
	} else {
		s.metaDB = store.NewMetadataDBWithExternalPg(profile.PgURL, s.pgBinDir, profile.DemoName, profile.Mode)
	}

	// Connect to the instance that stores bytebase's own metadata.
	storeDB, err := s.metaDB.Connect(profile.DatastorePort, profile.Readonly, profile.Version)
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect metadb")
	}

	schemaVer, err := storeDB.Open(ctx)
	if err != nil {
		// return s so that caller can call s.Close() to shut down the postgres server if embedded.
		return nil, errors.Wrap(err, "cannot open metadb")
	}
	s.SchemaVersion = schemaVer

	s.stateCfg = &state.State{
		InstanceDatabaseSyncChan:       make(chan *api.Instance, 100),
		InstanceOutstandingConnections: make(map[int]int),
	}
	storeInstance := store.New(storeDB)
	s.store = storeInstance
	s.licenseService, err = enterpriseService.NewLicenseService(profile.Mode, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create license service")
	}
	// Cache the license.
	s.licenseService.LoadSubscription(ctx)

	config, err := getInitSetting(ctx, storeInstance)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init config")
	}
	s.secret = config.secret
	s.workspaceID = config.workspaceID

	s.ActivityManager = activity.NewManager(storeInstance, profile)
	s.dbFactory = dbfactory.New(s.mysqlBinDir, s.mongoBinDir, s.pgBinDir, profile.DataDir, s.secret)
	e := echo.New()
	e.Debug = profile.Debug
	e.HideBanner = true
	e.HidePort = true

	// Disallow to be embedded in an iFrame.
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XFrameOptions: "DENY",
	}))

	embedFrontend(e)
	s.e = e

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

	if !profile.Readonly {
		s.SchemaSyncer = schemasync.NewSyncer(storeInstance, s.dbFactory, s.stateCfg, profile)
		// TODO(p0ny): enable Feishu provider only when it is needed.
		s.feishuProvider = feishu.NewProvider(profile.FeishuAPIURL)
		s.ApplicationRunner = apprun.NewRunner(storeInstance, s.ActivityManager, s.feishuProvider, profile)
		s.BackupRunner = backuprun.NewRunner(storeInstance, s.dbFactory, s.s3Client, s.stateCfg, &profile)
		s.RollbackRunner = rollbackrun.NewRunner(storeInstance, s.dbFactory, s.stateCfg)

		s.TaskScheduler = taskrun.NewScheduler(storeInstance, s.ApplicationRunner, s.SchemaSyncer, s.ActivityManager, s.licenseService, s.stateCfg, profile)
		s.TaskScheduler.Register(api.TaskGeneral, taskrun.NewDefaultExecutor())
		s.TaskScheduler.Register(api.TaskDatabaseCreate, taskrun.NewDatabaseCreateExecutor(storeInstance, s.dbFactory, s.SchemaSyncer, profile))
		s.TaskScheduler.Register(api.TaskDatabaseSchemaBaseline, taskrun.NewSchemaBaselineExecutor(storeInstance, s.dbFactory, s.ActivityManager, s.licenseService, s.stateCfg, s.SchemaSyncer, profile))
		s.TaskScheduler.Register(api.TaskDatabaseSchemaUpdate, taskrun.NewSchemaUpdateExecutor(storeInstance, s.dbFactory, s.ActivityManager, s.licenseService, s.stateCfg, s.SchemaSyncer, profile))
		s.TaskScheduler.Register(api.TaskDatabaseSchemaUpdateSDL, taskrun.NewSchemaUpdateSDLExecutor(storeInstance, s.dbFactory, s.ActivityManager, s.licenseService, s.stateCfg, s.SchemaSyncer, profile))
		s.TaskScheduler.Register(api.TaskDatabaseDataUpdate, taskrun.NewDataUpdateExecutor(storeInstance, s.dbFactory, s.ActivityManager, s.licenseService, s.stateCfg, profile))
		s.TaskScheduler.Register(api.TaskDatabaseBackup, taskrun.NewDatabaseBackupExecutor(storeInstance, s.dbFactory, s.s3Client, profile))
		s.TaskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostSync, taskrun.NewSchemaUpdateGhostSyncExecutor(storeInstance, s.stateCfg, s.secret))
		s.TaskScheduler.Register(api.TaskDatabaseSchemaUpdateGhostCutover, taskrun.NewSchemaUpdateGhostCutoverExecutor(storeInstance, s.dbFactory, s.ActivityManager, s.licenseService, s.stateCfg, s.SchemaSyncer, profile))
		s.TaskScheduler.Register(api.TaskDatabaseRestorePITRRestore, taskrun.NewPITRRestoreExecutor(storeInstance, s.dbFactory, s.s3Client, s.SchemaSyncer, s.stateCfg, profile))
		s.TaskScheduler.Register(api.TaskDatabaseRestorePITRCutover, taskrun.NewPITRCutoverExecutor(storeInstance, s.dbFactory, s.SchemaSyncer, s.BackupRunner, s.ActivityManager, profile))

		s.TaskCheckScheduler = taskcheck.NewScheduler(storeInstance, s.licenseService, s.stateCfg)
		statementSimpleExecutor := taskcheck.NewStatementAdvisorSimpleExecutor()
		s.TaskCheckScheduler.Register(api.TaskCheckDatabaseStatementFakeAdvise, statementSimpleExecutor)
		s.TaskCheckScheduler.Register(api.TaskCheckDatabaseStatementSyntax, statementSimpleExecutor)
		statementCompositeExecutor := taskcheck.NewStatementAdvisorCompositeExecutor(storeInstance, s.dbFactory)
		s.TaskCheckScheduler.Register(api.TaskCheckDatabaseStatementAdvise, statementCompositeExecutor)
		statementTypeExecutor := taskcheck.NewStatementTypeExecutor(storeInstance)
		s.TaskCheckScheduler.Register(api.TaskCheckDatabaseStatementType, statementTypeExecutor)
		databaseConnectExecutor := taskcheck.NewDatabaseConnectExecutor(storeInstance, s.dbFactory)
		s.TaskCheckScheduler.Register(api.TaskCheckDatabaseConnect, databaseConnectExecutor)
		migrationSchemaExecutor := taskcheck.NewMigrationSchemaExecutor(storeInstance, s.dbFactory)
		s.TaskCheckScheduler.Register(api.TaskCheckInstanceMigrationSchema, migrationSchemaExecutor)
		ghostSyncExecutor := taskcheck.NewGhostSyncExecutor(storeInstance, s.secret)
		s.TaskCheckScheduler.Register(api.TaskCheckGhostSync, ghostSyncExecutor)
		checkLGTMExecutor := taskcheck.NewLGTMExecutor(storeInstance)
		s.TaskCheckScheduler.Register(api.TaskCheckIssueLGTM, checkLGTMExecutor)
		pitrMySQLExecutor := taskcheck.NewPITRMySQLExecutor(storeInstance, s.dbFactory)
		s.TaskCheckScheduler.Register(api.TaskCheckPITRMySQL, pitrMySQLExecutor)

		// Anomaly scanner
		s.AnomalyScanner = anomaly.NewScanner(storeInstance, s.dbFactory, s.licenseService)

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
			return !common.HasPrefixes(c.Path(), internalAPIPrefix, openAPIPrefix, webhookAPIPrefix)
		},
		Format: `{"time":"${time_rfc3339}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"}` + "\n",
	}))
	e.Use(recoverMiddleware)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	webhookGroup := e.Group(webhookAPIPrefix)
	s.registerWebhookRoutes(webhookGroup)

	apiGroup := e.Group(internalAPIPrefix)
	apiGroup.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWTMiddleware(internalAPIPrefix, s.store, next, profile.Mode, config.secret)
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
		return aclMiddleware(s, internalAPIPrefix, ce, next, profile.Readonly)
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
	s.registerSubscriptionRoutes(apiGroup)
	s.registerSheetRoutes(apiGroup)
	s.registerSheetOrganizerRoutes(apiGroup)
	s.registerAnomalyRoutes(apiGroup)

	// Register healthz endpoint.
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK!\n")
	})

	// Setup the gRPC and grpc-gateway.
	authProvider := auth.New(s.store, s.secret, s.licenseService, profile.Mode)
	aclProvider := v1.NewACLInterceptor(s.store, s.secret, s.licenseService, profile.Mode)
	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(authProvider.AuthenticationInterceptor, aclProvider.ACLInterceptor),
	)
	v1pb.RegisterAuthServiceServer(s.grpcServer, v1.NewAuthService(s.store, s.secret, s.MetricReporter, &profile))
	v1pb.RegisterEnvironmentServiceServer(s.grpcServer, v1.NewEnvironmentService(s.store, s.licenseService))
	v1pb.RegisterInstanceServiceServer(s.grpcServer, v1.NewInstanceService(s.store, s.licenseService, s.secret))
	v1pb.RegisterProjectServiceServer(s.grpcServer, v1.NewProjectService(s.store))
	v1pb.RegisterDatabaseServiceServer(s.grpcServer, v1.NewDatabaseService(s.store))
	v1pb.RegisterInstanceRoleServiceServer(s.grpcServer, v1.NewInstanceRoleService(s.store, s.dbFactory))
	v1pb.RegisterOrgPolicyServiceServer(s.grpcServer, v1.NewOrgPolicyService(s.store, s.licenseService))
	v1pb.RegisterIdentityProviderServiceServer(s.grpcServer, v1.NewIdentityProviderService(s.store, s.licenseService))
	reflection.Register(s.grpcServer)

	// REST gateway proxy.
	grpcEndpoint := fmt.Sprintf(":%d", profile.GrpcPort)
	grpcConn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	mux := runtime.NewServeMux(runtime.WithForwardResponseOption(auth.GatewayResponseModifier))
	if err := v1pb.RegisterAuthServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterEnvironmentServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterInstanceServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterProjectServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterDatabaseServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterInstanceRoleServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterOrgPolicyServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	if err := v1pb.RegisterIdentityProviderServiceHandler(ctx, mux, grpcConn); err != nil {
		return nil, err
	}
	e.Any("/v1/*", echo.WrapHandler(mux))
	// GRPC web proxy.
	options := []grpcweb.Option{
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
	}
	wrappedGrpc := grpcweb.WrapServer(s.grpcServer, options...)
	e.Any("/bytebase.v1.*", echo.WrapHandler(wrappedGrpc))

	// Register open API routes
	s.registerOpenAPIRoutes(e, ce, profile)

	// Register pprof endpoints.
	pprof.Register(e)
	// Register prometheus metrics endpoint.
	p := prometheus.NewPrometheus("api", nil)
	p.Use(e)

	serverStarted = true
	return s, nil
}

func (s *Server) registerOpenAPIRoutes(e *echo.Echo, ce *casbin.Enforcer, prof config.Profile) {
	jwtMiddlewareFunc := func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWTMiddleware(openAPIPrefix, s.store, next, prof.Mode, s.secret)
	}
	aclMiddlewareFunc := func(next echo.HandlerFunc) echo.HandlerFunc {
		return aclMiddleware(s, openAPIPrefix, ce, next, prof.Readonly)
	}
	metricMiddlewareFunc := func(next echo.HandlerFunc) echo.HandlerFunc {
		return openAPIMetricMiddleware(s, next)
	}
	e.POST("/v1/sql/advise", s.sqlCheckController)
	e.POST("/v1/sql/schema/diff", schemaDiff)
	e.PATCH("/v1/instances/:instanceName/databases/:database", s.updateInstanceDatabase, jwtMiddlewareFunc, aclMiddlewareFunc, metricMiddlewareFunc)
	e.POST("/v1/issues", s.createIssueByOpenAPI, jwtMiddlewareFunc, aclMiddlewareFunc, metricMiddlewareFunc)
}

// initMetricReporter will initial the metric scheduler.
func (s *Server) initMetricReporter(workspaceID string) {
	enabled := s.profile.Mode == common.ReleaseModeProd && s.profile.DemoName != "" && !s.profile.DisableMetric
	if enabled {
		metricReporter := metricreport.NewReporter(s.store, s.licenseService, s.profile, workspaceID)
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

// retrieved via the SettingService upon startup.
type workspaceConfig struct {
	// secret used to sign the JWT auth token
	secret string
	// workspaceID used to initial the identify for a new workspace.
	workspaceID string
}

func getInitSetting(ctx context.Context, store *store.Store) (*workspaceConfig, error) {
	// secretLength is the length for the secret used to sign the JWT auto token.
	const secretLength = 32

	// initial branding
	if _, _, err := store.CreateSettingIfNotExist(ctx, &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding logo image in base64 string format.",
	}); err != nil {
		return nil, err
	}

	conf := &workspaceConfig{}

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
			return errors.Wrap(err, "failed to clear existing RUNNING tasks before starting the task scheduler")
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
		s.runnerWG.Add(1)
		go s.ApplicationRunner.Run(ctx, &s.runnerWG)
		if s.profile.Mode == common.ReleaseModeDev {
			s.runnerWG.Add(1)
			go s.RollbackRunner.Run(ctx, &s.runnerWG)
		}

		if s.MetricReporter != nil {
			s.runnerWG.Add(1)
			go s.MetricReporter.Run(ctx, &s.runnerWG)
		}
	}

	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port+1))
	if err != nil {
		return err
	}
	go func() {
		if err := s.grpcServer.Serve(listen); err != nil {
			log.Error("grpc server listen error", zap.Error(err))
		}
	}()
	return s.e.Start(fmt.Sprintf(":%d", port))
}

// Shutdown will shut down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Stopping Bytebase...")
	log.Info("Stopping web server...")

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
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
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
