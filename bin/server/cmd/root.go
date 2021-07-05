package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/server"
	"github.com/bytebase/bytebase/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// -----------------------------------Global constant BEGIN----------------------------------------
const (
	// Length for the secret used to sign the JWT auth token
	SECRET_LENGTH = 32

	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase
	GREETING_BANNER = `
██████╗ ██╗   ██╗████████╗███████╗██████╗  █████╗ ███████╗███████╗
██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝
██████╔╝ ╚████╔╝    ██║   █████╗  ██████╔╝███████║███████╗█████╗  
██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ██╔══██╗██╔══██║╚════██║██╔══╝  
██████╔╝   ██║      ██║   ███████╗██████╔╝██║  ██║███████║███████╗
╚═════╝    ╚═╝      ╚═╝   ╚══════╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝
                                                                  
%s
___________________________________________________________________________________________

`
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=BYE
	BYE_BANNER = `
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
	host    string
	port    int
	dataDir string
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
		Short: "Bytebase is a database schema change and version control platform",
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
				logConfig.DisableStacktrace = true
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

			start()

			fmt.Print(BYE_BANNER)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "https://localhost", "host where Bytebase is accessed from, must start with http:// or https://. This is used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().IntVar(&port, "port", 80, "port where Bytebase is accessed from. This is also used by Bytebase to create the webhook callback endpoint for VCS integration")
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
	// dns points to where Bytebase stores its own data
	dsn string
	// seedDir points to where to populate the initial data.
	seedDir string
	// force reset seed, true for testing and demo
	forceResetSeed bool
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
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		error := fmt.Errorf("--host %s must start with http:// or https://", host)
		return error
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
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		m.l.Info("SIGINT received.")
		if err := m.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		cancel()
	}()

	// Execute program.
	if err := m.Run(); err != nil {
		m.l.Error(err.Error())
		if err != http.ErrServerClosed {
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
	fmt.Printf("host=%s\n", host)
	fmt.Printf("port=%d\n", port)
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

func initSetting(settingService api.SettingService) (*config, error) {
	result := &config{}
	{
		configCreate := &api.SettingCreate{
			CreatorId:   api.SYSTEM_BOT_ID,
			Name:        "bb.auth.secret",
			Value:       bytebase.RandomString(SECRET_LENGTH),
			Description: "Random string used to sign the JWT auth token.",
		}
		config, err := settingService.CreateSettingIfNotExist(context.Background(), configCreate)
		if err != nil {
			return nil, err
		}
		result.secret = config.Value
	}

	return result, nil
}

func (m *main) Run() error {
	db := store.NewDB(m.l, m.profile.dsn, m.profile.seedDir, m.profile.forceResetSeed, readonly)
	if err := db.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	settingService := store.NewSettingService(m.l, db)
	config, err := initSetting(settingService)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	m.db = db

	server := server.NewServer(m.l, version, host, port, m.profile.mode, config.secret, readonly, demo, debug)
	server.PrincipalService = store.NewPrincipalService(m.l, db, server.CacheService)
	server.MemberService = store.NewMemberService(m.l, db, server.CacheService)
	server.ProjectService = store.NewProjectService(m.l, db, server.CacheService)
	server.ProjectMemberService = store.NewProjectMemberService(m.l, db)
	server.EnvironmentService = store.NewEnvironmentService(m.l, db, server.CacheService)
	server.DataSourceService = store.NewDataSourceService(m.l, db)
	server.DatabaseService = store.NewDatabaseService(m.l, db, server.CacheService)
	server.InstanceService = store.NewInstanceService(m.l, db, server.CacheService, server.DatabaseService, server.DataSourceService)
	server.TableService = store.NewTableService(m.l, db)
	server.IssueService = store.NewIssueService(m.l, db, server.CacheService)
	server.PipelineService = store.NewPipelineService(m.l, db, server.CacheService)
	server.StageService = store.NewStageService(m.l, db)
	server.TaskService = store.NewTaskService(m.l, db, store.NewTaskRunService(m.l, db))
	server.ActivityService = store.NewActivityService(m.l, db)
	server.BookmarkService = store.NewBookmarkService(m.l, db)
	server.VCSService = store.NewVCSService(m.l, db)
	server.RepositoryService = store.NewRepositoryService(m.l, db, server.ProjectService)

	m.server = server

	fmt.Printf(GREETING_BANNER, fmt.Sprintf("Version %s has started at %s:%d", version, host, port))

	if err := server.Run(); err != nil {
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
