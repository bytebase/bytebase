package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterprise "github.com/bytebase/bytebase/enterprise/service"
	dbdriver "github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/plugin/db/clickhouse"
	// Register mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	// Register snowflake driver.
	_ "github.com/bytebase/bytebase/plugin/db/snowflake"
	// Register sqlite driver.
	_ "github.com/bytebase/bytebase/plugin/db/sqlite"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/mysql"
)

// -----------------------------------Global constant BEGIN----------------------------------------
const (
	// secretLength is the length for the secret used to sign the JWT auth token
	secretLength = 32

	// greetingBanner is the greeting banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase
	greetingBanner = `
██████╗ ██╗   ██╗████████╗███████╗██████╗  █████╗ ███████╗███████╗
██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝
██████╔╝ ╚████╔╝    ██║   █████╗  ██████╔╝███████║███████╗█████╗
██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ██╔══██╗██╔══██║╚════██║██╔══╝
██████╔╝   ██║      ██║   ███████╗██████╔╝██║  ██║███████║███████╗
╚═════╝    ╚═╝      ╚═╝   ╚══════╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝

%s
___________________________________________________________________________________________

`
	// byeBanner is the bye banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=BYE
	byeBanner = `
██████╗ ██╗   ██╗███████╗
██╔══██╗╚██╗ ██╔╝██╔════╝
██████╔╝ ╚████╔╝ █████╗
██╔══██╗  ╚██╔╝  ██╔══╝
██████╔╝   ██║   ███████╗
╚═════╝    ╚═╝   ╚══════╝

`
)

// -----------------------------------Global constant END------------------------------------------

// -----------------------------------Command Line Config BEGIN------------------------------------
var (
	// Used for flags.
	host         string
	port         int
	frontendHost string
	frontendPort int
	dataDir      string
	// When we are running in readonly mode:
	// - The data file will be opened in readonly mode, no applicable migration or seeding will be applied.
	// - Requests other than GET will be rejected
	// - Any operations involving mutation will not start (e.g. Background schema syncer, task scheduler)
	readonly bool
	demo     bool
	debug    bool
	// pgURL must follow PostgreSQL connection URIs pattern.
	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
	pgURL string

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(_ *cobra.Command, _ []string) {
			if frontendHost == "" {
				frontendHost = host
			}
			if frontendPort == 0 {
				frontendPort = port
			}

			start()

			fmt.Print(byeBanner)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "http://localhost", "host where Bytebase backend is accessed from, must start with http:// or https://. This is used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().IntVar(&port, "port", 80, "port where Bytebase backend is accessed from. This is also used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().StringVar(&frontendHost, "frontend-host", "", "host where Bytebase frontend is accessed from, must start with http:// or https://. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --host")
	rootCmd.PersistentFlags().IntVar(&frontendPort, "frontend-port", 0, "port where Bytebase frontend is accessed from. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --port")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data", ".", "directory where Bytebase stores data. If relative path is supplied, then the path is relative to the directory where bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&readonly, "readonly", false, "whether to run in read-only mode")
	rootCmd.PersistentFlags().BoolVar(&demo, "demo", false, "whether to run using demo data")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "whether to enable debug level logging")
	// TODO(tianzhou): this needs more bake time. There are couple blocking issues:
	// 1. Currently, we will create a separate bytebase database to store the migration_history table, we need to put it inside the specified database here.
	// 2. We need to move the logic of creating bytebase metadata db logic outside. Because with --pg option, the db has already been created.
	// rootCmd.PersistentFlags().StringVar(&pgURL, "pg", "", "optional external PostgreSQL instance connection url; for example postgresql://user:secret@masterhost:5432/dbname?sslrootcert=cert")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------

