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

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterprise "github.com/bytebase/bytebase/enterprise/service"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	// Import sqlite3 driver.
	_ "github.com/mattn/go-sqlite3"

	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/plugin/db/clickhouse"
	// Register mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	_ "github.com/lib/pq"

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

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(cmd *cobra.Command, args []string) {
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
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------

// Profile is the configuration to start main server.
type Profile struct {
	// mode can be "release" or "dev"
	mode string
	// port is the binding port for server.
	port int
	// dataDir is the directory stores the data including Bytebase's own database, backups, etc.
	dataDir string
	// dsn points to where Bytebase stores its own data
	dsn string
	// seedDir points to where to populate the initial data.
	seedDir string
	// force reset seed, true for testing and demo
	forceResetSeed bool
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

	l *zap.Logger

	server *server.Server

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
func GetLogger() (*zap.Logger, error) {
	logConfig := zap.NewProductionConfig()
	// Always set encoding to "console" for now since we do not redirect to file.
	logConfig.Encoding = "console"
	// "console" encoding needs to use the corresponding development encoder config.
	logConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	if debug {
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return logConfig.Build()
}

func start() {
	logger, err := GetLogger()
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

	activeProfile := activeProfile(dataDir, port, demo)
	m := NewMain(activeProfile, logger)

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
func NewMain(activeProfile Profile, logger *zap.Logger) *Main {
	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", activeProfile.mode)
	fmt.Printf("server=%s:%d\n", host, activeProfile.port)
	fmt.Printf("frontend=%s:%d\n", frontendHost, frontendPort)
	fmt.Printf("dsn=%s\n", activeProfile.dsn)
	fmt.Printf("seedDir=%s\n", activeProfile.seedDir)
	fmt.Printf("readonly=%t\n", readonly)
	fmt.Printf("demo=%t\n", demo)
	fmt.Printf("debug=%t\n", debug)
	fmt.Println("-----Config END-------")

	return &Main{
		profile: &activeProfile,
		l:       logger,
	}
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

// Run will run the main server.
func (m *Main) Run(ctx context.Context) error {
	db := store.NewDB(m.l, m.profile.dsn, m.profile.seedDir, m.profile.forceResetSeed, readonly, version)
	if err := db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	settingService := store.NewSettingService(m.l, db)
	config, err := initSetting(ctx, settingService)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	m.db = db

	s := server.NewServer(m.l, version, host, m.profile.port, frontendHost, frontendPort, m.profile.mode, m.profile.dataDir, m.profile.backupRunnerInterval, config.secret, readonly, demo, debug)
	s.SettingService = settingService
	s.PrincipalService = store.NewPrincipalService(m.l, db, s.CacheService)
	s.MemberService = store.NewMemberService(m.l, db, s.CacheService)
	s.PolicyService = store.NewPolicyService(m.l, db, s.CacheService)
	s.ProjectService = store.NewProjectService(m.l, db, s.CacheService)
	s.ProjectMemberService = store.NewProjectMemberService(m.l, db)
	s.ProjectWebhookService = store.NewProjectWebhookService(m.l, db)
	s.EnvironmentService = store.NewEnvironmentService(m.l, db, s.CacheService)
	s.DataSourceService = store.NewDataSourceService(m.l, db)
	s.BackupService = store.NewBackupService(m.l, db, s.PolicyService)
	s.DatabaseService = store.NewDatabaseService(m.l, db, s.CacheService, s.PolicyService, s.BackupService)
	s.InstanceService = store.NewInstanceService(m.l, db, s.CacheService, s.DatabaseService, s.DataSourceService)
	s.InstanceUserService = store.NewInstanceUserService(m.l, db)
	s.TableService = store.NewTableService(m.l, db)
	s.ColumnService = store.NewColumnService(m.l, db)
	s.ViewService = store.NewViewService(m.l, db)
	s.IndexService = store.NewIndexService(m.l, db)
	s.IssueService = store.NewIssueService(m.l, db, s.CacheService)
	s.IssueSubscriberService = store.NewIssueSubscriberService(m.l, db)
	s.PipelineService = store.NewPipelineService(m.l, db, s.CacheService)
	s.StageService = store.NewStageService(m.l, db)
	s.TaskCheckRunService = store.NewTaskCheckRunService(m.l, db)
	s.TaskService = store.NewTaskService(m.l, db, store.NewTaskRunService(m.l, db), s.TaskCheckRunService)
	s.ActivityService = store.NewActivityService(m.l, db)
	s.InboxService = store.NewInboxService(m.l, db, s.ActivityService)
	s.BookmarkService = store.NewBookmarkService(m.l, db)
	s.VCSService = store.NewVCSService(m.l, db)
	s.RepositoryService = store.NewRepositoryService(m.l, db, s.ProjectService)
	s.AnomalyService = store.NewAnomalyService(m.l, db)
	s.LabelService = store.NewLabelService(m.l, db)
	s.DeploymentConfigService = store.NewDeploymentConfigService(m.l, db)
	s.SavedQueryService = store.NewSavedQueryService(m.l, db)
	s.SheetService = store.NewSheetService(m.l, db)

	s.ActivityManager = server.NewActivityManager(s, s.ActivityService)

	licenseService, err := enterprise.NewLicenseService(m.l, m.profile.dataDir, m.profile.mode)
	if err != nil {
		return err
	}
	s.LicenseService = licenseService
	s.InitSubscription()

	m.server = s

	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", version, host, m.profile.port))

	if err := s.Run(); err != nil {
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
		m.server.Shutdown(ctx)
	}

	if m.db != nil {
		m.l.Info("Trying to close database connections...")
		if err := m.db.Close(); err != nil {
			return err
		}
	}
	m.l.Info("Bytebase stopped properly.")
	return nil
}

// GetServer returns the server in main.
func (m *Main) GetServer() *server.Server {
	return m.server
}
