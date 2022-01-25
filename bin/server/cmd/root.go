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

	logger *zap.Logger

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(cmd *cobra.Command, args []string) {
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
			myLogger, err := logConfig.Build()
			if err != nil {
				panic(fmt.Errorf("failed to create logger. %w", err))
			}
			logger = myLogger
			defer logger.Sync()

			if err := preStart(); err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}

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
type profile struct {
	// mode can be "release" or "dev"
	mode string
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

type main struct {
	profile *profile

	l *zap.Logger

	server *server.Server

	db *store.DB
}

func preStart() error {
	if !common.HasPrefixes(host, "http://", "https://") {
		return fmt.Errorf("--host %s must start with http:// or https://", host)
	}

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

func start() {
	m := newMain()

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

func newMain() *main {
	activeProfile := activeProfile(dataDir, demo)

	fmt.Println("-----Config BEGIN-----")
	fmt.Printf("mode=%s\n", activeProfile.mode)
	fmt.Printf("server=%s:%d\n", host, port)
	fmt.Printf("frontend=%s:%d\n", frontendHost, frontendPort)
	fmt.Printf("dsn=%s\n", activeProfile.dsn)
	fmt.Printf("seedDir=%s\n", activeProfile.seedDir)
	fmt.Printf("readonly=%t\n", readonly)
	fmt.Printf("demo=%t\n", demo)
	fmt.Printf("debug=%t\n", debug)
	fmt.Println("-----Config END-------")

	return &main{
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

func (m *main) Run(ctx context.Context) error {
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

	s := server.NewServer(m.l, version, host, port, frontendHost, frontendPort, m.profile.mode, dataDir, m.profile.backupRunnerInterval, config.secret, readonly, demo, debug)
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

	s.ActivityManager = server.NewActivityManager(s, s.ActivityService)

	licenseService, err := enterprise.NewLicenseService(m.l, dataDir)
	if err != nil {
		return err
	}
	s.LicenseService = licenseService

	m.server = s

	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", version, host, port))

	if err := s.Run(); err != nil {
		return err
	}

	return nil
}

// Close gracefully stops the program.
func (m *main) Close() error {
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