// Profile is the configuration to start main server.
type Profile struct {
	// mode can be "prod" or "dev"
	mode common.ReleaseMode
	// port is the binding port for server.
	port int
	// datastorePort is the binding port for database instance for storing Bytebase data.
	datastorePort int
	// pgUser is the user we use to connect to bytebase's Postgres database.
	// The name of the database storing metadata is the same as pgUser.
	pgUser string
	// dataDir is the directory stores the data including Bytebase's own database, backups, etc.
	dataDir string
	// demoDataDir points to where to populate the initial data.
	demoDataDir string
	// backupRunnerInterval is the interval for backup runner.
	backupRunnerInterval time.Duration
}

// retrieved via the SettingService upon startup
type config struct {
	// secret used to sign the JWT auth token
	secret string
}

// Main is the main server for Bytebase.
type Main struct {
	profile *Profile

	l   *zap.Logger
	lvl *zap.AtomicLevel

	server *server.Server
	// serverCancel cancels any runner on the server.
	// Then the runnerWG waits for all runners to finish before we shutdown the server.
	// Otherwise, we will get database is closed error from runner when we shutdown the server.
	serverCancel context.CancelFunc

	pg *postgres.Instance
	// pgStarted only used if we install Postgres and boot it,
	// Close() will shutdown Postgres if it is true.
	pgStarted bool
	// db is a connection to the database storing Bytebase data.
	db *store.DB
}

func checkDataDir() error {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + dataDir)
		if err != nil {
			return err
		}
		dataDir = absDir
	}

	// Trim trailing / in case user supplies
	dataDir = strings.TrimRight(dataDir, "/")

	if _, err := os.Stat(dataDir); err != nil {
		error := fmt.Errorf("unable to access --data %s, %w", dataDir, err)
		return error
	}

	return nil
}

// GetLogger will return a logger.
func GetLogger() (*zap.Logger, *zap.AtomicLevel, error) {
	atom := zap.NewAtomicLevelAt(zap.InfoLevel)
	if debug {
		atom.SetLevel(zap.DebugLevel)
	}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stdout),
		atom,
	))
	return logger, &atom, nil
}

