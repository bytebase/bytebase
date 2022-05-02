package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bytebase/bytebase/metadb"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterprise "github.com/bytebase/bytebase/enterprise/service"
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

type FlagConf struct {
	// Used for bytebase command line config
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
}

// -----------------------------------Command Line Config BEGIN------------------------------------
var (
	flagConf FlagConf
	rootCmd  = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(_ *cobra.Command, _ []string) {
			if flagConf.frontendHost == "" {
				flagConf.frontendHost = flagConf.host
			}
			if flagConf.frontendPort == 0 {
				flagConf.frontendPort = flagConf.port
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
	rootCmd.PersistentFlags().StringVar(&flagConf.host, "host", "http://localhost", "host where Bytebase backend is accessed from, must start with http:// or https://. This is used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().IntVar(&flagConf.port, "port", 80, "port where Bytebase backend is accessed from. This is also used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().StringVar(&flagConf.frontendHost, "frontend-host", "", "host where Bytebase frontend is accessed from, must start with http:// or https://. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --host")
	rootCmd.PersistentFlags().IntVar(&flagConf.frontendPort, "frontend-port", 0, "port where Bytebase frontend is accessed from. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --port")
	rootCmd.PersistentFlags().StringVar(&flagConf.dataDir, "data", ".", "directory where Bytebase stores data. If relative path is supplied, then the path is relative to the directory where bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&flagConf.readonly, "readonly", false, "whether to run in read-only mode")
	rootCmd.PersistentFlags().BoolVar(&flagConf.demo, "demo", false, "whether to run using demo data")
	rootCmd.PersistentFlags().BoolVar(&flagConf.debug, "debug", false, "whether to enable debug level logging")
	// TODO(tianzhou): this needs more bake time. There are couple blocking issues:
	// 1. Currently, we will create a separate bytebase database to store the migration_history table, we need to put it inside the specified database here.
	// 2. We need to move the logic of creating bytebase metadata db logic outside. Because with --pg option, the db has already been created.
	rootCmd.PersistentFlags().StringVar(&flagConf.pgURL, "pg", "", "optional external PostgreSQL instance connection url(must provide dbname); for example postgresql://user:secret@masterhost:5432/dbname?sslrootcert=cert")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------

// retrieved via the SettingService upon startup
type config struct {
	// secret used to sign the JWT auth token
	secret string
}

// Main is the main server for Bytebase.
type Main struct {
	profile *server.Profile

	l   *zap.Logger
	lvl *zap.AtomicLevel

	server *server.Server
	// serverCancel cancels any runner on the server.
	// Then the runnerWG waits for all runners to finish before we shutdown the server.
	// Otherwise, we will get database is closed error from runner when we shutdown the server.
	serverCancel context.CancelFunc

	metadataDB *metadb.MetadataDB
	// db is a connection to the database storing Bytebase data.
	db *store.DB
}

func useEmbedDB() bool {
	return len(flagConf.pgURL) == 0
}

func checkDataDir() error {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(flagConf.dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + flagConf.dataDir)
		if err != nil {
			return err
		}
		flagConf.dataDir = absDir
	}

	// Trim trailing / in case user supplies
	flagConf.dataDir = strings.TrimRight(flagConf.dataDir, "/")

	if _, err := os.Stat(flagConf.dataDir); err != nil {
		error := fmt.Errorf("unable to access --data %s, %w", flagConf.dataDir, err)
		return error
	}

	return nil
}

// GetLogger will return a logger.
func GetLogger() (*zap.Logger, *zap.AtomicLevel, error) {
	atom := zap.NewAtomicLevelAt(zap.InfoLevel)
	if flagConf.debug {
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

	// check flags
	if !common.HasPrefixes(flagConf.host, "http://", "https://") {
		logger.Error(fmt.Sprintf("--host %s must start with http:// or https://", flagConf.host))
		return
	}
	if err := checkDataDir(); err != nil {
		logger.Error(err.Error())
		return
	}

	// We use port+1 for datastore port.
	datastorePort := flagConf.port + 1
	activeProfile := activeProfile(flagConf.dataDir, flagConf.port, datastorePort, flagConf.demo)
	m, err := NewMain(activeProfile, logger)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	m.lvl = level

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
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
func NewMain(activeProfile server.Profile, logger *zap.Logger) (*Main, error) {
	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", activeProfile.Mode)
	fmt.Printf("server=%s:%d\n", flagConf.host, activeProfile.Port)
	fmt.Printf("datastore=%s:%d\n", flagConf.host, activeProfile.DatastorePort)
	fmt.Printf("frontend=%s:%d\n", flagConf.frontendHost, flagConf.frontendPort)
	fmt.Printf("demoDataDir=%s\n", activeProfile.DemoDataDir)
	fmt.Printf("readonly=%t\n", flagConf.readonly)
	fmt.Printf("demo=%t\n", flagConf.demo)
	fmt.Printf("debug=%t\n", flagConf.debug)
	fmt.Println("-----Config END-------")

	var metadataDB *metadb.MetadataDB
	var err error
	if useEmbedDB() {
		metadataDB, err = metadb.NewMetadataDBWithEmbedPg(&activeProfile, logger)
	} else {
		metadataDB, err = metadb.NewMetadataDBWithExternalPg(&activeProfile, logger, flagConf.pgURL)
	}
	if err != nil {
		return nil, err
	}

	return &Main{
		profile:    &activeProfile,
		l:          logger,
		metadataDB: metadataDB,
	}, nil
}

func initSetting(ctx context.Context, store *store.Store) (*config, error) {
	result := &config{}
	{
		configCreate := &api.SettingCreate{
			CreatorID:   api.SystemBotID,
			Name:        api.SettingAuthSecret,
			Value:       common.RandomString(secretLength),
			Description: "Random string used to sign the JWT auth token.",
		}
		config, err := store.CreateSettingIfNotExist(ctx, configCreate)
		if err != nil {
			return nil, err
		}
		result.secret = config.Value
	}

	return result, nil
}

func initBranding(ctx context.Context, store *store.Store) error {
	configCreate := &api.SettingCreate{
		CreatorID:   api.SystemBotID,
		Name:        api.SettingBrandingLogo,
		Value:       "",
		Description: "The branding logo image in base64 string format.",
	}
	_, err := store.CreateSettingIfNotExist(ctx, configCreate)
	if err != nil {
		return err
	}

	return nil
}

// Run will run the main server.
func (m *Main) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.serverCancel = cancel

	db, err := m.metadataDB.Connect(flagConf.readonly, version)
	if err != nil {
		return fmt.Errorf("cannot new db: %w", err)
	}
	if err := db.Open(ctx); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}
	m.db = db

	cacheService := server.NewCacheService(m.l)
	storeInstance := store.New(m.l, db, cacheService)

	config, err := initSetting(ctx, storeInstance)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	err = initBranding(ctx, storeInstance)
	if err != nil {
		return fmt.Errorf("failed to init branding: %w", err)
	}

	s := server.NewServer(m.l, storeInstance, m.lvl, version, flagConf.host, m.profile.Port, flagConf.frontendHost, flagConf.frontendPort, m.profile.DatastorePort, m.profile.Mode, m.profile.DataDir, m.profile.BackupRunnerInterval, config.secret, flagConf.readonly, flagConf.demo, flagConf.debug)

	s.ActivityManager = server.NewActivityManager(s, storeInstance)

	licenseService, err := enterprise.NewLicenseService(m.l, m.profile.DataDir, m.profile.Mode)
	if err != nil {
		return err
	}
	s.LicenseService = licenseService
	s.InitSubscription()

	m.server = s

	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", version, flagConf.host, m.profile.Port))

	if err := s.Run(ctx); err != nil {
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

	if err := m.metadataDB.Close(); err != nil {
		return err
	}

	m.l.Info("Bytebase stopped properly.")
	return nil
}

// GetServer returns the server in main.
func (m *Main) GetServer() *server.Server {
	return m.server
}