func start() {
	logger, level, err := GetLogger()
	if err != nil {
		panic(fmt.Errorf("failed to create logger, %w", err))
	}
	defer logger.Sync()

	if !common.HasPrefixes(host, "http://", "https://") {
		logger.Error(fmt.Sprintf("--host %s must start with http:// or https://", host))
		return
	}
	if err := checkDataDir(); err != nil {
		logger.Error(err.Error())
		return
	}

	// We use port+1 as datastore port.
	datastorePort := port + 1
	activeProfile := activeProfile(dataDir, port, datastorePort, demo)

	ctx, cancel := context.WithCancel(context.Background())

	m, err := NewMain(ctx, activeProfile, logger, level)
	if err != nil {
		logger.Error(err.Error())
		cancel()
		return
	}

	// Setup signal handlers.
	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		m.l.Info("SIGINT received.")
		if err := m.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		cancel()
	}()

	// Execute program.
	if err := m.Run(ctx); err != nil {
		if err != http.ErrServerClosed {
			m.l.Error(err.Error())
			m.Close()
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}

// NewMain creates a main server based on profile.
func NewMain(ctx context.Context, activeProfile Profile, l *zap.Logger, logLevel *zap.AtomicLevel) (*Main, error) {
	/* <------- Configure directory -------> */
	resourceDir := path.Join(activeProfile.dataDir, "resources")
	pgDataDir := common.GetPostgresDataDir(activeProfile.dataDir)

	/* <------- Print some help info -------> */
	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", activeProfile.mode)
	fmt.Printf("server=%s:%d\n", host, activeProfile.port)
	fmt.Printf("datastore=%s:%d\n", host, activeProfile.datastorePort)
	fmt.Printf("frontend=%s:%d\n", frontendHost, frontendPort)
	if useEmbeddedDB() {
		fmt.Printf("resourceDir=%s\n", resourceDir)
		fmt.Printf("pgdataDir=%s\n", pgDataDir)
	}
	fmt.Printf("demoDataDir=%s\n", activeProfile.demoDataDir)
	fmt.Printf("readonly=%t\n", readonly)
	fmt.Printf("demo=%t\n", demo)
	fmt.Printf("debug=%t\n", debug)
	fmt.Println("-----Config END-------")

	/* <------- Install Postgres on machine if using embedded db  -------> */
	var pgInstance *postgres.Instance
	if useEmbeddedDB() {
		l.Info("Preparing embedded PostgreSQL instance...")
		var err error
		// Installs the Postgres binary and creates the 'activeProfile.pgUser' user/database
		// to store Bytebase's own metadata.
		pgInstance, err = postgres.Install(resourceDir, pgDataDir, activeProfile.pgUser)
		if err != nil {
			return nil, err
		}
	}

	var db *store.DB
	var pgStarted = false
	if useEmbeddedDB() {
		var err error
		db, err = newEmbeddedDB(l, &activeProfile, pgInstance)
		if err != nil {
			return nil, err
		}
		pgStarted = true
	} else {
		var err error
		db, err = newExternalDB(l, &activeProfile)
		if err != nil {
			return nil, err
		}
	}

	// Open the connection of `bb` in Postgres and setup migration if needed
	cancelCtx, cancel := context.WithCancel(ctx)
	if err := db.Open(cancelCtx); err != nil {
		cancel()
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	s, err := setupService(ctx, l, logLevel, db, &activeProfile)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("cannot setupService: %w", err)
	}
	return &Main{
		profile:      &activeProfile,
		l:            l,
		lvl:          logLevel,
		server:       s,
		serverCancel: cancel,
		pg:           pgInstance,
		pgStarted:    pgStarted,
		db:           db,
	}, nil
}

func initSetting(ctx context.Context, settingService api.SettingService) (*config, error) {
	result := &config{}
	{
		configCreate := &api.SettingCreate{
			CreatorID:   api.SystemBotID,
			Name:        api.SettingAuthSecret,
			Value:       common.RandomString(secretLength),
			Description: "Random string used to sign the JWT auth token.",
		}
		config, err := settingService.CreateSettingIfNotExist(ctx, configCreate)
		if err != nil {
			return nil, err
		}
		result.secret = config.Value
	}

	return result, nil
}

func initBranding(ctx context.Context, settingService api.SettingService) error {
	configCreate := &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding logo image in base64 string format.",
	}
	_, err := settingService.CreateSettingIfNotExist(ctx, configCreate)
	if err != nil {
		return err
	}

	return nil
}

func useEmbeddedDB() bool {
	return len(pgURL) == 0
}

func newExternalDB(l *zap.Logger, profile *Profile) (*store.DB, error) {
	u, err := url.Parse(pgURL)
	if err != nil {
		return nil, err
	}

	l.Info("Establishing external PostgreSQL connection...", zap.String("pgURL", u.Redacted()))

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("invalid connection protocol: %s", u.Scheme)
	}

	connCfg := dbdriver.ConnectionConfig{}

	if u.User != nil {
		connCfg.Username = u.User.Username()
		connCfg.Password, _ = u.User.Password()
	}

	if connCfg.Username == "" {
		return nil, fmt.Errorf("missing user in the --pg connection string")
	}

	if host, port, err := net.SplitHostPort(u.Host); err != nil {
		connCfg.Host = u.Host
	} else {
		connCfg.Host = host
		connCfg.Port = port
	}

	// By default, follow the PG convention to use user name as the database name
	connCfg.Database = connCfg.Username
	if u.Path != "" {
		connCfg.Database = u.Path[1:]
	}

	q := u.Query()
	connCfg.TLSConfig = dbdriver.TLSConfig{
		SslCA:   q.Get("sslrootcert"),
		SslKey:  q.Get("sslkey"),
		SslCert: q.Get("sslcert"),
	}

	db := store.NewDB(l, connCfg, profile.demoDataDir, readonly, version, profile.mode)
	return db, nil
}

func newEmbeddedDB(l *zap.Logger, profile *Profile, pg *postgres.Instance) (*store.DB, error) {
	if err := pg.Start(profile.datastorePort, os.Stderr, os.Stderr); err != nil {
		return nil, err
	}

	// Even when Postgres opens Unix domain socket only for connection, it still requires a port as ID to differentiate different Postgres instances.
	connCfg := dbdriver.ConnectionConfig{
		Username: profile.pgUser,
		Password: "",
		Host:     common.GetPostgresSocketDir(),
		Port:     fmt.Sprintf("%d", profile.datastorePort),
	}
	db := store.NewDB(l, connCfg, profile.demoDataDir, readonly, version, profile.mode)
	return db, nil
}

func setupService(ctx context.Context, l *zap.Logger, logLevel *zap.AtomicLevel, db *store.DB, profile *Profile) (*server.Server, error) {
	settingService := store.NewSettingService(l, db)
	config, err := initSetting(ctx, settingService)
	if err != nil {
		return nil, fmt.Errorf("failed to init config %w", err)
	}
	err = initBranding(ctx, settingService)
	if err != nil {
		return nil, fmt.Errorf("failed to init branding: %w", err)
	}

	cacheService := server.NewCacheService(l)
	storeInstance := store.New(l, db, cacheService)

	s := server.NewServer(l, storeInstance, logLevel, version, host, profile.port, frontendHost, frontendPort, profile.datastorePort, profile.mode, profile.dataDir, profile.backupRunnerInterval, config.secret, readonly, demo, debug)
	s.SettingService = settingService
	s.ProjectService = store.NewProjectService(l, db, cacheService)
	s.ProjectMemberService = store.NewProjectMemberService(l, db)
	s.ProjectWebhookService = store.NewProjectWebhookService(l, db)
	s.DatabaseService = store.NewDatabaseService(l, db, cacheService, storeInstance)
	// TODO(dragonly): remove this hack
	storeInstance.DatabaseService = s.DatabaseService
	s.InstanceUserService = store.NewInstanceUserService(l, db)
	s.TableService = store.NewTableService(l, db)
	s.ColumnService = store.NewColumnService(l, db)
	s.ViewService = store.NewViewService(l, db)
	s.IndexService = store.NewIndexService(l, db)
	s.IssueService = store.NewIssueService(l, db, cacheService)
	s.PipelineService = store.NewPipelineService(l, db, cacheService)
	s.StageService = store.NewStageService(l, db)
	s.TaskCheckRunService = store.NewTaskCheckRunService(l, db)
	s.TaskService = store.NewTaskService(l, db, store.NewTaskRunService(l, db), s.TaskCheckRunService)
	s.RepositoryService = store.NewRepositoryService(l, db, s.ProjectService)
	s.LabelService = store.NewLabelService(l, db)
	s.DeploymentConfigService = store.NewDeploymentConfigService(l, db)
	s.SheetService = store.NewSheetService(l, db)

	s.ActivityManager = server.NewActivityManager(s, storeInstance)

	licenseService, err := enterprise.NewLicenseService(l, profile.dataDir, profile.mode)
	if err != nil {
		return nil, err
	}
	s.LicenseService = licenseService
	s.InitSubscription()

	return s, nil
}

// Run will run the main server.
func (m *Main) Run(ctx context.Context) error {
	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", version, host, m.profile.port))
	if err := m.server.Run(ctx); err != nil {
		return err
	}
	return nil
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	m.l.Info("Trying to stop Bytebase...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m.server != nil {
		m.l.Info("Trying to gracefully shutdown server...")
		m.serverCancel()
		m.server.Shutdown(ctx)
	}

	if m.db != nil {
		m.l.Info("Trying to close database connections...")
		if err := m.db.Close(); err != nil {
			return err
		}
	}

	if m.pgStarted {
		m.l.Info("Trying to shutdown postgresql server...")

		if err := m.pg.Stop(os.Stdout, os.Stderr); err != nil {
			return err
		}
		m.pgStarted = false
	}
	m.l.Info("Bytebase stopped properly.")
	return nil
}

// GetServer returns the server in main.
func (m *Main) GetServer() *server.Server {
	return m.server
}
